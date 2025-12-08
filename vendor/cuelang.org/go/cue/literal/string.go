// file: vendor/cuelang.org/go/cue/literal/string.go
// Copyright 2019 CUE Authors
// ... (License) ...

package literal

import (
	// "errors" // REMOVED: No longer needed for static init
	"strings"
	"unicode"
	"unicode/utf8"
)

// STUBBED: Use a custom string type for errors.
// This allows us to use 'const', which requires NO static initialization at runtime.
type litErr string

func (e litErr) Error() string { return string(e) }

const (
	errSyntax            = litErr("invalid syntax")
	errInvalidWhitespace = litErr("invalid string: invalid whitespace")
	errMissingNewline    = litErr("invalid string: opening quote of multiline string must be followed by newline")
	errUnmatchedQuote    = litErr("invalid string: unmatched quote")
	errSurrogate         = litErr("unmatched surrogate pair")
	errEscapedLastNewline = litErr("last newline of multiline string cannot be escaped")
)

// Unquote interprets s as a single- or double-quoted, single- or multi-line...
func Unquote(s string) (string, error) {
	info, nStart, _, err := ParseQuotes(s, s)
	if err != nil {
		return "", err
	}
	s = s[nStart:]
	return info.Unquote(s)
}

// QuoteInfo describes the type of quotes used for a string.
type QuoteInfo struct {
	quote      string
	whitespace string
	numHash    int
	multiline  bool
	char       byte
	numChar    byte
}

func (q QuoteInfo) IsDouble() bool {
	return q.char == '"'
}

func (q QuoteInfo) IsMulti() bool {
	return q.multiline
}

func (q QuoteInfo) Whitespace() string {
	return q.whitespace
}

func ParseQuotes(start, end string) (q QuoteInfo, nStart, nEnd int, err error) {
	for i, c := range start {
		if c != '#' {
			break
		}
		q.numHash = i + 1
	}
	s := start[q.numHash:]
	switch s[0] {
	case '"', '\'':
		q.char = s[0]
		if len(s) > 3 && s[1] == s[0] && s[2] == s[0] {
			switch s[3] {
			case '\n':
				q.quote = start[:3+q.numHash]
			case '\r':
				if len(s) > 4 && s[4] == '\n' {
					q.quote = start[:4+q.numHash]
					break
				}
				fallthrough
			default:
				return q, 0, 0, errMissingNewline
			}
			q.multiline = true
			q.numChar = 3
			nStart = len(q.quote) + 1
		} else {
			q.quote = start[:1+q.numHash]
			q.numChar = 1
			nStart = len(q.quote)
		}
	default:
		return q, 0, 0, errSyntax
	}
	quote := start[:int(q.numChar)+q.numHash]
	for i := 0; i < len(quote); i++ {
		if j := len(end) - i - 1; j < 0 || quote[i] != end[j] {
			return q, 0, 0, errUnmatchedQuote
		}
	}
	if q.multiline {
		i := len(end) - len(quote)
		for i > 0 {
			r, size := utf8.DecodeLastRuneInString(end[:i])
			if r == '\n' || !unicode.IsSpace(r) {
				break
			}
			i -= size
		}
		q.whitespace = end[i : len(end)-len(quote)]

		if len(start) > nStart && start[nStart] != '\n' {
			if !strings.HasPrefix(start[nStart:], q.whitespace) {
				return q, 0, 0, errInvalidWhitespace
			}
			nStart += len(q.whitespace)
		}
	}

	return q, nStart, int(q.numChar) + q.numHash, nil
}

func (q QuoteInfo) Unquote(s string) (string, error) {
	if len(s) > 0 && !q.multiline {
		if strings.ContainsAny(s, "\n\r") {
			return "", errSyntax
		}

		if s[len(s)-1] == q.char && q.numHash == 0 {
			if s := s[:len(s)-1]; isSimple(s, rune(q.char)) {
				return s, nil
			}
		}
	}

	buf := make([]byte, 0, 3*len(s)/2)
	stripNL := false
	wasEscapedNewline := false
	for len(s) > 0 {
		switch s[0] {
		case '\r':
			s = s[1:]
			wasEscapedNewline = false
			continue
		case '\n':
			var err error
			s, err = skipWhitespaceAfterNewline(s[1:], q)
			if err != nil {
				return "", err
			}
			stripNL = true
			wasEscapedNewline = false
			buf = append(buf, '\n')
			continue
		}
		c, multibyte, ss, err := unquoteChar(s, q)
		if surHigh <= c && c < surEnd {
			if c >= surLow {
				return "", errSurrogate
			}
			var cl rune
			cl, _, ss, err = unquoteChar(ss, q)
			if cl < surLow || surEnd <= cl {
				return "", errSurrogate
			}
			c = 0x10000 + (c-surHigh)*0x400 + (cl - surLow)
		}

		if err != nil {
			return "", err
		}

		s = ss
		if c < 0 {
			switch c {
			case escapedNewline:
				var err error
				s, err = skipWhitespaceAfterNewline(s, q)
				if err != nil {
					return "", err
				}
				wasEscapedNewline = true
				continue
			case terminatedByQuote:
				if wasEscapedNewline {
					return "", errEscapedLastNewline
				}
				if stripNL {
					buf = buf[:len(buf)-1]
				}
			case terminatedByExpr:
			default:
				panic("unreachable")
			}
			return string(buf), nil
		}
		stripNL = false
		wasEscapedNewline = false
		if !multibyte {
			buf = append(buf, byte(c))
		} else {
			buf = utf8.AppendRune(buf, c)
		}
	}
	return "", errUnmatchedQuote
}

func skipWhitespaceAfterNewline(s string, q QuoteInfo) (string, error) {
	switch {
	case !q.multiline:
		fallthrough
	default:
		return "", errInvalidWhitespace
	case strings.HasPrefix(s, q.whitespace):
		s = s[len(q.whitespace):]
	case strings.HasPrefix(s, "\n"):
	case strings.HasPrefix(s, "\r\n"):
	}
	return s, nil
}

const (
	surHigh = 0xD800
	surLow  = 0xDC00
	surEnd  = 0xE000
)

func isSimple(s string, quote rune) bool {
	for _, r := range s {
		if r == quote || r == '\\' {
			return false
		}
		if surHigh <= r && r < surEnd {
			return false
		}
	}
	return true
}

const (
	terminatedByQuote = rune(-1)
	terminatedByExpr  = rune(-2)
	escapedNewline    = rune(-3)
)

func unquoteChar(s string, info QuoteInfo) (value rune, multibyte bool, tail string, err error) {
	switch c := s[0]; {
	case c == info.char && info.char != 0:
		for i := 1; byte(i) < info.numChar; i++ {
			if i >= len(s) || s[i] != info.char {
				return rune(info.char), false, s[1:], nil
			}
		}
		for i := 0; i < info.numHash; i++ {
			if i+int(info.numChar) >= len(s) || s[i+int(info.numChar)] != '#' {
				return rune(info.char), false, s[1:], nil
			}
		}
		if ln := int(info.numChar) + info.numHash; len(s) != ln {
			return 0, false, s[ln:], errSyntax
		}
		return terminatedByQuote, false, "", nil
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}

	if len(s) <= 1+info.numHash {
		return '\\', false, s[1:], nil
	}
	for i := 1; i <= info.numHash && i < len(s); i++ {
		if s[i] != '#' {
			return '\\', false, s[1:], nil
		}
	}

	c := s[1+info.numHash]
	s = s[2+info.numHash:]

	switch c {
	case 'a': value = '\a'
	case 'b': value = '\b'
	case 'f': value = '\f'
	case 'n': value = '\n'
	case 'r': value = '\r'
	case 't': value = '\t'
	case 'v': value = '\v'
	case '/': value = '/'
	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x': n = 2
		case 'u': n = 4
		case 'U': n = 8
		}
		var v rune
		if len(s) < n {
			err = errSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = errSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c == 'x' {
			if info.char == '"' {
				err = errSyntax
				return
			}
			value = v
			break
		}
		if v > utf8.MaxRune {
			err = errSyntax
			return
		}
		value = v
		multibyte = true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		if info.char == '"' {
			err = errSyntax
			return
		}
		v := rune(c) - '0'
		if len(s) < 2 {
			err = errSyntax
			return
		}
		for j := 0; j < 2; j++ {
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				err = errSyntax
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = errSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		if c != info.char {
			err = errSyntax
			return
		}
		value = rune(c)
	case '(':
		if s != "" {
			return 0, false, s, errSyntax
		}
		value = terminatedByExpr
	case '\r':
		if len(s) == 0 || s[0] != '\n' {
			err = errSyntax
			return
		}
		s = s[1:]
		value = escapedNewline
	case '\n':
		value = escapedNewline
	default:
		err = errSyntax
		return
	}
	tail = s
	return
}

func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}

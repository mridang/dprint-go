// file: vendor/cuelang.org/go/cue/literal/num.go
package literal

import (
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
)

// NumInfo contains information about a parsed numbers.
type NumInfo struct {
	pos token.Pos
	src string
	p   int
	ch  byte
	buf []byte

	mul     Multiplier
	base    byte
	neg     bool
	UseSep  bool
	isFloat bool
	err     error
}

func (p *NumInfo) String() string {
	return string(p.buf)
}

func (p *NumInfo) Multiplier() Multiplier {
	return p.mul
}

func (p *NumInfo) IsInt() bool {
	return !p.isFloat
}

func ParseNum(s string, n *NumInfo) error {
	*n = NumInfo{pos: n.pos, src: s, buf: n.buf[:0]}
	if !n.next() {
		return n.errorf("invalid number %q", s)
	}
	switch n.ch {
	case '-':
		n.neg = true
		n.buf = append(n.buf, '-')
		n.next()
	case '+':
		n.next()
	}
	seenDecimalPoint := false
	if n.ch == '.' {
		n.next()
		seenDecimalPoint = true
	}
	err := n.scanNumber(seenDecimalPoint)
	if err != nil {
		return err
	}
	if n.err != nil {
		return n.err
	}
	if n.p < len(n.src) {
		return n.errorf("invalid number %q", s)
	}
	if len(n.buf) == 0 {
		n.buf = append(n.buf, '0')
	}
	return nil
}

func (p *NumInfo) errorf(format string, args ...interface{}) error {
	return errors.Newf(p.pos, format, args...)
}

type Multiplier byte

const (
	mul1 Multiplier = 1 + iota
	mul2
	mul3
	mul4
	mul5
	mul6
	mul7
	mul8

	mulBin = 0x10
	mulDec = 0x20

	K = mulDec | mul1
	M = mulDec | mul2
	G = mulDec | mul3
	T = mulDec | mul4
	P = mulDec | mul5
	E = mulDec | mul6
	Z = mulDec | mul7
	Y = mulDec | mul8

	Ki = mulBin | mul1
	Mi = mulBin | mul2
	Gi = mulBin | mul3
	Ti = mulBin | mul4
	Pi = mulBin | mul5
	Ei = mulBin | mul6
	Zi = mulBin | mul7
	Yi = mulBin | mul8
)

func (p *NumInfo) next() bool {
	if p.p >= len(p.src) {
		p.ch = 0
		return false
	}
	p.ch = p.src[p.p]
	p.p++
	if p.ch == '.' {
		if len(p.buf) == 0 {
			p.buf = append(p.buf, '0')
		}
		p.buf = append(p.buf, '.')
	}
	return true
}

func (p *NumInfo) digitVal(ch byte) (d int) {
	switch {
	case '0' <= ch && ch <= '9':
		d = int(ch - '0')
	case ch == '_':
		p.UseSep = true
		return 0
	case 'a' <= ch && ch <= 'f':
		d = int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		d = int(ch - 'A' + 10)
	default:
		return 16
	}
	return d
}

func (p *NumInfo) scanMantissa(base int) bool {
	hasDigit := false
	var last byte
	for p.digitVal(p.ch) < base {
		if p.ch != '_' {
			p.buf = append(p.buf, p.ch)
			hasDigit = true
		}
		last = p.ch
		p.next()
	}
	if last == '_' {
		p.err = p.errorf("illegal '_' in number")
	}
	return hasDigit
}

func (p *NumInfo) scanNumber(seenDecimalPoint bool) error {
	p.base = 10

	if seenDecimalPoint {
		p.isFloat = true
		if !p.scanMantissa(10) {
			return p.errorf("illegal fraction %q", p.src)
		}
		goto exponent
	}

	if p.ch == '0' {
		p.next()
		switch p.ch {
		case 'x', 'X':
			p.base = 16
			p.next()
			if !p.scanMantissa(16) {
				return p.errorf("illegal hexadecimal number %q", p.src)
			}
		case 'b':
			p.base = 2
			p.next()
			if !p.scanMantissa(2) {
				return p.errorf("illegal binary number %q", p.src)
			}
		case 'o':
			p.base = 8
			p.next()
			if !p.scanMantissa(8) {
				return p.errorf("illegal octal number %q", p.src)
			}
		default:
			p.scanMantissa(8)
			if p.ch == '8' || p.ch == '9' {
				p.scanMantissa(10)
				if p.ch != '.' && p.ch != 'e' && p.ch != 'E' {
					return p.errorf("illegal integer number %q", p.src)
				}
			}
			switch p.ch {
			case 'e', 'E':
				if len(p.buf) == 0 {
					p.buf = append(p.buf, '0')
				}
				fallthrough
			case '.':
				goto fraction
			}
			if len(p.buf) > 0 {
				p.base = 8
			}
		}
		goto exit
	}

	if !p.scanMantissa(10) {
		return p.errorf("illegal number start %q", p.src)
	}

fraction:
	if p.ch == '.' {
		p.isFloat = true
		p.next()
		p.scanMantissa(10)
	}

exponent:
	switch p.ch {
	case 'K', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y':
		// REPLACEMENT: Use switch instead of map lookup to avoid static init
		p.mul = charToMultiplier(p.ch)
		p.next()
		if p.ch == 'i' {
			p.mul |= mulBin
			p.next()
		} else {
			p.mul |= mulDec
		}
		p.isFloat = false
		return nil

	case 'e':
		// lowercase e is treated as exponent start, handled below
		fallthrough

	// Note: 'E' is handled in the case above because 'E' (Exa) clashes with E notation.
	// In the original code 'E' was in the map.
	// The original logic checked the map first.
	// If p.ch is 'E', it enters the case above.
	// However, if it's meant to be an exponent, we need to be careful.
	// Logic: CUE multipliers are usually suffix only.
	// If we are here, we might be parsing an exponent.
	// The original code check: case 'K', 'M', ...: p.mul = map[...]
	// BUT 'e' and 'E' were also checked in the exponent block below in original.
	// Actually, looking at original code: 'E' is in charToMul (Exa).
	// If it matches a multiplier, it returns early.
	// If it hits case 'e', 'E' below, it parses exponent.
	// This implies an ambiguity in CUE: 1E can be 1 Exa or 1E(incomplete).
	// For formatting purposes, we stick to the original logic precedence.

	// Wait, strictly speaking, standard float parsing handles 'E' as exponent.
	// The original code had 'E' in the map for Exa.
	// Let's stick to the structure:

	default:
		// if it was 'e', we fall through to here
		if p.ch == 'e' || p.ch == 'E' {
			p.isFloat = true
			p.next()
			p.buf = append(p.buf, 'e') // normalize to e
			if p.ch == '-' || p.ch == '+' {
				p.buf = append(p.buf, p.ch)
				p.next()
			}
			if !p.scanMantissa(10) {
				return p.errorf("illegal exponent %q", p.src)
			}
		}
	}

exit:
	return nil
}

// Replacement for the global map charToMul to avoid static initialization
func charToMultiplier(c byte) Multiplier {
	switch c {
	case 'K': return mul1
	case 'M': return mul2
	case 'G': return mul3
	case 'T': return mul4
	case 'P': return mul5
	case 'E': return mul6
	case 'Z': return mul7
	case 'Y': return mul8
	}
	return 0
}

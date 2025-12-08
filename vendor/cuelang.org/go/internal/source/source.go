// Copyright 2019 CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package source contains utility functions that standardize reading source
// bytes across cue packages.
package source

import (
	"bytes"
	"fmt"
)

// ReadAll loads the source bytes for the given arguments. If src != nil,
// ReadAll converts src to a []byte if possible; otherwise it returns an
// error. If src == nil, ReadAll returns the result of reading the file
// specified by filename.
func ReadAll(filename string, src any) ([]byte, error) {
	if src != nil {
		switch src := src.(type) {
		case string:
			return []byte(src), nil
		case []byte:
			return src, nil
		case *bytes.Buffer:
			return src.Bytes(), nil
		}
		return nil, fmt.Errorf("invalid source type %T", src)
	}
	return nil, fmt.Errorf("ff")
}


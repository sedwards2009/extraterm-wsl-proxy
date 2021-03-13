/*
 * Copyright 2020 Simon Edwards <simon@simonzone.com>
 *
 * This source code is licensed under the MIT license which is detailed in the LICENSE.txt file.
 */
package utf8sanitize

import (
	"bytes"
	"unicode/utf8"
)

type Utf8Sanitizer struct {
	remainder []byte
}

const utf8MaxEncodingLength = 4

// Bytes to clean UTF8 string sanitizer.
//
// This aims to prevent invalid UTF8 strings being formed due to
// codepoints crossing byte buffer boundaries in a stream of bytes.
func NewUtf8Sanitizer() *Utf8Sanitizer {
	return new(Utf8Sanitizer)
}

func (this *Utf8Sanitizer) Sanitize(newInput []byte) string {
	var input []byte
	position := 0

	if len(this.remainder) != 0 {
		input = bytes.Join([][]byte{this.remainder, newInput}, []byte{})
	} else {
		input = newInput
	}

	for position < len(input) {
		r, size := utf8.DecodeRune(input[position:])
		if r == utf8.RuneError {
			if position < len(input)-utf8MaxEncodingLength {
				// Accept the corrupt data and try to go past it.
				size = 1
			} else {
				// Split the buffer just before the bad encoding in case this
				// is a multi-byte encoding which has been cut in half.
				this.remainder = input[position:]
				return string(input[:position])
			}
		}
		position += size
	}

	return string(input)
}

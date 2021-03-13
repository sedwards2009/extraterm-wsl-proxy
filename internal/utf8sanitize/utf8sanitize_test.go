/*
 * Copyright 2020 Simon Edwards <simon@simonzone.com>
 *
 * This source code is licensed under the MIT license which is detailed in the LICENSE.txt file.
 */
package utf8sanitize

import (
	"testing"
)

func TestSimpleDeocde(t *testing.T) {
	sanitizer := NewUtf8Sanitizer()
	cleanString := sanitizer.Sanitize([]byte("Hello world!"))
	if cleanString != "Hello world!" {
		t.Fail()
	}
}

func Test2PartDeocde(t *testing.T) {
	sanitizer := NewUtf8Sanitizer()
	if cleanString := sanitizer.Sanitize([]byte("Hello")); cleanString != "Hello" {
		t.Fail()
	}

	if cleanString := sanitizer.Sanitize([]byte("world!")); cleanString != "world!" {
		t.Fail()
	}
}
func TestSplitCodePoint(t *testing.T) {
	source := []byte("Hello, 世")
	sanitizer := NewUtf8Sanitizer()
	if cleanString := sanitizer.Sanitize(source[:len(source)-1]); cleanString != "Hello, " {
		t.Fail()
	}

	if cleanString := sanitizer.Sanitize(source[len(source)-1:]); cleanString != "世" {
		t.Fail()
	}
}

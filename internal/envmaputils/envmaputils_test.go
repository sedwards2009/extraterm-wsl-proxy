/*
 * Copyright 2020 Simon Edwards <simon@simonzone.com>
 *
 * This source code is licensed under the MIT license which is detailed in the LICENSE.txt file.
 */
package envmaputils

import (
	"sort"
	"testing"
)

func TestKeyValueMapToArray(t *testing.T) {
	testMap := map[string]string{
		"foo":  "bar",
		"smeg": "it"}

	result := KeyValueMapToArray(&testMap)
	sort.Strings(*result)
	if (*result)[0] != "foo=bar" {
		t.Fail()
	}
	if (*result)[0] != "smeg=it" {
		t.Fail()
	}
}

func TestKeyValueArrayToMap(t *testing.T) {
	testEnv := []string{
		"foo=bar", "smeg=it=all"}
	result := KeyValueArrayToMap(testEnv)
	if (*result)["foo"] != "bar" {
		t.Fail()
	}

	if (*result)["smeg"] != "it=all" {
		t.Fail()
	}
}

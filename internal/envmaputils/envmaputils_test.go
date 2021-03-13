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
		t.Logf("foo=bar test failed")
	}
	if (*result)[1] != "smeg=it" {
		t.Logf("smeg=it test failed")
	}
}

func TestKeyValueArrayToMap(t *testing.T) {
	testEnv := []string{
		"foo=bar", "smeg=it=all"}
	result := KeyValueArrayToMap(testEnv)
	if (*result)["foo"] != "bar" {
		t.Logf("['foo'] != 'bar' failed")
	}

	if (*result)["smeg"] != "it=all" {
		t.Logf("['smeg'] != 'it=all' failed")
	}
}

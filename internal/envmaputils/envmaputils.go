/*
 * Copyright 2020 Simon Edwards <simon@simonzone.com>
 *
 * This source code is licensed under the MIT license which is detailed in the LICENSE.txt file.
 */
package envmaputils

import "strings"

func KeyValueArrayToMap(array []string) *map[string]string {
	result := map[string]string{}
	for _, entry := range array {
		parts := strings.Split(entry, "=")
		key := parts[0]
		value := strings.Join(parts[1:], "=")
		result[key] = value
	}
	return &result
}

func KeyValueMapToArray(envMap *map[string]string) *[]string {
	result := make([]string, len(*envMap))
	result = result[:0]
	for key, value := range *envMap {
		result = append(result, key+"="+value)
	}
	return &result
}

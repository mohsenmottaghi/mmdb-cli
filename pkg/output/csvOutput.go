/*
Copyright 2024 The InfraZ Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package output

import (
	"encoding/csv"
	"encoding/json"
	"maps"
	"os"
	"sort"
	"strconv"
)

func CsvOutput(data []byte, options OutputOptions) error {
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	switch v := parsed.(type) {
	case []any:
		return csvWriteArray(w, v)
	case map[string]any:
		rows, keys := expandObjectToRows(v, "", false)
		return csvWriteRows(w, rows, keys)
	default:
		return w.Write([]string{scalarToString(parsed)})
	}
}

func csvWriteArray(w *csv.Writer, arr []any) error {
	var rows []map[string]string
	var columnOrder []string

	for _, item := range arr {
		itemRows, itemKeys := expandToRows(item, "")
		rows = append(rows, itemRows...)
		columnOrder = mergeKeyOrder(columnOrder, itemKeys)
	}

	return csvWriteRows(w, rows, columnOrder)
}

func csvWriteRows(w *csv.Writer, rows []map[string]string, columnOrder []string) error {
	allKeySet := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			allKeySet[key] = struct{}{}
		}
	}

	// Build headers: columnOrder first, then any keys not yet covered.
	seen := make(map[string]struct{}, len(columnOrder))
	headers := make([]string, 0, len(allKeySet))
	for _, key := range columnOrder {
		if _, ok := allKeySet[key]; ok {
			headers = append(headers, key)
			seen[key] = struct{}{}
		}
	}
	var extra []string
	for key := range allKeySet {
		if _, ok := seen[key]; !ok {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	headers = append(headers, extra...)

	if err := w.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(headers))
		for i, header := range headers {
			record[i] = row[header]
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func expandToRows(value any, prefix string) ([]map[string]string, []string) {
	switch v := value.(type) {
	case map[string]any:
		return expandObjectToRows(v, prefix, false)
	case []any:
		if isObjectArray(v) {
			var rows []map[string]string
			var keys []string
			for _, item := range v {
				itemRows, itemKeys := expandToRows(item, prefix)
				rows = append(rows, itemRows...)
				keys = mergeKeyOrder(keys, itemKeys)
			}
			return rows, keys
		}
		flat := make(map[string]string)
		flattenInto(flat, v, prefix)
		return []map[string]string{flat}, sortedKeysOf(flat)
	default:
		return []map[string]string{{prefix: scalarToString(value)}}, []string{prefix}
	}
}

func expandObjectToRows(obj map[string]any, prefix string, inlineObjects bool) ([]map[string]string, []string) {
	directScalars := make(map[string]string)
	inlinedFields := make(map[string]string)
	var nestedItems []any
	nestedFound := false

	for key, val := range obj {
		fullKey := joinKey(prefix, key)
		switch v := val.(type) {
		case []any:
			if isObjectArray(v) && !nestedFound {
				nestedItems = v
				nestedFound = true
			} else {
				flattenInto(directScalars, v, fullKey)
			}
		case map[string]any:
			if inlineObjects {
				flattenInto(inlinedFields, v, "")
			} else {
				flattenInto(directScalars, v, fullKey)
			}
		default:
			flattenInto(directScalars, v, fullKey)
		}
	}

	contextKeys := sortedKeysOf(directScalars)
	dataKeys := sortedKeysOf(inlinedFields)

	if !nestedFound {
		baseRow := make(map[string]string, len(directScalars)+len(inlinedFields))
		maps.Copy(baseRow, directScalars)
		maps.Copy(baseRow, inlinedFields)
		return []map[string]string{baseRow}, append(contextKeys, dataKeys...)
	}

	var rows []map[string]string
	var childKeys []string

	for _, item := range nestedItems {
		itemObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		childRows, itemKeys := expandObjectToRows(itemObj, "", true)
		childKeys = mergeKeyOrder(childKeys, itemKeys)
		for _, childRow := range childRows {
			row := make(map[string]string, len(directScalars)+len(inlinedFields)+len(childRow))
			maps.Copy(row, directScalars)
			maps.Copy(row, inlinedFields)
			maps.Copy(row, childRow)
			rows = append(rows, row)
		}
	}

	if len(rows) == 0 {
		baseRow := make(map[string]string, len(directScalars)+len(inlinedFields))
		maps.Copy(baseRow, directScalars)
		maps.Copy(baseRow, inlinedFields)
		return []map[string]string{baseRow}, append(contextKeys, dataKeys...)
	}

	// Final column order: parent context keys -> child keys -> parent data keys.
	orderedKeys := appendUnique(contextKeys, childKeys)
	orderedKeys = appendUnique(orderedKeys, dataKeys)
	return rows, orderedKeys
}

// isObjectArray returns true only when every element of arr is a JSON object.
func isObjectArray(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	for _, item := range arr {
		if _, ok := item.(map[string]any); !ok {
			return false
		}
	}
	return true
}

func flattenInto(dst map[string]string, value any, prefix string) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			flattenInto(dst, child, joinKey(prefix, key))
		}
	case []any:
		for i, item := range v {
			flattenInto(dst, item, joinKey(prefix, strconv.Itoa(i)))
		}
	default:
		dst[prefix] = scalarToString(v)
	}
}

func scalarToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// mergeKeyOrder returns base with any keys from extra appended that are not
// already present, preserving the order of both slices.
func mergeKeyOrder(base, extra []string) []string {
	if base == nil {
		return append([]string{}, extra...)
	}
	seen := make(map[string]struct{}, len(base))
	for _, k := range base {
		seen[k] = struct{}{}
	}
	result := append([]string{}, base...)
	for _, k := range extra {
		if _, ok := seen[k]; !ok {
			result = append(result, k)
			seen[k] = struct{}{}
		}
	}
	return result
}

// appendUnique appends keys from extra to base, skipping duplicates.
func appendUnique(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, k := range base {
		seen[k] = struct{}{}
	}
	result := append([]string{}, base...)
	for _, k := range extra {
		if _, ok := seen[k]; !ok {
			result = append(result, k)
			seen[k] = struct{}{}
		}
	}
	return result
}

func sortedKeysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

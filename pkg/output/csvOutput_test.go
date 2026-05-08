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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCsvOutput(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		wantContain []string
	}{
		{
			name:        "Simple object",
			data:        []byte(`{"name":"John","age":30}`),
			wantContain: []string{"age,name", "30,John"},
		},
		{
			name:        "Array of flat objects",
			data:        []byte(`[{"ip":"1.0.0.1","country":"AU"},{"ip":"2.0.0.1","country":"US"}]`),
			wantContain: []string{"country,ip", "AU,1.0.0.1", "US,2.0.0.1"},
		},
		{
			name:        "Nested object flattened with dot notation",
			data:        []byte(`{"country":{"iso_code":"US","name":"United States"}}`),
			wantContain: []string{"country.iso_code,country.name", "US,United States"},
		},
		{
			name:        "Boolean and float fields",
			data:        []byte(`{"active":true,"score":3.14}`),
			wantContain: []string{"active,score", "true,3.14"},
		},
		{
			name:        "Null field becomes empty string",
			data:        []byte(`{"empty":null,"name":"test"}`),
			wantContain: []string{"empty,name", ",test"},
		},
		{
			name:        "Array with missing keys uses empty string",
			data:        []byte(`[{"a":"1","b":"2"},{"a":"3"}]`),
			wantContain: []string{"a,b", "1,2", "3,"},
		},
		{
			name:        "Array of scalars uses index notation",
			data:        []byte(`{"tags":["go","cli"]}`),
			wantContain: []string{"tags.0,tags.1", "go,cli"},
		},
		{
			// Expanded items contain only scalar fields; each becomes its own row.
			// "query" (parent context) comes before "network" (expanded item context).
			name: "Nested array of scalar-only objects expands to rows",
			data: []byte(`{"query":"1.1.1.1","records":[{"network":"1.1.1.0/24"},{"network":"1.1.1.1/32"}]}`),
			wantContain: []string{
				"query,network",
				"1.1.1.1,1.1.1.0/24",
				"1.1.1.1,1.1.1.1/32",
			},
		},
		{
			// Context columns ("query", "network") lead; inlined data columns follow sorted.
			// Neither "records." nor "record." appears in any column name.
			name: "Expanded items with object fields: context columns lead, objects inlined",
			data: []byte(`[{"query":"1.1.1.1","records":[{"network":"1.1.1.0/24","record":{"autonomous_system_number":13335,"country":{"iso_code":"AU"}}}]},{"query":"8.8.8.8","records":[{"network":"8.8.8.0/24","record":{"autonomous_system_number":15169,"country":{"iso_code":"US"}}}]}]`),
			wantContain: []string{
				"query,network,autonomous_system_number,country.iso_code",
				"1.1.1.1,1.1.1.0/24,13335,AU",
				"8.8.8.8,8.8.8.0/24,15169,US",
			},
		},
		{
			name: "Top-level array with nested scalar-only object-arrays",
			data: []byte(`[{"query":"1.1.1.1","records":[{"network":"1.1.1.0/24"}]},{"query":"8.8.8.8","records":[{"network":"8.8.8.0/24"},{"network":"8.8.8.8/32"}]}]`),
			wantContain: []string{
				"query,network",
				"1.1.1.1,1.1.1.0/24",
				"8.8.8.8,8.8.8.0/24",
				"8.8.8.8,8.8.8.8/32",
			},
		},
		{
			name:    "Invalid JSON",
			data:    []byte(`{"broken":`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			out := captureStdout(t, func() {
				err = CsvOutput(tt.data, OutputOptions{})
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				for _, s := range tt.wantContain {
					assert.Contains(t, out, s)
				}
			}
		})
	}
}

func TestOutputCsvContent(t *testing.T) {
	data := []byte(`{"name":"John","age":30}`)
	options := OutputOptions{Format: "csv"}

	out := captureStdout(t, func() {
		err := Output(data, options)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "age,name")
	assert.Contains(t, out, "30,John")
}

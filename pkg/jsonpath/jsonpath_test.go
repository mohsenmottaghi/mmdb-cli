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

package jsonpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid simple field access",
			expression: "{.country}",
			wantErr:    false,
		},
		{
			name:       "valid nested field access",
			expression: "{.country.iso_code}",
			wantErr:    false,
		},
		{
			name:       "valid filter expression",
			expression: `{[?(@.country.iso_code=="US")]}`,
			wantErr:    false,
		},
		{
			name:       "valid wildcard",
			expression: "{.names.*}",
			wantErr:    false,
		},
		{
			name:       "valid range with end",
			expression: `{range .items[*]}{.network}{"\n"}{end}`,
			wantErr:    false,
		},
		{
			name:       "valid union",
			expression: "{['network','record']}",
			wantErr:    false,
		},
		{
			name:       "valid negative index",
			expression: "{.items[-1:]}",
			wantErr:    false,
		},
		{
			name:       "invalid syntax - unclosed brace",
			expression: "{.country",
			wantErr:    true,
		},
		{
			name:       "invalid syntax - bad filter",
			expression: "{[?(@.country==}",
			wantErr:    true,
		},
		{
			name:       "empty string is valid",
			expression: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateExpression(tt.expression)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteTemplate(t *testing.T) {
	t.Parallel()

	root := map[string]interface{}{
		"apiVersion": "mmdb-cli/v1",
		"kind":       "InspectList",
		"items": []map[string]interface{}{
			{
				"query":   "1.1.1.1",
				"network": "1.1.1.0/24",
				"record": map[string]interface{}{
					"registered_country": map[string]interface{}{
						"iso_code": "AU",
					},
				},
			},
			{
				"query":   "8.8.8.8",
				"network": "8.8.8.0/24",
				"record": map[string]interface{}{
					"registered_country": map[string]interface{}{
						"iso_code": "US",
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		expression string
		root       any
		want       string
		wantErr    bool
	}{
		{
			name:       "simple field access",
			expression: "{.apiVersion}",
			root:       root,
			want:       "mmdb-cli/v1",
			wantErr:    false,
		},
		{
			name:       "range over items - network only",
			expression: `{range .items[*]}{.network}{"\n"}{end}`,
			root:       root,
			want:       "1.1.1.0/24\n8.8.8.0/24\n",
			wantErr:    false,
		},
		{
			name:       "range with tab-separated fields",
			expression: `{range .items[*]}{.network}{"\t"}{.record.registered_country.iso_code}{"\n"}{end}`,
			root:       root,
			want:       "1.1.1.0/24\tAU\n8.8.8.0/24\tUS\n",
			wantErr:    false,
		},
		{
			name:       "range with inline filter",
			expression: `{range .items[?(@.record.registered_country.iso_code=="AU")]}{.network}{"\n"}{end}`,
			root:       root,
			want:       "1.1.1.0/24\n",
			wantErr:    false,
		},
		{
			name:       "wildcard field access",
			expression: "{.items[*].network}",
			root:       root,
			want:       "1.1.1.0/24 8.8.8.0/24",
			wantErr:    false,
		},
		{
			name:       "missing field renders empty - no error",
			expression: "{.items[0].nonexistent}",
			root:       root,
			want:       "",
			wantErr:    false,
		},
		{
			name:       "no implicit trailing newline",
			expression: "{.apiVersion}",
			root:       root,
			want:       "mmdb-cli/v1",
			wantErr:    false,
		},
		{
			name:       "invalid expression returns error",
			expression: "{.unclosed",
			root:       root,
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ExecuteTemplate(tt.expression, tt.root)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

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
	"bytes"
	"fmt"

	k8sjsonpath "k8s.io/client-go/util/jsonpath"
)

// ValidateExpression parses the expression and returns an error if it is
// syntactically invalid. Call this once up-front to give users a clear error
// before starting any expensive iteration.
func ValidateExpression(expression string) error {
	j := k8sjsonpath.New("validate")
	if err := j.Parse(expression); err != nil {
		return fmt.Errorf("invalid jsonpath expression: %w", err)
	}
	return nil
}

// ExecuteTemplate renders expression against root and returns raw bytes exactly
// as the k8s JSONPath library produces them. No implicit newline is added.
func ExecuteTemplate(expression string, root any) ([]byte, error) {
	j := k8sjsonpath.New("template").AllowMissingKeys(true)
	if err := j.Parse(expression); err != nil {
		return nil, fmt.Errorf("invalid jsonpath expression: %w", err)
	}
	var buf bytes.Buffer
	if err := j.Execute(&buf, root); err != nil {
		return nil, fmt.Errorf("jsonpath execution error: %w", err)
	}
	data := buf.Bytes()
	if data == nil {
		data = []byte{}
	}
	return data, nil
}

// Copyright 2024 The MMDB CLI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inspect

import (
	"encoding/json"
	"net"
	"testing"
)

func TestDetermineLookupNetwork(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1", "192.168.1.1/32"},
		{"2001:db8::", "2001:db8::/128"},
		{"192.168.1.0/24", "192.168.1.0/24"},
		{"2001:db8::/32", "2001:db8::/32"},
	}

	for _, test := range tests {
		result := determineLookupNetwork(test.input)
		if result != test.expected {
			t.Errorf("determineLookupNetwork(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestDetermineLookupNetworkInvalidInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("determineLookupNetwork did not panic on invalid input")
		}
	}()

	determineLookupNetwork("invalid_input")
}

func TestMMDBReader(t *testing.T) {
	testInput := "../../output/GeoLite2-Country.mmdb"

	_, err := mmdbReader(testInput)
	if err != nil {
		t.Errorf("mmdbReader() error = %v; want nil", err)
	}
}

func TestMMDBLookup(t *testing.T) {
	testInput := "../../output/GeoLite2-Country.mmdb"
	testsData := []struct {
		query    string
		expected string
	}{
		{"1.1.1.1", `{"registered_country":{"geoname_id":2077456,"iso_code":"AU","names":{"de":"Australien","en":"Australia","es":"Australia","fr":"Australie","ja":"オーストラリア","pt-BR":"Austrália","ru":"Австралия","zh-CN":"澳大利亚"}}}`},
	}

	for _, test := range testsData {

		reader, _ := mmdbReader(testInput)
		query := net.ParseIP(test.query)

		record, err := mmdbLookup(reader, query)

		recordJson, _ := json.Marshal(record)
		expectedJson := []byte(test.expected)

		if (err != nil) || (string(recordJson) != string(expectedJson)) {
			t.Errorf("mmdbLookup() = %v; want %v", string(recordJson), string(expectedJson))
		}
	}
}
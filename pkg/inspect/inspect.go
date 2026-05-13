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

package inspect

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/InfraZ/mmdb-cli/pkg/jsonpath"
	"github.com/oschwald/maxminddb-golang"
)

type CmdInspectConfig struct {
	InputFile string
	Inputs    []string
	JSONPath  string
}

// InspectResult holds the output of InspectInMMDB. When RawOutput is true,
// Data contains raw JSONPath template output and should be printed directly.
// When false, Data is JSON and should go through output.Output.
type InspectResult struct {
	Data      []byte
	RawOutput bool
}

func determineLookupNetwork(input string) (string, error) {
	var lookupNetwork string

	if !strings.Contains(input, "/") {
		if strings.Contains(input, ".") {
			lookupNetwork = input + "/32"
		} else if strings.Contains(input, ":") {
			lookupNetwork = input + "/128"
		} else {
			err := errors.New("invalid input")
			return lookupNetwork, err
		}
	} else {
		lookupNetwork = input
	}

	return lookupNetwork, nil
}

func mmdbReader(input string) (*maxminddb.Reader, error) {
	db, err := maxminddb.Open(input)
	if err != nil {
		return nil, fmt.Errorf("failed to open MMDB database: %w", err)
	}
	return db, nil
}

func mmdbLookup(reader *maxminddb.Reader, query net.IP) (any, error) {
	var records any
	err := reader.Lookup(query, &records)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IP %s: %w", query, err)
	}
	return records, nil
}

func mmdbNetworksWithin(reader *maxminddb.Reader, query *net.IPNet) *maxminddb.Networks {
	networksList := reader.NetworksWithin(
		query,
		maxminddb.SkipAliasedNetworks,
	)
	return networksList
}

type inspectQueryResult struct {
	query   string
	records []map[string]interface{}
}

func InspectInMMDB(cfg CmdInspectConfig) (InspectResult, error) {
	reader, err := mmdbReader(cfg.InputFile)
	if err != nil {
		return InspectResult{}, err
	}

	var mode jsonpath.Mode
	if cfg.JSONPath != "" {
		if err := jsonpath.ValidateExpression(cfg.JSONPath); err != nil {
			return InspectResult{}, fmt.Errorf("invalid JSONPath expression: %w", err)
		}
		mode = jsonpath.DetectMode(cfg.JSONPath)
	}

	queryResults := make([]inspectQueryResult, 0, len(cfg.Inputs))

	for _, input := range cfg.Inputs {
		lookupNetwork, err := determineLookupNetwork(input)
		if err != nil {
			return InspectResult{}, fmt.Errorf("invalid input: %s", input)
		}

		_, netIPNet, err := net.ParseCIDR(lookupNetwork)
		if err != nil {
			return InspectResult{}, fmt.Errorf("invalid input: %s", input)
		}

		inputNetworks := mmdbNetworksWithin(reader, netIPNet)
		records := make([]map[string]interface{}, 0)

		for inputNetworks.Next() {
			var anyNetwork any
			address, err := inputNetworks.Network(&anyNetwork)
			if err != nil {
				return InspectResult{}, fmt.Errorf("failed to get network: %w", err)
			}

			record, err := mmdbLookup(reader, address.IP)
			if err != nil {
				return InspectResult{}, fmt.Errorf("failed to lookup record: %w", err)
			}

			records = append(records, map[string]interface{}{
				"network": address.String(),
				"record":  record,
			})
		}

		if cfg.JSONPath != "" && mode == jsonpath.ModeLegacyFilter {
			filtered := make([]map[string]interface{}, 0)
			for _, entry := range records {
				rec, _ := entry["record"].(map[string]interface{})
				match, err := jsonpath.MatchesRecord(cfg.JSONPath, rec)
				if err != nil {
					return InspectResult{}, fmt.Errorf("failed to evaluate JSONPath expression: %w", err)
				}
				if match {
					filtered = append(filtered, entry)
				}
			}
			records = filtered
		}

		queryResults = append(queryResults, inspectQueryResult{query: input, records: records})
	}

	if cfg.JSONPath != "" && mode == jsonpath.ModeTemplate {
		items := make([]map[string]interface{}, 0)
		queries := make([]map[string]interface{}, 0, len(queryResults))
		for _, qr := range queryResults {
			for _, entry := range qr.records {
				items = append(items, map[string]interface{}{
					"query":   qr.query,
					"network": entry["network"],
					"record":  entry["record"],
				})
			}
			queries = append(queries, map[string]interface{}{
				"query":   qr.query,
				"records": qr.records,
			})
		}

		root := map[string]interface{}{
			"apiVersion": "mmdb-cli/v1",
			"kind":       "InspectList",
			"items":      items,
			"queries":    queries,
		}

		data, err := jsonpath.ExecuteTemplate(cfg.JSONPath, root)
		if err != nil {
			return InspectResult{}, fmt.Errorf("failed to execute JSONPath template: %w", err)
		}
		return InspectResult{Data: data, RawOutput: true}, nil
	}

	// No JSONPath or legacy filter: return the current public shape.
	result := make([]map[string]interface{}, 0, len(queryResults))
	for _, qr := range queryResults {
		result = append(result, map[string]interface{}{
			"query":   qr.query,
			"records": qr.records,
		})
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return InspectResult{}, fmt.Errorf("failed to marshal result: %w", err)
	}
	return InspectResult{Data: jsonData, RawOutput: false}, nil
}

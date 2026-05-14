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

package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/InfraZ/mmdb-cli/pkg/inspect"
	"github.com/InfraZ/mmdb-cli/pkg/jsonpath"
	"github.com/InfraZ/mmdb-cli/pkg/output"

	"github.com/spf13/cobra"
)

const (
	inspectCmdName      = "inspect"
	inspectCmdShortDesc = "Inspect an IP address or CIDR in the MMDB file"
	inspectCmdLongDesc  = `This command allows you to inspect an IP address or CIDR in the MMDB file`

	jsonpathFormatPrefix = "jsonpath="
)

var cmdInspectConfig inspect.CmdInspectConfig

// inspectCmd represents the generate command
var inspectCmd = &cobra.Command{
	Use:   inspectCmdName,
	Short: inspectCmdShortDesc,
	Long:  inspectCmdLongDesc + "\n\nArgs:\n  [IP/CIDR]  IP address or CIDR to inspect in the MMDB file, It can be a single or multiple IP addresses or CIDRs",
	Run: func(cmd *cobra.Command, args []string) {
		cmdInspectConfig.Inputs = cmd.Flags().Args()

		if strings.HasPrefix(outputOptions.Format, jsonpathFormatPrefix) {
			expr := strings.TrimPrefix(outputOptions.Format, jsonpathFormatPrefix)
			if jsonpath.IsLegacyTopLevelFilter(expr) {
				log.Fatalf("top-level filter %q is not supported against the template root; use '{.items[?(...)]}' or '{range .items[?(...)]}...{end}'", expr)
			}
			cmdInspectConfig.JSONPath = expr
		}

		inspectResult, err := inspect.InspectInMMDB(cmdInspectConfig)
		if err != nil {
			log.Fatal(err)
		}

		if inspectResult.RawOutput {
			fmt.Print(string(inspectResult.Data))
			return
		}

		err = output.Output(inspectResult.Data, outputOptions)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	inspectCmd.Flags().StringVarP(&cmdInspectConfig.InputFile, "input", "i", "", "Input path of the MMDB file")
	inspectCmd.Flags().StringVarP(&outputOptions.Format, "format", "f", "yaml", `Output format (yaml, json, json-pretty, xml, csv, jsonpath='{...}')`)

	inspectCmd.Args = cobra.MinimumNArgs(1)

	inspectCmd.MarkFlagRequired("input")
}

/*
// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2024 The Nodevin Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
*/

package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var requestCmd = &cobra.Command{
	Use:   "request [network]",
	Short: "Make an RPC request to a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		network := args[0]

		method := viper.GetString("method")
		params := viper.GetString("params")
		headers := viper.GetString("header")
		endpoint := viper.GetString("endpoint")
		port := viper.GetInt("port")
		user := viper.GetString("rpc-user")
		pass := viper.GetString("rpc-pass")

		if method == "" {
			printUsageAndExample()
			return errors.New("HTTP method is required")
		}

		if endpoint == "" {
			endpoint = "http://127.0.0.1"
		}

		if port == 0 {
			port = utils.NetworkDefaultRPCPorts()[network]
		}
		if port == 0 {
			return fmt.Errorf("unknown network %q and no --port given", network)
		}

		url := fmt.Sprintf("%s:%d", endpoint, port)

		response, err := makeRequest(network, url, method, params, headers, user, pass)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}

		fmt.Println(string(response))
		return nil
	},
}

func printUsageAndExample() {
	fmt.Printf("Usage: %s request [network] --method <http-method> --params <json-data> --rpc-user <rpc-username> --rpc-pass <rpc-password> --header <optional-extra-headers> --endpoint <optional-api-endpoint>\n", utils.GetNodevinExecutable())
	fmt.Printf("Example: `%s request bitcoin --method getblockcount`\n", utils.GetNodevinExecutable())
	fmt.Printf("Example: `%s request bitcoin --method getblockheader --params '[\"00000000c937983704a73af28acdec37b049d214adbda81d7e2a3dd146f6ed09\"]'`\n", utils.GetNodevinExecutable())
}

func makeRequest(network, url, method, params, headers, user, pass string) ([]byte, error) {
	// geth-family JSON-RPC servers (e.g. core-geth) strictly reject anything
	// other than "jsonrpc":"2.0" with a -32600 "invalid request" error, while
	// Bitcoin-derived daemons are lenient about the field and already work
	// with "1.0" (the long-standing default here).
	jsonrpcVersion := "1.0"
	if isEthereumStyleRPC(network) {
		jsonrpcVersion = "2.0"
	}

	jsonData := map[string]interface{}{
		"jsonrpc": jsonrpcVersion,
		"id":      "nodevin",
		"method":  method,
	}

	var jsonParams []interface{}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &jsonParams); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	} else {
		jsonParams = []interface{}{}
	}

	jsonData["params"] = jsonParams

	jsonValue, _ := json.Marshal(jsonData)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if headers != "" {
		for _, header := range strings.Split(headers, ",") {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("request failed with status code %d: Unauthorized\nMaybe consider using the --user and --pass flags?", resp.StatusCode)
		}
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	return body, nil
}

func init() {
	requestCmd.Flags().StringP("method", "m", "", "HTTP method to use for the request")
	requestCmd.Flags().StringP("params", "p", "", "JSON data to send in the request body")
	requestCmd.Flags().StringP("header", "H", "", "Optional extra headers")
	requestCmd.Flags().StringP("endpoint", "e", "http://127.0.0.1", "Optional API endpoint")
	requestCmd.Flags().IntP("port", "P", 0, "Optional port to override the default")

	viper.BindPFlag("method", requestCmd.Flags().Lookup("method"))
	viper.BindPFlag("params", requestCmd.Flags().Lookup("params"))
	viper.BindPFlag("header", requestCmd.Flags().Lookup("header"))
	viper.BindPFlag("endpoint", requestCmd.Flags().Lookup("endpoint"))
	viper.BindPFlag("port", requestCmd.Flags().Lookup("port"))
}

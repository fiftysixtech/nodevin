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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMakeRequest_ReturnsResponseBody is the regression test for the bug
// where the response body from every nodevin request call was thrown away,
// so the command "succeeded" with no output for every chain. This proves
// makeRequest actually returns the bytes the server sent back.
func TestMakeRequest_ReturnsResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":42,"error":null,"id":"nodevin"}`))
	}))
	defer server.Close()

	body, err := makeRequest("bitcoin", server.URL, "getblockcount", "", "", "", "")
	if err != nil {
		t.Fatalf("makeRequest() returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response body %q: %v", body, err)
	}
	if parsed["result"] != float64(42) {
		t.Errorf("response body = %s, want result=42", body)
	}
}

// TestMakeRequest_SendsBasicAuthWhenCredentialsProvided proves user/pass
// flow into the outgoing request's Authorization header.
func TestMakeRequest_SendsBasicAuthWhenCredentialsProvided(t *testing.T) {
	var gotUser, gotPass string
	var gotOK bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotOK = r.BasicAuth()
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if _, err := makeRequest("bitcoin", server.URL, "getblockcount", "", "", "myuser", "mypass"); err != nil {
		t.Fatalf("makeRequest() returned error: %v", err)
	}

	if !gotOK {
		t.Fatal("expected the request to carry HTTP basic auth, got none")
	}
	if gotUser != "myuser" || gotPass != "mypass" {
		t.Errorf("BasicAuth() = (%q, %q), want (%q, %q)", gotUser, gotPass, "myuser", "mypass")
	}
}

// TestMakeRequest_ReturnsErrorOnUnauthorized proves the 401 path surfaces a
// helpful error instead of silently swallowing the failure.
func TestMakeRequest_ReturnsErrorOnUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := makeRequest("bitcoin", server.URL, "getblockcount", "", "", "", "")
	if err == nil {
		t.Fatal("expected an error for a 401 response, got nil")
	}
}

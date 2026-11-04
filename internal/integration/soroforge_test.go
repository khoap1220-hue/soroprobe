package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListTrackedContractsPaginationAndAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-ro-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"contracts":[{"contract_id":"CAAA","network":"testnet","wasm_hash":"abcd"}],"next_cursor":"p2"}`))
			return
		}
		_, _ = w.Write([]byte(`{"contracts":[{"contract_id":"CBBB","network":"testnet","wasm_hash":"ef01"}],"next_cursor":""}`))
	}))
	defer srv.Close()

	client, err := NewSoroForgeClient(srv.URL, "secret-ro-token", srv.Client(), 5*time.Second)
	if err != nil {
		t.Fatalf("NewSoroForgeClient: %v", err)
	}
	contracts, err := client.ListTrackedContracts(context.Background(), "testnet")
	if err != nil {
		t.Fatalf("ListTrackedContracts: %v", err)
	}
	if len(contracts) != 2 || contracts[0].ContractID != "CAAA" || contracts[1].ContractID != "CBBB" {
		t.Fatalf("unexpected contracts: %+v", contracts)
	}
	if ok, _ := CompareWasmHash("ABCD", "abcd"); !ok {
		t.Errorf("expected case-insensitive hash match")
	}
	if ok, _ := CompareWasmHash("abcd", "1234"); ok {
		t.Errorf("expected mismatch on drifted hash")
	}
}

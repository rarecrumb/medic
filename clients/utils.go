package clients

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// RPCResponse represents a standard JSON-RPC response
type RPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  string `json:"result"`
	ID      int    `json:"id"`
}

type SyncChecker interface {
	CheckSyncStatus() (*SyncStatus, error)
}

type SyncStatus struct {
	IsSyncing     bool
	StartingBlock *uint64
	CurrentBlock  *uint64
	HighestBlock  *uint64
}

// DetectClientType determines the type of Ethereum client by calling web3_clientVersion
func DetectClientType(nodeURL string) (string, error) {
	// JSON-RPC request payload
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "web3_clientVersion",
		"params":  []interface{}{},
		"id":      1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Send the request
	resp, err := http.Post(nodeURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read and unmarshal the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var rpcResponse RPCResponse
	err = json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return "", err
	}

	// Determine the client type
	switch {
	case strings.Contains(rpcResponse.Result, "Nethermind"):
		return "Nethermind", nil
	case strings.Contains(rpcResponse.Result, "erigon"):
		return "Erigon", nil
	case strings.Contains(rpcResponse.Result, "reth"):
		return "Reth", nil
	default:
		return "Unknown", nil
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Create a struct for your JSON payload
type Payload struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// EthRPCResponse represents the standard response from an Ethereum JSON-RPC call
type EthRPCResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  Block  `json:"result"`
}

// Block represents the Ethereum block details
type Block struct {
	Number           string   `json:"number"`
	Hash             string   `json:"hash"`
	ParentHash       string   `json:"parentHash"`
	Nonce            string   `json:"nonce"`
	Sha3Uncles       string   `json:"sha3Uncles"`
	LogsBloom        string   `json:"logsBloom"`
	TransactionsRoot string   `json:"transactionsRoot"`
	StateRoot        string   `json:"stateRoot"`
	ReceiptsRoot     string   `json:"receiptsRoot"`
	Miner            string   `json:"miner"`
	Difficulty       string   `json:"difficulty"`
	TotalDifficulty  string   `json:"totalDifficulty"`
	ExtraData        string   `json:"extraData"`
	Size             string   `json:"size"`
	GasLimit         string   `json:"gasLimit"`
	GasUsed          string   `json:"gasUsed"`
	Timestamp        string   `json:"timestamp"`
	Transactions     []string `json:"transactions"`
	Uncles           []string `json:"uncles"`
}

var isNodeHealthy bool

// getLatestBlock fetches the latest block information from the Ethereum node
func getLatestBlock(nodeURL string) (Block, error) {
	// JSON-RPC request payload
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{"latest", false},
		"id":      1,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Block{}, err
	}

	// Create a new POST request with the payload
	req, err := http.NewRequest("POST", nodeURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return Block{}, err
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Block{}, err
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Block{}, err
	}

	// Unmarshal the JSON response
	var rpcResponse EthRPCResponse
	err = json.Unmarshal(responseBody, &rpcResponse)
	if err != nil {
		return Block{}, err
	}

	return rpcResponse.Result, nil
}

func checkNodeHealth(clientURL string) bool {
	// Fetch the latest block information (assuming you have a function for this)
	latestBlock, err := getLatestBlock(clientURL)
	if err != nil {
		log.Println("Error getting latest block:", err)
		return false
	}

	// Convert the block timestamp from hexadecimal to an integer
	blockTimeHex := latestBlock.Timestamp
	blockTimeSec, err := strconv.ParseInt(blockTimeHex, 0, 64)
	if err != nil {
		log.Println("Error parsing block timestamp:", err)
		return false
	}

	// Convert Ethereum block time from seconds to a time.Time object
	blockTime := time.Unix(blockTimeSec, 0)

	// Get the current time
	currentTime := time.Now()

	// Compare block time with the current time
	maxSecondsBehind := viper.GetInt("max-seconds-behind")
	if currentTime.Sub(blockTime).Seconds() > float64(maxSecondsBehind) {
		log.Println("ERROR: Node is too far behind")
		return false
	}

	return true
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	clientURL := viper.GetString("eth-url") // Ensure this is set in your viper configuration
	if checkNodeHealth(clientURL) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func init() {
	// Set default values
	pflag.String("eth-url", "http://localhost:8545", "URL of Ethereum client")
	pflag.Int("max-seconds-behind", 60, "Maximum number of seconds behind")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Reading from a config file (optional)
	viper.SetConfigName("config") // name of the config file (without extension)
	viper.SetConfigType("yaml")   // or whichever config format you prefer
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Printf("No config file found or error reading it: %s\n", err)
	}
	viper.AutomaticEnv()
}

func main() {
	http.HandleFunc("/ready", readinessHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

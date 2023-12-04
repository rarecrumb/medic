package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
)

// Define a struct to parse the Ethereum client's health response
// {"check_block":"DISABLED","max_seconds_behind":"HEALTHY","min_peer_count":"HEALTHY","synced":"ERROR: not synced"}
type HealthResponse struct {
	CheckBlock       string `json:"check_block"`
	MaxSecondsBehind string `json:"max_seconds_behind"`
	MinPeerCount     string `json:"min_peer_count"`
	Synced           string `json:"synced"`
}

var isNodeHealthy bool

func setupConfig() {
	// Set default values
	viper.SetDefault("eth_url", "http://localhost:8545")
	viper.SetDefault("min_peer_count", 10)
	viper.SetDefault("max_seconds_behind", 60)
	viper.SetDefault("synced", true)

	// Environment variables
	viper.AutomaticEnv()

	// Set up command line flags
	// ... (if you want to use command-line flags, set them up here)

	// Reading from a config file (optional)
	viper.SetConfigName("config") // name of the config file (without extension)
	viper.SetConfigType("yaml")   // or whichever config format you prefer
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		log.Printf("No config file found or error reading it: %s\n", err)
	}
}

func isHealthyOrDisabled(status string) bool {
	return status == "HEALTHY" || status == "DISABLED"
}

func checkNodeHealth(clientURL string) {
	for {
		// Send request to Ethereum client's health endpoint
		client := &http.Client{}
		req, err := http.NewRequest("GET", clientURL+"/health", nil)
		if err != nil {
			log.Println("Error creating request:", err)
			isNodeHealthy = false
			continue
		}

		// Conditionally add headers based on configuration
		if viper.IsSet("check_block") {
			checkBlock := viper.GetInt("check_block")
			req.Header.Add("X-ERIGON-HEALTHCHECK", "check_block"+strconv.Itoa(checkBlock))
		}
		if viper.IsSet("min_peer_count") {
			minPeerCount := viper.GetInt("min_peer_count")
			req.Header.Add("X-ERIGON-HEALTHCHECK", "min_peer_count"+strconv.Itoa(minPeerCount))
		}
		if viper.IsSet("max_seconds_behind") {
			maxSecondsBehind := viper.GetInt("max_seconds_behind")
			req.Header.Add("X-ERIGON-HEALTHCHECK", "max_seconds_behind"+strconv.Itoa(maxSecondsBehind))
		}
		if viper.GetBool("synced") {
			req.Header.Add("X-ERIGON-HEALTHCHECK", "synced")
		}

		// Perform the request
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error checking node health:", err)
			isNodeHealthy = false
			continue
		}
		defer resp.Body.Close()

		// Parse the response
		var healthResponse HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&healthResponse)
		if err != nil {
			log.Println("Error decoding health response:", err)
			isNodeHealthy = false
			continue
		}

		// Determine health based on response
		isNodeHealthy = isHealthyOrDisabled(healthResponse.Synced) &&
			isHealthyOrDisabled(healthResponse.MinPeerCount) &&
			isHealthyOrDisabled(healthResponse.MaxSecondsBehind) &&
			isHealthyOrDisabled(healthResponse.CheckBlock)
	}
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	if isNodeHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func main() {
	setupConfig()

	http.HandleFunc("/ready", readinessHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

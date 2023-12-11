package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rarecrumb/medic/clients"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	// Set default values
	pflag.String("log-level", "info", "Log level")
	pflag.String("eth-url", "http://localhost:8545", "URL of the Ethereum client")
	pflag.Int("max-seconds-behind", 30, "Maximum number of seconds behind a block can be")
	pflag.Int("min-peers", 3, "Minimum number of peers the node should have")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.AutomaticEnv()
	log.Info().Msg("Service initialized")
}

func main() {
	url := viper.GetString("eth-url")
	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil
	retryClient.RetryMax = 50
	retryClient.RetryWaitMin = 5 * time.Second
	retryClient.RetryWaitMax = 15 * time.Second

	// Making a lightweight request to check node readiness
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new request")
	}

	resp, err := retryClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to perform a new request")
	}
	defer resp.Body.Close()

	http.HandleFunc("/ready", readinessHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal().Err(err).Msg("Failed to start the server")
		os.Exit(1) // Exit the program after logging the fatal error
	}
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	url := viper.GetString("eth-url")
	if nodeHealth(url) {
		w.WriteHeader(http.StatusOK)
	} else {
		log.Warn().Msg("Node is not healthy")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func blockDelta(url string) (int, error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to the Ethereum client")
		return 0, err
	}

	// Get the latest block
	blockNumber, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve the latest block")
		return 0, err
	}

	// Get the block timestamps
	blockTimestamp := time.Unix(int64(blockNumber.Time()), 0)
	currentTimestamp := time.Now()

	// Get the max-seconds-behind value
	maxSecondsBehind := viper.GetInt("max-seconds-behind")

	delta := currentTimestamp.Sub(blockTimestamp).Seconds()

	// Compare the timestamps
	if delta > float64(maxSecondsBehind) {
		log.Error().Msgf("Node is too far behind: %f", delta)
		return int(delta), err
	}

	return int(delta), nil
}

func checkNodePeers(url string) (int, error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to the Ethereum client")
		return 0, err
	}

	// Get the number of peers
	peerCount, err := client.PeerCount(context.Background())
	count := int(peerCount)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve the number of peers")
		return 0, err
	}

	// Get the min-peers value
	minPeers := viper.GetInt("min-peers")

	// Compare the number of peers
	if count < minPeers {
		return count, err
	}

	return count, nil
}

func nodeHealth(url string) bool {

	var isNodeHealthy bool
	// Check the block timestamp
	blockDelta, err := blockDelta(url)
	if err != nil {
		log.Error().
			Err(err).
			Int("block_delta", int(blockDelta)).
			Msg("Failed health check by block time delta")

		return false
	}

	// Check the number of peers
	peerCount, err := checkNodePeers(url)
	if err != nil {
		log.Error().
			Err(err).
			Int("peers", peerCount).
			Msg("Failed health check by peer count")

		return false
	}

	// Nethermind health check
	clientType, err := clients.DetectClientType(viper.GetString("eth-url"))
	if err != nil {
		log.Info().Err(err).Msg("Failed to detect the client type")
	}
	if clientType != "Nethermind" {
		isNodeHealthy = peerCount >= viper.GetInt("min-peers") &&
			blockDelta <= viper.GetInt("max-seconds-behind")

		log.Info().
			Bool("is_node_healthy", isNodeHealthy).
			Int("peer_count", peerCount).
			Int("block_delta", int(blockDelta)).
			Str("client_type", clientType).
			Msg("Node health check")

		return isNodeHealthy
	} else if clientType == "Nethermind" {
		health, err := clients.NethermindHealthCheck(url)
		if err != nil {
			log.Error().Err(err).Msg("Failed to retrieve the Nethermind health")
			return false
		}
		if len(health.Entries.NodeHealth.Data.Errors) != 0 {
			log.Error().Msgf("Node health errors: %v", health.Entries.NodeHealth.Data.Errors)
			return false
		}
		if health.Entries.NodeHealth.Data.IsSyncing {
			log.Error().Msg("Node is syncing")
			return false
		}
		isNodeHealthy = !health.Entries.NodeHealth.Data.IsSyncing &&
			peerCount >= viper.GetInt("min-peers") &&
			blockDelta <= viper.GetInt("max-seconds-behind")

		log.Info().
			Bool("is_node_healthy", isNodeHealthy).
			Bool("is_syncing", health.Entries.NodeHealth.Data.IsSyncing).
			Int("peer_count", peerCount).
			Int("block_delta", int(blockDelta)).
			Str("client_type", clientType).
			Msg("Node health check")

		return isNodeHealthy
	}
	return false
}

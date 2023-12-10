package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rarecrumb/medic/clients"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	// Setup logger
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Set default values
	pflag.String("eth-url", "http://localhost:8545", "URL of the Ethereum client")
	pflag.Int("max-seconds-behind", 60, "Maximum number of seconds behind a block can be")
	pflag.Int("min-peers", 3, "Minimum number of peers the node should have")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.AutomaticEnv()
	log.Info().Msg("Service initialized")
}

func main() {
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
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func blockDelta(url string) (uint64, error) {
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
		return uint64(delta), err
	}

	return uint64(delta), nil
}

func checkNodePeers(url string) (uint64, error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to the Ethereum client")
		return 0, err
	}

	// Get the number of peers
	peerCount, err := client.PeerCount(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve the number of peers")
		return 0, err
	}

	// Get the min-peers value
	minPeers := viper.GetUint64("min-peers")

	// Compare the number of peers
	if peerCount < minPeers {
		return peerCount, err
	}

	return peerCount, nil
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
			Int("peers", int(peerCount)).
			Msg("Failed health check by peer count")

		return false
	}

	// Nethermind health check
	clientType, err := clients.DetectClientType(viper.GetString("eth-url"))
	if err != nil {
		log.Info().Err(err).Msg("Failed to detect the client type")
	}
	if clientType != "Nethermind" {
		log.Info().
			Int("peers", int(peerCount)).
			Int("block_delta", int(blockDelta)).
			Msg("OK")

		isNodeHealthy = peerCount >= uint64(viper.GetInt("min-peers")) &&
			blockDelta <= uint64(viper.GetInt("max-seconds-behind"))

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
		log.Info().
			Bool("syncing", health.Entries.NodeHealth.Data.IsSyncing).
			Int("peers", int(peerCount)).
			Int("block_delta", int(blockDelta)).
			Msg("OK")

		isNodeHealthy = !health.Entries.NodeHealth.Data.IsSyncing &&
			peerCount >= uint64(viper.GetInt("min-peers")) &&
			blockDelta <= uint64(viper.GetInt("max-seconds-behind"))

		return isNodeHealthy
	}
	return false
}

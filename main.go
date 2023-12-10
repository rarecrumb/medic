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
	clientURL := viper.GetString("eth-url")
	if checkNodeHealth(clientURL) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func checkBlockDelta(clientURL string) bool {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(clientURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to the Ethereum client")
	}

	// Get the latest block
	blockNumber, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve the latest block")
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
		return false
	}

	return true
}

func checkNodePeers(clientURL string) bool {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(clientURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to the Ethereum client")
	}

	// Get the number of peers
	peerCount, err := client.PeerCount(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve the number of peers")
	}

	// Get the min-peers value
	minPeers := viper.GetUint64("min-peers")

	// Compare the number of peers
	if peerCount < minPeers {
		return false
	}

	return true
}

func checkNodeHealth(clientURL string) bool {
	var isNodeHealthy bool
	var isClientHealthy bool
	clientType, err := clients.DetectClientType(clientURL)
	if err != nil {
		log.Error().Err(err).Msg("Error detecting client type")
	}

	switch clientType {

	case "Nethermind":
		// Call the Nethermind-specific health check function
		isClientHealthy, err = clients.CheckNethermindHealth(clientURL)
		if err != nil {
			log.Error().Err(err).Msg("Nethermind health check failed")
		}
		if !isClientHealthy {
			log.Error().Msg("Nethermind node is not healthy")
		}

	case "Erigon":
		isClientHealthy = true

	default:
		isClientHealthy = true
	}
	isNodeHealthy = isClientHealthy && checkBlockDelta(clientURL) && checkNodePeers(clientURL)
	return isNodeHealthy
}

package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

type NethermindHealth struct {
	Status        string `json:"status"`
	TotalDuration string `json:"totalDuration"`
	Entries       struct {
		NodeHealth struct {
			Data struct {
				IsSyncing bool     `json:"IsSyncing"`
				Errors    []string `json:"Errors"`
			} `json:"data"`
			Description string `json:"description"`
			Duration    string `json:"duration"`
			Status      string `json:"status"`
		} `json:"node-health"`
	} `json:"entries"`
}

func CheckNethermindHealth(url string) (bool, error) {
	// Construct the full URL
	fullURL := fmt.Sprintf("%s/health", url)

	// Make the HTTP GET request
	resp, err := http.Get(fullURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read the health response")
		return false, err
	}

	// Unmarshal the JSON response
	var health NethermindHealth
	err = json.Unmarshal(body, &health)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal the health response")
		return false, err
	}

	// Evaluate the health status
	if health.Status == "Healthy" && !health.Entries.NodeHealth.Data.IsSyncing && len(health.Entries.NodeHealth.Data.Errors) == 0 {
		return true, nil
	}

	return false, fmt.Errorf("Nethermind node is not healthy: %+v", health)
}

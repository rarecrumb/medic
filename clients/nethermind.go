package clients

import (
	"encoding/json"
	"net/http"
)

type NethermindHealth struct {
	Status  string `json:"status"`
	Entries struct {
		NodeHealth struct {
			Data struct {
				Errors    []string `json:"Errors"`
				IsSyncing bool     `json:"IsSyncing"`
			} `json:"data"`
		} `json:"node-health"`
	} `json:"entries"`
}

func NethermindHealthCheck(url string) (*NethermindHealth, error) {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health NethermindHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}

package main

import (
	"encoding/json"
	"os"
)

func loadConfig() ([]Service, error) {
	fh, err := os.Open("services.json")
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var services []Service

	dec := json.NewDecoder(fh)

	if err := dec.Decode(&services); err != nil {
		return nil, err
	}

	return services, nil
}

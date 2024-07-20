package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

type App struct {
	LogGroupName string `json:"log_group_name"`
}

func findConfigFile() (*string, error) {
	// TODO: Thrash out possible config file locations
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		loc := path.Join(configHome, "tracey", "tracey.json")
		if _, err := os.Stat(loc); err == nil {
			return &loc, nil
		}
	}
	return nil, errors.New("no config file found")
}

func Parse() (*App, error) {
	path, err := findConfigFile()
	if err != nil {
		return nil, fmt.Errorf("cannot determine config path: %w", err)
	}
	f, err := os.Open(*path)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	var cfg App
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file json: %w", err)
	}
	return &cfg, nil
}

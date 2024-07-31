package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/itchyny/gojq"
)

type App struct {
	Logs         Logs     `json:"logs"`
	ExcludePaths []string `json:"exclude_paths,omitempty"`

	// These are populated after parsing JSON
	ParsedExcludePaths []regexp.Regexp `json:"-"`
}

type Logs struct {
	Groups []string   `json:"groups"`
	Fields []LogField `json:"fields,omitempty"`

	// These are populated after parsing JSON
	ParsedFields []ParsedLogField `json:"-"`
}

type LogField struct {
	Title string `json:"title"`
	Query string `json:"query"`
}

type ParsedLogField struct {
	Title string     `json:"-"`
	Query gojq.Query `json:"-"`
}

func findConfigFile() (*string, error) {
	// Check cwd and parents
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}
	filename := ".tracey.json"
	for {
		filePath := filepath.Join(dir, filename)
		if _, statErr := os.Stat(filePath); statErr == nil {
			return &filePath, nil
		}

		parentDir := filepath.Dir(dir)
		// Check if we have reached the root directory
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	// Now check XDG_CONFIG_HOME
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		loc := path.Join(configHome, "tracey", "tracey.json")
		if _, statErr := os.Stat(loc); statErr == nil {
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

	logs := &cfg.Logs
	logs.ParsedFields = make([]ParsedLogField, len(logs.Fields))
	for i, field := range logs.Fields {
		lf, jqErr := gojq.Parse(field.Query)
		if jqErr != nil {
			return nil, fmt.Errorf("error parsing log field: %w", jqErr)
		}
		logs.ParsedFields[i] = ParsedLogField{Title: field.Title, Query: *lf}
	}

	cfg.ParsedExcludePaths = make([]regexp.Regexp, len(cfg.ExcludePaths))
	for i, exclude := range cfg.ExcludePaths {
		re, reErr := regexp.Compile(exclude)
		if reErr != nil {
			return nil, fmt.Errorf("error compiling exclude regex: %w", reErr)
		}
		cfg.ParsedExcludePaths[i] = *re
	}

	return &cfg, nil
}

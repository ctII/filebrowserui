package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

type Config struct {
	User string `json:"user"`
	Pass string `json:"pass"`

	// Host is the full URI of the host.
	// Ex: https://filebrowser.clayton.coffee
	Host string `json:"host"`

	// Dir is the parent folder that contains our files.
	// Ex: ~/.config/filebrowser/
	Dir string `json:"-"`

	// loaded is true if we loaded from a configuration file succesfully,
	// or false if some error occurred and we should warn user before saving.
	// TODO: smartly determine what we can save safely, likely everything.
	loaded bool

	// changed is true if a field that would be written to disk has changed.
	changed bool
}

var config *Config

func GetConfig() *Config {
	return config
}

const configFileName = "config.json"

func parseConfigPath(path string) error {
	// #nosec G304 -- we want to include the filepath, since it is a configuration file
	bs, err := os.ReadFile(path) // TODO: check permissions on our secrets file and warn user if they are bad
	if err != nil {
		// if the file doesn't exist, just continue with the defaults
		if errors.Is(err, fs.ErrNotExist) {
			slog.Info("configuration file doesn't exist, instead using defaults", "error", err)
			config.loaded = true
			return nil
		}

		return fmt.Errorf("could not read config file (%v): %w", path, err)
	}

	// TODO: validate that this configuration file is actually ours

	c := *config // TODO: check this actually copys the struct
	if err = json.Unmarshal(bs, &c); err != nil {
		return fmt.Errorf("invalid json config file (%v): %w", path, err)
	}

	// single op assignment to prevent json parsing errors from changing config
	// this allows using defaults if json unmarshaling errors out
	config = &c
	config.loaded = true

	return nil
}

func parseConfig() error {
	config = &Config{} // TODO: this is bad, but currently how we signal to logic() that we have run before

	defaultConfigDir, err := os.UserConfigDir()
	if err != nil {
		slog.Warn("could not get default user config directory, attempting to use current directory", "error", err)

		defaultConfigDir, err := os.Getwd()
		if err != nil {
			slog.Warn("could not get current working directory, using \"\" as base path or \"configDir\" parameter if defined. hoping relative paths work", "error", err)
		} else {
			config.Dir = defaultConfigDir
		}
	}

	configDir := flag.String("configDir", defaultConfigDir, "path to configuration directory for filebrowser")
	flag.Parse() // TODO: if this fails we os.Exit, where everywhere else we just send a warning.
	config.Dir = filepath.Join(*configDir, "filebrowserui")
	slog.Info("using configuration directory", "path", config.Dir)

	configFilePath := filepath.Join(config.Dir, configFileName)

	return parseConfigPath(configFilePath)
}

func saveConfig() error {
	bs, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("json marshaling error, report this error to developer: %w", err)
	}

	// create all parent directories and our directory with unix perm 750
	if err := os.MkdirAll(config.Dir, 0750); err != nil {
		return fmt.Errorf("could not create configuration directory (%v): %w", config.Dir, err)
	}

	configFilePath := filepath.Join(config.Dir, configFileName)

	// write configuration file to our directory, but only user writable
	if err := os.WriteFile(configFilePath, bs, 0600); err != nil {
		return fmt.Errorf("could not write file (%v): %w", configFilePath, err)
	}

	slog.Info("saved configuration file", "path", configFilePath)

	return nil
}

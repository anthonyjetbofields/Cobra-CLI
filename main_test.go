package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func TestConfigReloadPreservesEnvOverrides(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")

	initialConfig := []byte("server:\n  port: 8080\n  debug: false\n")
	if err := os.WriteFile(configPath, initialConfig, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("COBRA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.BindEnv("server.port", "COBRA_SERVER_PORT"); err != nil {
		t.Fatalf("Failed to bind env var: %v", err)
	}

	t.Setenv("COBRA_SERVER_PORT", "9090")

	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if port := viper.GetString("server.port"); port != "9090" {
		t.Errorf("Expected port to be overridden by env to 9090, got %s", port)
	}
	if debug := viper.GetBool("server.debug"); debug != false {
		t.Errorf("Expected debug to be false, got %v", debug)
	}

	reloadChan := make(chan struct{})
	viper.OnConfigChange(func(e fsnotify.Event) {
		if err := viper.ReadInConfig(); err != nil {
			t.Logf("Error reloading config (retrying): %v", err)
			return
		}
		viper.AutomaticEnv()
		_ = viper.BindEnv("server.port", "COBRA_SERVER_PORT")
		select {
		case <-reloadChan:
		default:
			close(reloadChan)
		}
	})
	viper.WatchConfig()

	updatedConfig := []byte("server:\n  port: 8080\n  debug: true\n")
	if err := os.WriteFile(configPath, updatedConfig, 0644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	select {
	case <-reloadChan:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for config reload event")
	}

	if port := viper.GetString("server.port"); port != "9090" {
		t.Errorf("After reload, expected port to remain 9090 (env override), got %s", port)
	}
	if debug := viper.GetBool("server.debug"); debug != true {
		t.Errorf("After reload, expected debug to be updated to true, got %v", debug)
	}
}

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func writeShipConfig() error {
	configPath, _, err := shipConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := os.Chmod(configPath, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(file, "SHIP_DOMAIN=%s\n", getenv("SHIP_DOMAIN"))
	dns := getenv("SHIP_DNS")
	if dns == "" {
		dns = "manual"
	}
	fmt.Fprintf(file, "SHIP_DNS=%s\n", dns)
	writeConfigLine(file, "SHIP_DASHBOARD_SERVICE", getenv("SHIP_DASHBOARD_SERVICE"), "k8s")
	writeConfigLine(file, "SHIP_IMAGE_PREFIX", getenv("SHIP_IMAGE_PREFIX"), "ship")
	writeConfigLine(file, "KIND_CLUSTER", getenv("KIND_CLUSTER"), "ship")
	writeConfigLine(file, "SHIP_REPO", getenv("SHIP_REPO"), sourceRepo)
	writeConfigLine(file, "SHIP_REF", getenv("SHIP_REF"), sourceRef)
	writeConfigLine(file, "CLOUDFLARE_API_TOKEN", getenv("CLOUDFLARE_API_TOKEN"), "")
	writeConfigLine(file, "CF_API_TOKEN", getenv("CF_API_TOKEN"), "")
	writeConfigLine(file, "CLOUDFLARE_ACCOUNT_ID", getenv("CLOUDFLARE_ACCOUNT_ID"), "")
	writeConfigLine(file, "CF_ACCOUNT_ID", getenv("CF_ACCOUNT_ID"), "")
	writeConfigLine(file, "CLOUDFLARE_ZONE_ID", getenv("CLOUDFLARE_ZONE_ID"), "")
	writeConfigLine(file, "CF_ZONE_ID", getenv("CF_ZONE_ID"), "")
	writeConfigLine(file, "CLOUDFLARE_TUNNEL_ID", getenv("CLOUDFLARE_TUNNEL_ID"), "")
	writeConfigLine(file, "SHIP_CLOUDFLARE_TUNNEL_NAME", getenv("SHIP_CLOUDFLARE_TUNNEL_NAME"), "")
	return nil
}

func writeConfigLine(writer io.Writer, key string, value string, defaultValue string) {
	if value != "" && value != defaultValue {
		fmt.Fprintf(writer, "%s=%s\n", key, value)
	}
}

func shipConfigPath() (string, string, error) {
	if path := os.Getenv("SHIP_CONFIG"); path != "" {
		return path, filepath.Dir(path), nil
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		dir := filepath.Join(configHome, "ship")
		return filepath.Join(dir, "config.env"), dir, nil
	}

	if currentOS == "windows" {
		configHome = firstEnv("LOCALAPPDATA", "APPDATA")
		if configHome == "" {
			var err error
			configHome, err = os.UserConfigDir()
			if err != nil {
				return "", "", fmt.Errorf("resolve ship config dir: %w", err)
			}
		}
		dir := filepath.Join(configHome, "ship")
		return filepath.Join(dir, "config.env"), dir, nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		configHome = firstEnv("LOCALAPPDATA", "APPDATA")
	} else {
		configHome = filepath.Join(home, ".config")
	}
	if configHome == "" {
		return "", "", fmt.Errorf("missing HOME for ship config")
	}
	dir := filepath.Join(configHome, "ship")
	return filepath.Join(dir, "config.env"), dir, nil
}

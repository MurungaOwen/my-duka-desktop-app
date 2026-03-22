package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Config struct {
	DBPath     string
	Mode       DeploymentMode
	DeviceID   string
	DeviceName string
}

func DefaultConfig() (Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("resolve working directory: %w", err)
	}

	deviceName, err := os.Hostname()
	if err != nil {
		deviceName = "myduka-device"
	}

	return Config{
		DBPath:     filepath.Join(wd, "myduka.sqlite"),
		Mode:       DeploymentModeStandalone,
		DeviceID:   uuid.NewString(),
		DeviceName: deviceName,
	}, nil
}

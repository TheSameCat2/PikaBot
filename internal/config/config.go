package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	MatrixHomeserver  string
	MatrixAccessToken string
	MatrixUser        string
	MatrixPassword    string
	MatrixUserID      string
	MatrixRoomID      string
	AllowedMXIDs      map[string]struct{}

	DockerContainerName string

	RCONHost string
	RCONPort int
	RCONPass string

	CommandPrefix string
	DataDir       string
}

func Load() (Config, error) {
	cfg := Config{
		MatrixHomeserver:    strings.TrimSpace(os.Getenv("MATRIX_HOMESERVER")),
		MatrixAccessToken:   strings.TrimSpace(os.Getenv("MATRIX_ACCESS_TOKEN")),
		MatrixUser:          strings.TrimSpace(os.Getenv("MATRIX_USER")),
		MatrixPassword:      strings.TrimSpace(os.Getenv("MATRIX_PASSWORD")),
		MatrixUserID:        strings.TrimSpace(os.Getenv("MATRIX_USER_ID")),
		MatrixRoomID:        strings.TrimSpace(os.Getenv("MATRIX_ROOM_ID")),
		DockerContainerName: envOrDefault("DOCKER_CONTAINER_NAME", "Palworld"),
		RCONHost:            envOrDefault("RCON_HOST", "127.0.0.1"),
		RCONPort:            intEnvOrDefault("RCON_PORT", 25575),
		RCONPass:            strings.TrimSpace(os.Getenv("RCON_PASS")),
		CommandPrefix:       envOrDefault("COMMAND_PREFIX", "!"),
		DataDir:             envOrDefault("DATA_DIR", "./data"),
	}

	cfg.AllowedMXIDs = parseAllowlist(os.Getenv("ALLOWED_MXIDS"))

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) SyncTokenPath() string {
	return filepath.Join(c.DataDir, "sync.token")
}

func (c Config) AccessTokenPath() string {
	return filepath.Join(c.DataDir, "matrix_access.token")
}

func (c Config) validate() error {
	if c.MatrixHomeserver == "" {
		return errors.New("MATRIX_HOMESERVER is required")
	}
	if c.MatrixRoomID == "" {
		return errors.New("MATRIX_ROOM_ID is required")
	}
	if len(c.AllowedMXIDs) == 0 {
		return errors.New("ALLOWED_MXIDS must include at least one MXID")
	}
	if c.MatrixAccessToken == "" {
		if c.MatrixUser == "" || c.MatrixPassword == "" {
			return errors.New("set MATRIX_ACCESS_TOKEN or both MATRIX_USER and MATRIX_PASSWORD")
		}
	}
	if c.RCONPass == "" {
		return errors.New("RCON_PASS is required")
	}
	if c.RCONPort <= 0 {
		return fmt.Errorf("invalid RCON_PORT: %d", c.RCONPort)
	}
	if strings.TrimSpace(c.CommandPrefix) == "" {
		return errors.New("COMMAND_PREFIX must not be empty")
	}
	return nil
}

func parseAllowlist(input string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, raw := range strings.Split(input, ",") {
		mxid := strings.TrimSpace(raw)
		if mxid == "" {
			continue
		}
		out[mxid] = struct{}{}
	}
	return out
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intEnvOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

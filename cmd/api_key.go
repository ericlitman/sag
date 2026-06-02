package cmd

import (
	"fmt"
	"os"
	"strings"
)

func ensureAPIKey() error {
	if cfg.APIKey == "" {
		key, err := resolveAPIKeyFromFile(cfg.Provider)
		if err != nil {
			return err
		}
		cfg.APIKey = key
	}
	if cfg.APIKey == "" {
		cfg.APIKey = resolveAPIKeyFromEnv(cfg.Provider)
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("missing %s API key (set --api-key, --api-key-file, or %s)", providerDisplayName(cfg.Provider), primaryAPIKeyEnv(cfg.Provider))
	}
	return nil
}

func resolveAPIKeyFromFile(provider string) (string, error) {
	path := cfg.APIKeyFile
	if path == "" {
		path = providerAPIKeyFileEnv(provider)
	}
	if path == "" {
		path = os.Getenv("SAG_API_KEY_FILE")
	}
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read api key file: %w", err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("api key file %q is empty", path)
	}
	return key, nil
}

func resolveAPIKeyFromEnv(provider string) string {
	for _, name := range apiKeyEnvNames(provider) {
		if env := strings.TrimSpace(os.Getenv(name)); env != "" {
			return env
		}
	}
	return ""
}

func providerAPIKeyFileEnv(provider string) string {
	switch provider {
	case providerFishAudio:
		if env := strings.TrimSpace(os.Getenv("FISH_AUDIO_API_KEY_FILE")); env != "" {
			return env
		}
		if env := strings.TrimSpace(os.Getenv("FISH_API_KEY_FILE")); env != "" {
			return env
		}
	default:
		if env := strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY_FILE")); env != "" {
			return env
		}
	}
	return ""
}

func apiKeyEnvNames(provider string) []string {
	switch provider {
	case providerFishAudio:
		return []string{"FISH_AUDIO_API_KEY", "FISH_API_KEY", "SAG_API_KEY"}
	default:
		return []string{"ELEVENLABS_API_KEY", "SAG_API_KEY"}
	}
}

func primaryAPIKeyEnv(provider string) string {
	switch provider {
	case providerFishAudio:
		return "FISH_AUDIO_API_KEY"
	default:
		return "ELEVENLABS_API_KEY"
	}
}

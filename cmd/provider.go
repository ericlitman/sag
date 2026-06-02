package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/steipete/sag/internal/elevenlabs"
	"github.com/steipete/sag/internal/fishaudio"
	"github.com/steipete/sag/internal/tts"
)

const (
	providerElevenLabs = "elevenlabs"
	providerFishAudio  = "fish"
)

func applyProviderEnv(cmd *cobra.Command) error {
	rootFlags := cmd.Root().PersistentFlags()
	if !rootFlags.Changed("provider") {
		if env := strings.TrimSpace(os.Getenv("SAG_PROVIDER")); env != "" {
			cfg.Provider = env
		}
	}
	provider, err := normalizeProvider(cfg.Provider)
	if err != nil {
		return err
	}
	cfg.Provider = provider

	if !rootFlags.Changed("base-url") && strings.TrimSpace(cfg.BaseURL) == "" {
		switch provider {
		case providerElevenLabs:
			cfg.BaseURL = strings.TrimSpace(os.Getenv("ELEVENLABS_BASE_URL"))
		case providerFishAudio:
			cfg.BaseURL = strings.TrimSpace(os.Getenv("FISH_AUDIO_BASE_URL"))
		}
	}
	return nil
}

func normalizeProvider(provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", providerElevenLabs, "eleven", "11labs":
		return providerElevenLabs, nil
	case providerFishAudio, "fishaudio", "fish-audio", "fish_audio":
		return providerFishAudio, nil
	default:
		return "", fmt.Errorf("unknown provider %q; choose elevenlabs or fish", provider)
	}
}

func newProviderClient(provider, apiKey, baseURL string) (tts.Client, error) {
	switch provider {
	case providerElevenLabs:
		return elevenlabs.NewClient(apiKey, baseURL), nil
	case providerFishAudio:
		return fishaudio.NewClient(apiKey, baseURL), nil
	default:
		return nil, fmt.Errorf("unknown provider %q; choose elevenlabs or fish", provider)
	}
}

func defaultModelForProvider(provider string) string {
	switch provider {
	case providerFishAudio:
		return "s2-pro"
	default:
		return "eleven_v3"
	}
}

func providerDisplayName(provider string) string {
	switch provider {
	case providerFishAudio:
		return "Fish Audio"
	default:
		return "ElevenLabs"
	}
}

func defaultVoiceFromEnv(provider string) (string, bool) {
	switch provider {
	case providerFishAudio:
		if env := strings.TrimSpace(os.Getenv("FISH_AUDIO_VOICE_ID")); env != "" {
			return env, true
		}
		if env := strings.TrimSpace(os.Getenv("FISH_AUDIO_REFERENCE_ID")); env != "" {
			return env, true
		}
	default:
		if env := strings.TrimSpace(os.Getenv("ELEVENLABS_VOICE_ID")); env != "" {
			return env, true
		}
	}
	if env := strings.TrimSpace(os.Getenv("SAG_VOICE_ID")); env != "" {
		return env, true
	}
	return "", false
}

func defaultVoiceEnvName(provider string) string {
	switch provider {
	case providerFishAudio:
		return "FISH_AUDIO_VOICE_ID"
	default:
		return "ELEVENLABS_VOICE_ID"
	}
}

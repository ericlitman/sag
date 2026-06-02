package tts

import (
	"context"
	"io"
)

// Client is the provider-neutral surface used by the CLI.
type Client interface {
	ListVoices(ctx context.Context) ([]Voice, error)
	SearchVoices(ctx context.Context, search string, limit int) ([]Voice, error)
	GetVoice(ctx context.Context, voiceID string) (Voice, error)
	StreamTTS(ctx context.Context, voiceID string, payload Request, latency int) (io.ReadCloser, error)
	ConvertTTS(ctx context.Context, voiceID string, payload Request) ([]byte, error)
}

// Voice represents a provider voice/model entry.
type Voice struct {
	VoiceID     string            `json:"voice_id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
	PreviewURL  string            `json:"preview_url"`
}

// Request configures a text-to-speech request payload.
type Request struct {
	Text                   string         `json:"text"`
	ModelID                string         `json:"model_id,omitempty"`
	VoiceSettings          *VoiceSettings `json:"voice_settings,omitempty"`
	OutputFormat           string         `json:"output_format,omitempty"`
	Seed                   *uint32        `json:"seed,omitempty"`
	ApplyTextNormalization string         `json:"apply_text_normalization,omitempty"`
	LanguageCode           string         `json:"language_code,omitempty"`
}

// VoiceSettings tunes synthesis parameters for a request.
type VoiceSettings struct {
	Stability       *float64 `json:"stability,omitempty"`
	SimilarityBoost *float64 `json:"similarity_boost,omitempty"`
	Style           *float64 `json:"style,omitempty"`
	UseSpeakerBoost *bool    `json:"use_speaker_boost,omitempty"`
	Speed           *float64 `json:"speed,omitempty"`
}

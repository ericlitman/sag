package fishaudio

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/steipete/sag/internal/tts"
)

func TestNewClientDefaultsBase(t *testing.T) {
	c := NewClient("key", "")
	if c.baseURL != "https://api.fish.audio" {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
}

func TestSearchVoicesUsesModelCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		if got := r.URL.Query().Get("title"); got != "Sarah" {
			t.Fatalf("unexpected title query: %q", got)
		}
		if got := r.URL.Query().Get("page_size"); got != "2" {
			t.Fatalf("unexpected page_size: %q", got)
		}
		_, _ = w.Write([]byte(`{"total":1,"items":[{"_id":"fish-sarah","type":"tts","title":"Sarah","description":"Warm voice","visibility":"public","state":"trained","tags":["english"],"languages":["en"]}],"has_more":false}`))
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	voices, err := c.SearchVoices(context.Background(), "Sarah", 2)
	if err != nil {
		t.Fatalf("SearchVoices error: %v", err)
	}
	if len(voices) != 1 {
		t.Fatalf("expected one voice, got %+v", voices)
	}
	if voices[0].VoiceID != "fish-sarah" || voices[0].Name != "Sarah" || voices[0].Category != "tts/public" {
		t.Fatalf("unexpected voice mapping: %+v", voices[0])
	}
	if voices[0].Labels["state"] != "trained" || voices[0].Labels["language1"] != "en" {
		t.Fatalf("unexpected labels: %+v", voices[0].Labels)
	}
}

func TestStreamTTSBuildsFishRequest(t *testing.T) {
	speed := 1.1
	seed := uint32(42)
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		if got := r.Header.Get("model"); got != "s2-pro" {
			t.Fatalf("unexpected model header: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte("audio-data"))
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	rc, err := c.StreamTTS(context.Background(), "fish-sarah", tts.Request{
		Text:                   "hello",
		ModelID:                "s2-pro",
		OutputFormat:           "mp3_44100_128",
		Seed:                   &seed,
		ApplyTextNormalization: "off",
		VoiceSettings:          &tts.VoiceSettings{Speed: &speed},
	}, 2)
	if err != nil {
		t.Fatalf("StreamTTS error: %v", err)
	}
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(rc)
	if string(data) != "audio-data" {
		t.Fatalf("unexpected body: %q", string(data))
	}
	if gotBody["reference_id"] != "fish-sarah" {
		t.Fatalf("expected reference_id, got %v", gotBody["reference_id"])
	}
	if gotBody["format"] != "mp3" || gotBody["sample_rate"] != float64(44100) || gotBody["mp3_bitrate"] != float64(128) {
		t.Fatalf("unexpected format fields: %+v", gotBody)
	}
	if gotBody["normalize"] != false || gotBody["latency"] != "balanced" || gotBody["seed"] != float64(42) {
		t.Fatalf("unexpected controls: %+v", gotBody)
	}
	prosody, ok := gotBody["prosody"].(map[string]any)
	if !ok || prosody["speed"] != 1.1 || prosody["normalize_loudness"] != true {
		t.Fatalf("unexpected prosody: %+v", gotBody["prosody"])
	}
}

func TestStreamTTSErrorIncludesStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL)
	_, err := c.StreamTTS(context.Background(), "fish-sarah", tts.Request{Text: "hello"}, 0)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 error, got %v", err)
	}
}

func TestFishFormatMapsSagFormats(t *testing.T) {
	format, sampleRate, mp3Bitrate, opusBitrate, err := fishFormat("opus_48000_64")
	if err != nil {
		t.Fatalf("fishFormat error: %v", err)
	}
	if format != "opus" || *sampleRate != 48000 || mp3Bitrate != nil || *opusBitrate != 64 {
		t.Fatalf("unexpected opus mapping: %s %v %v %v", format, sampleRate, mp3Bitrate, opusBitrate)
	}

	format, sampleRate, _, _, err = fishFormat("pcm_44100")
	if err != nil {
		t.Fatalf("fishFormat pcm error: %v", err)
	}
	if format != "wav" || *sampleRate != 44100 {
		t.Fatalf("unexpected wav mapping: %s %v", format, sampleRate)
	}
}

func TestBuildFishRequestRejectsNegativeLatency(t *testing.T) {
	_, err := buildFishRequest("fish-sarah", tts.Request{Text: "hello"}, -1)
	if err == nil || !strings.Contains(err.Error(), "latency tier") {
		t.Fatalf("expected latency error, got %v", err)
	}
}

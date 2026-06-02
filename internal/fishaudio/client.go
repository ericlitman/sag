package fishaudio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/steipete/sag/internal/tts"
)

// Client talks to the Fish Audio HTTP API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient returns a Client configured with the given API key and base URL.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.fish.audio"
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type modelListResponse struct {
	Items   []modelEntity `json:"items"`
	HasMore bool          `json:"has_more"`
}

type modelEntity struct {
	ID          string   `json:"_id"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	State       string   `json:"state"`
	Visibility  string   `json:"visibility"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Languages   []string `json:"languages"`
}

// ListVoices fetches available Fish Audio voice models.
func (c *Client) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	return c.listModels(ctx, "", 100)
}

// SearchVoices finds Fish Audio voice models using the title filter.
func (c *Client) SearchVoices(ctx context.Context, search string, limit int) ([]tts.Voice, error) {
	search = strings.TrimSpace(search)
	if search == "" {
		return c.ListVoices(ctx)
	}
	return c.listModels(ctx, search, limit)
}

// GetVoice fetches metadata for a specific Fish Audio voice model.
func (c *Client) GetVoice(ctx context.Context, voiceID string) (tts.Voice, error) {
	voiceID = strings.TrimSpace(voiceID)
	if voiceID == "" {
		return tts.Voice{}, fmt.Errorf("voice ID is required")
	}
	models, err := c.listModels(ctx, "", 100)
	if err != nil {
		return tts.Voice{}, err
	}
	for _, model := range models {
		if model.VoiceID == voiceID {
			return model, nil
		}
	}
	return tts.Voice{}, fmt.Errorf("fish voice model %q not found", voiceID)
}

func (c *Client) listModels(ctx context.Context, title string, limit int) ([]tts.Voice, error) {
	pageSize := 100
	if limit > 0 && limit < pageSize {
		pageSize = limit
	}
	voices := make([]tts.Voice, 0, pageSize)
	for page := 1; ; page++ {
		u, err := url.Parse(c.baseURL)
		if err != nil {
			return nil, err
		}
		u.Path = path.Join(u.Path, "/model")
		q := u.Query()
		q.Set("page_size", fmt.Sprint(pageSize))
		q.Set("page_number", fmt.Sprint(page))
		if title != "" {
			q.Set("title", title)
		}
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("list fish voices failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var body modelListResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		_ = resp.Body.Close()

		for _, item := range body.Items {
			voices = append(voices, modelToVoice(item))
			if limit > 0 && len(voices) >= limit {
				return voices[:limit], nil
			}
		}
		if !body.HasMore || len(body.Items) == 0 {
			return voices, nil
		}
	}
}

func modelToVoice(model modelEntity) tts.Voice {
	labels := map[string]string{}
	if model.State != "" {
		labels["state"] = model.State
	}
	if model.Visibility != "" {
		labels["visibility"] = model.Visibility
	}
	for i, tag := range model.Tags {
		if tag != "" {
			labels[fmt.Sprintf("tag%d", i+1)] = tag
		}
	}
	for i, language := range model.Languages {
		if language != "" {
			labels[fmt.Sprintf("language%d", i+1)] = language
		}
	}
	category := strings.Trim(strings.Join([]string{model.Type, model.Visibility}, "/"), "/")
	return tts.Voice{
		VoiceID:     model.ID,
		Name:        model.Title,
		Category:    category,
		Description: model.Description,
		Labels:      labels,
	}
}

type ttsRequest struct {
	Text                      string   `json:"text"`
	ReferenceID               string   `json:"reference_id,omitempty"`
	Prosody                   *prosody `json:"prosody,omitempty"`
	Format                    string   `json:"format,omitempty"`
	SampleRate                *int     `json:"sample_rate,omitempty"`
	MP3Bitrate                *int     `json:"mp3_bitrate,omitempty"`
	OpusBitrate               *int     `json:"opus_bitrate,omitempty"`
	Normalize                 *bool    `json:"normalize,omitempty"`
	Latency                   string   `json:"latency,omitempty"`
	Seed                      *uint32  `json:"seed,omitempty"`
	Temperature               *float64 `json:"temperature,omitempty"`
	TopP                      *float64 `json:"top_p,omitempty"`
	RepetitionPenalty         *float64 `json:"repetition_penalty,omitempty"`
	ConditionOnPreviousChunks *bool    `json:"condition_on_previous_chunks,omitempty"`
}

type prosody struct {
	Speed             *float64 `json:"speed,omitempty"`
	NormalizeLoudness *bool    `json:"normalize_loudness,omitempty"`
}

// StreamTTS requests audio for text-to-speech and returns the response body.
func (c *Client) StreamTTS(ctx context.Context, voiceID string, payload tts.Request, latency int) (io.ReadCloser, error) {
	return c.doTTS(ctx, voiceID, payload, latency)
}

// ConvertTTS downloads the full audio before returning.
func (c *Client) ConvertTTS(ctx context.Context, voiceID string, payload tts.Request) ([]byte, error) {
	resp, err := c.doTTS(ctx, voiceID, payload, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Close()
	}()
	data, err := io.ReadAll(resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) doTTS(ctx context.Context, voiceID string, payload tts.Request, latency int) (io.ReadCloser, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "/v1/tts")

	fishPayload, err := buildFishRequest(voiceID, payload, latency)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := json.Marshal(fishPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(payload.ModelID)
	if model == "" {
		model = "s2-pro"
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("model", model)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer func() {
			_ = resp.Body.Close()
		}()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fish TTS failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return resp.Body, nil
}

func buildFishRequest(voiceID string, payload tts.Request, latency int) (ttsRequest, error) {
	if strings.TrimSpace(payload.Text) == "" {
		return ttsRequest{}, fmt.Errorf("text is required")
	}
	if strings.TrimSpace(voiceID) == "" {
		return ttsRequest{}, fmt.Errorf("fish reference_id is required")
	}
	format, sampleRate, mp3Bitrate, opusBitrate, err := fishFormat(payload.OutputFormat)
	if err != nil {
		return ttsRequest{}, err
	}
	req := ttsRequest{
		Text:        payload.Text,
		ReferenceID: voiceID,
		Format:      format,
		SampleRate:  sampleRate,
		MP3Bitrate:  mp3Bitrate,
		OpusBitrate: opusBitrate,
		Seed:        payload.Seed,
	}
	if payload.VoiceSettings != nil && payload.VoiceSettings.Speed != nil {
		req.Prosody = &prosody{Speed: payload.VoiceSettings.Speed}
		normalizeLoudness := true
		req.Prosody.NormalizeLoudness = &normalizeLoudness
	}
	switch strings.ToLower(strings.TrimSpace(payload.ApplyTextNormalization)) {
	case "", "auto", "on":
		if payload.ApplyTextNormalization != "" {
			v := true
			req.Normalize = &v
		}
	case "off":
		v := false
		req.Normalize = &v
	default:
		return ttsRequest{}, fmt.Errorf("unsupported Fish normalize value %q", payload.ApplyTextNormalization)
	}
	switch {
	case latency < 0:
		return ttsRequest{}, fmt.Errorf("latency tier must be >= 0")
	case latency == 0:
	default:
		req.Latency = "balanced"
	}
	return req, nil
}

func fishFormat(outputFormat string) (format string, sampleRate *int, mp3Bitrate *int, opusBitrate *int, err error) {
	outputFormat = strings.TrimSpace(strings.ToLower(outputFormat))
	if outputFormat == "" {
		return "mp3", nil, nil, nil, nil
	}
	parts := strings.Split(outputFormat, "_")
	format = parts[0]
	switch format {
	case "mp3", "opus", "pcm", "wav":
	default:
		return "", nil, nil, nil, fmt.Errorf("unsupported Fish output format %q", outputFormat)
	}
	if outputFormat == "pcm_44100" {
		format = "wav"
	}
	if len(parts) >= 2 {
		v, parseErr := strconv.Atoi(parts[1])
		if parseErr != nil {
			return "", nil, nil, nil, fmt.Errorf("invalid Fish sample rate in %q", outputFormat)
		}
		sampleRate = &v
	}
	if len(parts) >= 3 {
		v, parseErr := strconv.Atoi(parts[2])
		if parseErr != nil {
			return "", nil, nil, nil, fmt.Errorf("invalid Fish bitrate in %q", outputFormat)
		}
		switch format {
		case "mp3":
			mp3Bitrate = &v
		case "opus":
			opusBitrate = &v
		}
	}
	return format, sampleRate, mp3Bitrate, opusBitrate, nil
}

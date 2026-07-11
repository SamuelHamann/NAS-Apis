// Package gemini is a minimal client for the Gemini API's generateContent
// endpoint — just enough to send an image with a text prompt and get back a
// text response. It talks to the REST API directly rather than depending on
// the full Google GenAI SDK, matching this repo's preference for a small
// dependency footprint.
package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// ErrNoAPIKey is returned by GenerateFromImage when the Client was
// constructed with an empty API key.
var ErrNoAPIKey = errors.New("gemini: no API key configured")

// Client calls the Gemini generateContent REST API for a single model.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// New creates a Client for the given model. An empty apiKey is allowed —
// GenerateFromImage simply fails with ErrNoAPIKey on every call — so callers
// can construct one unconditionally from config at startup.
func New(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type generateRequest struct {
	Contents         []requestContent  `json:"contents"`
	GenerationConfig *generationConfig `json:"generationConfig,omitempty"`
}

type generationConfig struct {
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
	ResponseSchema   *Schema `json:"responseSchema,omitempty"`
}

// Schema is a deliberately partial mirror of Gemini's structured-output
// schema format (itself a restricted subset of OpenAPI 3.0's Schema
// object) — only the fields this client actually uses. Passed to
// GenerateFromImage to force the model's response to be JSON matching this
// shape, via generationConfig.responseMimeType="application/json".
type Schema struct {
	Type       string             `json:"type"` // "OBJECT", "ARRAY", "STRING", "NUMBER", "BOOLEAN"
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Nullable   bool               `json:"nullable,omitempty"`
}

type requestContent struct {
	Parts []requestPart `json:"parts"`
}

type requestPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64-encoded
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GenerateFromImage sends prompt alongside an inline image (imageData,
// labelled with mimeType, e.g. "image/jpeg") to Gemini and returns the
// model's response as text — the first text part of the first candidate.
// Images are sent inline (base64, in the request body) rather than via the
// separate Files API, which only makes sense for something receipt-photo-
// sized.
//
// When schema is non-nil, the response is forced to be JSON matching it
// (generationConfig.responseMimeType="application/json" +
// responseSchema) — the returned string is then JSON text ready to
// json.Unmarshal, not prose. Pass nil for a plain-text response.
func (c *Client) GenerateFromImage(ctx context.Context, prompt string, imageData []byte, mimeType string, schema *Schema) (string, error) {
	if c.apiKey == "" {
		return "", ErrNoAPIKey
	}

	payload := generateRequest{
		Contents: []requestContent{{
			Parts: []requestPart{
				{Text: prompt},
				{InlineData: &inlineData{
					MimeType: mimeType,
					Data:     base64.StdEncoding.EncodeToString(imageData),
				}},
			},
		}},
	}
	if schema != nil {
		payload.GenerationConfig = &generationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
		}
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gemini: encode request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Header rather than a ?key=... query param, so the key never ends up
	// in a proxy's/load balancer's URL-based access log.
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed generateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: response had no text content")
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

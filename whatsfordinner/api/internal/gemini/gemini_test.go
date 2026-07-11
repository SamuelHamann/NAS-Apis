package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &Client{
		apiKey:     "test-key",
		model:      "gemini-3.1-flash-lite",
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
}

func TestGenerateFromImageSendsRequestAndParsesResponse(t *testing.T) {
	var gotPath, gotAPIKey, gotContentType string
	var gotBody generateRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-goog-api-key")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]any{{"text": "Milk x1 $3.50"}}}},
			},
		})
	})

	got, err := client.GenerateFromImage(context.Background(), "list items", []byte("fake-image-bytes"), "image/jpeg", nil)
	if err != nil {
		t.Fatalf("GenerateFromImage: %v", err)
	}
	if got != "Milk x1 $3.50" {
		t.Errorf("expected %q, got %q", "Milk x1 $3.50", got)
	}

	if gotPath != "/models/gemini-3.1-flash-lite:generateContent" {
		t.Errorf("unexpected request path %q", gotPath)
	}
	if gotAPIKey != "test-key" {
		t.Errorf("expected api key header %q, got %q", "test-key", gotAPIKey)
	}
	if !strings.HasPrefix(gotContentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", gotContentType)
	}
	if len(gotBody.Contents) != 1 || len(gotBody.Contents[0].Parts) != 2 {
		t.Fatalf("unexpected request body shape: %+v", gotBody)
	}
	if gotBody.Contents[0].Parts[0].Text != "list items" {
		t.Errorf("expected prompt part %q, got %q", "list items", gotBody.Contents[0].Parts[0].Text)
	}
	inline := gotBody.Contents[0].Parts[1].InlineData
	if inline == nil || inline.MimeType != "image/jpeg" {
		t.Fatalf("expected inline image data with mime type image/jpeg, got %+v", inline)
	}
	if gotBody.GenerationConfig != nil {
		t.Errorf("expected no generationConfig when schema is nil, got %+v", gotBody.GenerationConfig)
	}
}

func TestGenerateFromImageWithSchemaSetsGenerationConfig(t *testing.T) {
	var gotBody generateRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]any{{"text": `{"items":[]}`}}}},
			},
		})
	})

	schema := &Schema{
		Type: "OBJECT",
		Properties: map[string]*Schema{
			"items": {Type: "ARRAY", Items: &Schema{Type: "STRING"}},
		},
		Required: []string{"items"},
	}

	got, err := client.GenerateFromImage(context.Background(), "extract items", []byte("x"), "image/jpeg", schema)
	if err != nil {
		t.Fatalf("GenerateFromImage: %v", err)
	}
	if got != `{"items":[]}` {
		t.Errorf("expected raw JSON text %q, got %q", `{"items":[]}`, got)
	}

	if gotBody.GenerationConfig == nil {
		t.Fatal("expected generationConfig to be set when a schema is passed")
	}
	if gotBody.GenerationConfig.ResponseMimeType != "application/json" {
		t.Errorf("expected responseMimeType application/json, got %q", gotBody.GenerationConfig.ResponseMimeType)
	}
	if gotBody.GenerationConfig.ResponseSchema == nil || gotBody.GenerationConfig.ResponseSchema.Type != "OBJECT" {
		t.Errorf("expected the schema to round-trip into the request, got %+v", gotBody.GenerationConfig.ResponseSchema)
	}
}

func TestGenerateFromImageNoAPIKey(t *testing.T) {
	client := &Client{model: "gemini-3.1-flash-lite", baseURL: "http://unused", httpClient: http.DefaultClient}

	_, err := client.GenerateFromImage(context.Background(), "prompt", []byte("x"), "image/jpeg", nil)
	if err != ErrNoAPIKey {
		t.Errorf("expected ErrNoAPIKey, got %v", err)
	}
}

func TestGenerateFromImageNonOKStatus(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	})

	_, err := client.GenerateFromImage(context.Background(), "prompt", []byte("x"), "image/jpeg", nil)
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}

func TestGenerateFromImageEmptyCandidates(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []map[string]any{}})
	})

	_, err := client.GenerateFromImage(context.Background(), "prompt", []byte("x"), "image/jpeg", nil)
	if err == nil {
		t.Fatal("expected an error when the response has no candidates")
	}
}

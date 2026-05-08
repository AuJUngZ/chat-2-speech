package tts

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListVoicesReturnsSlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"voices":[{"name":"voice1"},{"name":"voice2"}]}`))
	}))
	defer server.Close()

	cfg := &EngineConfig{
		SpeechRateMultiplier: 1.0,
		CloudTTSEnabled:      true,
		CloudAPIKey:          "test-key",
		cloudBaseURL:         server.URL,
	}
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	voices, err := engine.ListVoices("en")
	if err != nil {
		t.Fatalf("ListVoices failed: %v", err)
	}

	if voices == nil {
		t.Error("ListVoices returned nil slice")
	}
}

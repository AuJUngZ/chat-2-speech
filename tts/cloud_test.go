package tts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudBackendSpeaks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"audioContent":"bm90LXJlYWwtYXVkaW8="}`)) // "not-real-audio" in base64
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

	if ce, ok := engine.(*cloudEngine); ok {
		ce.skipPlayback = true
	}

	err = engine.Speak(context.Background(), "hello", "en")
	if err != nil {
		t.Errorf("Speak failed: %v", err)
	}
}

func TestCloudBackendUsesAPIKey(t *testing.T) {
	var receivedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.URL.Query().Get("key")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &EngineConfig{
		SpeechRateMultiplier: 1.0,
		CloudTTSEnabled:      true,
		CloudAPIKey:          "my-secret-key",
		cloudBaseURL:         server.URL,
	}
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	if ce, ok := engine.(*cloudEngine); ok {
		ce.skipPlayback = true
	}

	engine.Speak(context.Background(), "test", "en")

	if receivedKey == "" {
		t.Error("Cloud backend did not send API key in request")
	}
}

func TestCloudListVoicesReturnsHardcodedList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"voices":[{"name":"voice1","naturalSampleRateHertz":24000},{"name":"voice2","naturalSampleRateHertz":48000}]}`))
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

	if len(voices) < 2 {
		t.Errorf("ListVoices returned %d voices, want at least 2", len(voices))
	}
}

func TestOSBackendSpeaks(t *testing.T) {
	cfg := &EngineConfig{
		SpeechRateMultiplier: 1.0,
		CloudAPIKey:          "",
	}
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine with OS backend failed: %v", err)
	}

	if engine == nil {
		t.Error("NewEngine returned nil engine")
	}

	_ = engine.Speak(context.Background(), "test", "en")
}

func TestCloudBackendFallsBack(t *testing.T) {
	// Server returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &EngineConfig{
		SpeechRateMultiplier: 1.0,
		CloudTTSEnabled:      true,
		CloudAPIKey:          "test-key",
		cloudBaseURL:         server.URL,
	}
	engine, _ := NewEngine(cfg)

	// Since windowsEngine might fail on this environment, we'll mock the fallback
	if ce, ok := engine.(*cloudEngine); ok {
		ce.skipPlayback = true
		mockFallback := &mockEngine{}
		ce.fallback = mockFallback
		
		err := engine.Speak(context.Background(), "test", "en")
		if err != nil {
			t.Errorf("Speak failed even with fallback: %v", err)
		}
		if !mockFallback.speakCalled {
			t.Error("Fallback engine was not called on cloud failure")
		}
	}
}

type mockEngine struct {
	speakCalled bool
}

func (m *mockEngine) Speak(ctx context.Context, text, lang string) error {
	m.speakCalled = true
	return nil
}

func (m *mockEngine) EstimateDuration(text, lang string) float64 { return 0 }
func (m *mockEngine) ListVoices(lang string) ([]string, error)  { return nil, nil }
func (m *mockEngine) SetErrorCallback(func(err error)) {}

func TestNewEngineRespectsEnabledFlag(t *testing.T) {
	cfg := &EngineConfig{
		SpeechRateMultiplier: 1.0,
		CloudTTSEnabled:      false,
		CloudAPIKey:          "some-key",
	}
	engine, _ := NewEngine(cfg)

	// Since we can't easily check the type of the returned engine because it's an interface,
	// and we don't want to export cloudEngine/windowsEngine types more than necessary,
	// we'll rely on the behavior or just check if it's the expected interface implementation
	// if we can. In this case, cloudEngine has a baseURL field which windowsEngine doesn't.
	// But let's just check the NewEngine logic for now.
	
	if _, ok := engine.(*cloudEngine); ok {
		t.Error("NewEngine returned cloudEngine even though CloudTTSEnabled is false")
	}

	cfg.CloudTTSEnabled = true
	engine, _ = NewEngine(cfg)
	if _, ok := engine.(*cloudEngine); !ok {
		t.Error("NewEngine did not return cloudEngine even though CloudTTSEnabled is true and API key is present")
	}
}

func TestNewEngineWithCloudBackendConfigured(t *testing.T) {
	cfg := &EngineConfig{
		SpeechRateMultiplier: 1.0,
		CloudTTSEnabled:      true,
		CloudAPIKey:          "fake-api-key",
	}
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine with cloud backend failed: %v", err)
	}

	if engine == nil {
		t.Error("NewEngine returned nil engine")
	}
}

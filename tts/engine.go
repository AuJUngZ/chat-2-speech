package tts

import (
	"context"
)

type EngineConfig struct {
	SpeechRateMultiplier float64
	CloudTTSEnabled      bool
	CloudAPIKey          string
	ThaiVoiceName        string
	EnglishVoiceName     string
	cloudBaseURL         string
}

type Engine interface {
	Speak(ctx context.Context, text, lang string) error
	EstimateDuration(text, lang string) float64
	ListVoices(lang string) ([]string, error)
}

type durationCalculator struct {
	charsPerSec float64
	multiplier  float64
}

func (dc *durationCalculator) seconds(text string) float64 {
	return float64(len(text)) / (dc.charsPerSec * dc.multiplier)
}

type engine struct {
	cfg EngineConfig
}

const (
	thaiCharsPerSec    = 150.0
	englishCharsPerSec = 130.0
)

func NewEngine(cfg *EngineConfig) (Engine, error) {
	if cfg.CloudTTSEnabled && cfg.CloudAPIKey != "" {
		ce := newCloudEngine(*cfg)
		if c, ok := ce.(*cloudEngine); ok {
			c.fallback = newWindowsEngine(*cfg)
		}
		return ce, nil
	}
	return newWindowsEngine(*cfg), nil
}

func newWindowsEngine(cfg EngineConfig) Engine {
	return &windowsEngine{cfg: cfg}
}

func (e *engine) EstimateDuration(text, lang string) float64 {
	return (&durationCalculator{
		charsPerSec: charsPerSecForLang(lang),
		multiplier:  e.cfg.SpeechRateMultiplier,
	}).seconds(text)
}

func charsPerSecForLang(lang string) float64 {
	switch lang {
	case "th":
		return thaiCharsPerSec
	case "en":
		return englishCharsPerSec
	default:
		return englishCharsPerSec
	}
}

func (e *engine) Speak(ctx context.Context, text, lang string) error {
	return nil
}

func (e *engine) ListVoices(lang string) ([]string, error) {
	return nil, nil
}

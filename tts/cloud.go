package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const geminiTTSURL = "https://generativelanguage.googleapis.com"

type voiceInfo struct {
	name                   string
	naturalSampleRateHertz int
}

type cloudEngine struct {
	cfg           EngineConfig
	client        *http.Client
	baseURL       string
	skipPlayback  bool
	voiceMetadata map[string]voiceInfo
	fallback      Engine
	errCallback   func(err error)
}

type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voice_name,omitempty"`
}

type VoiceConfig struct {
	PrebuiltVoiceConfig PrebuiltVoiceConfig `json:"prebuilt_voice_config,omitempty"`
}

type SpeechConfig struct {
	VoiceConfig VoiceConfig `json:"voice_config,omitempty"`
}

type GenerateContentConfig struct {
	ResponseModalities []string      `json:"response_modalities,omitempty"`
	SpeechConfig       SpeechConfig `json:"speech_config,omitempty"`
}

func newCloudEngine(cfg EngineConfig) Engine {
	baseURL := cfg.cloudBaseURL
	if baseURL == "" {
		baseURL = geminiTTSURL
	}
	return &cloudEngine{
		cfg: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL:       baseURL,
		voiceMetadata: make(map[string]voiceInfo),
	}
}

func (e *cloudEngine) Speak(ctx context.Context, text, lang string) error {
	err := e.speak(ctx, text, lang)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fmt.Printf("Gemini TTS failed, falling back to OS-native: %v\n", err)
		if e.errCallback != nil {
			e.errCallback(err)
		}
		if e.fallback != nil {
			return e.fallback.Speak(ctx, text, lang)
		}
	}
	return err
}

func (e *cloudEngine) SetErrorCallback(cb func(err error)) {
	e.errCallback = cb
}

func (e *cloudEngine) speak(ctx context.Context, text, lang string) error {
	voiceName := e.voiceNameForLang(lang)
	if voiceName == "" {
		voiceName = "Kore"
	}

	reqBody := map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"parts": []interface{}{
					map[string]string{"text": text},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseModalities": []string{"AUDIO"},
		},
		"safetySettings": []interface{}{
			map[string]string{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"},
			map[string]string{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"},
			map[string]string{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"},
			map[string]string{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"},
		},
	}

	if voiceName != "" {
		reqBody["generationConfig"] = map[string]interface{}{
			"responseModalities": []string{"AUDIO"},
			"speechConfig": map[string]interface{}{
				"voiceConfig": map[string]interface{}{
					"prebuiltVoiceConfig": map[string]string{"voiceName": voiceName},
				},
			},
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	model := e.cfg.GeminiModel
	if model == "" {
		model = "gemini-3.1-flash-tts-preview"
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", e.baseURL, model, e.cfg.CloudAPIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini TTS request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						Data     string `json:"data"`
						MimeType string `json:"mimeType"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Candidates) == 0 {
		return fmt.Errorf("no candidates in response")
	}
	if len(result.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("no parts in response")
	}
	inlineData := result.Candidates[0].Content.Parts[0].InlineData
	if inlineData.Data == "" {
		return fmt.Errorf("no audio data in response, mimeType: %s", inlineData.MimeType)
	}
	audioData, err := base64.StdEncoding.DecodeString(inlineData.Data)
	if err != nil {
		return fmt.Errorf("failed to decode audio data: %v", err)
	}

	return e.playAudio(ctx, audioData, inlineData.MimeType)
}

func (e *cloudEngine) playAudio(ctx context.Context, data []byte, mimeType string) error {
	if e.skipPlayback {
		return nil
	}
	tempDir := os.TempDir()
	var tempFile string
	if mimeType == "audio/wav" || mimeType == "audio/wave" {
		tempFile = filepath.Join(tempDir, fmt.Sprintf("chat-alert-tts-%d.wav", time.Now().UnixNano()))
		if err := os.WriteFile(tempFile, data, 0644); err != nil {
			return err
		}
	} else {
		tempFile = filepath.Join(tempDir, fmt.Sprintf("chat-alert-tts-%d.pcm", time.Now().UnixNano()))
		if err := os.WriteFile(tempFile, data, 0644); err != nil {
			return err
		}
		defer os.Remove(tempFile)

		wavFile := filepath.Join(tempDir, fmt.Sprintf("chat-alert-tts-%d.wav", time.Now().UnixNano()))
		if err := e.convertPcmToWav(tempFile, wavFile); err != nil {
			os.Remove(wavFile)
			return fmt.Errorf("failed to convert PCM to WAV: %v", err)
		}
		tempFile = wavFile
	}
	defer os.Remove(tempFile)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("Add-Type -AssemblyName presentationCore; $player = New-Object system.windows.media.mediaplayer; $player.open('%s'); $player.Play(); while($player.NaturalDuration.HasTimeSpan -eq $false){Start-Sleep -Milliseconds 10}; $dur = $player.NaturalDuration.TimeSpan; while($player.Position -lt $dur){Start-Sleep -Milliseconds 50}; Start-Sleep -Milliseconds 100", tempFile))

	return cmd.Run()
}

func (e *cloudEngine) convertPcmToWav(pcmFile, wavFile string) error {
	pcmFileEsc := strings.ReplaceAll(pcmFile, "'", "''")
	wavFileEsc := strings.ReplaceAll(wavFile, "'", "''")
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`Add-Type -TypeDefinition @"
using System;
using System.IO;
using System.Text;
public class WavWriter {
    public static void Write(string pcmPath, string wavPath, int sampleRate, int channels, int bitsPerSample) {
        byte[] pcmData = File.ReadAllBytes(pcmPath);
        int dataSize = pcmData.Length;
        using (var wav = new FileStream(wavPath, FileMode.Create)) {
            using (var bw = new BinaryWriter(wav)) {
                bw.Write(Encoding.ASCII.GetBytes("RIFF"));
                bw.Write(36 + dataSize);
                bw.Write(Encoding.ASCII.GetBytes("WAVE"));
                bw.Write(Encoding.ASCII.GetBytes("fmt "));
                bw.Write(16);
                bw.Write((short)1);
                bw.Write((short)channels);
                bw.Write(sampleRate);
                bw.Write(sampleRate * channels * bitsPerSample / 8);
                bw.Write((short)(channels * bitsPerSample / 8));
                bw.Write((short)bitsPerSample);
                bw.Write(Encoding.ASCII.GetBytes("data"));
                bw.Write(dataSize);
                bw.Write(pcmData);
            }
        }
    }
}
"@; [WavWriter]::Write('%s', '%s', 24000, 1, 16)`, pcmFileEsc, wavFileEsc))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("exit %v: %s", err, stderr.String())
	}
	return nil
}

func (e *cloudEngine) voiceNameForLang(lang string) string {
	if e.cfg.GeminiVoiceName != "" {
		return e.cfg.GeminiVoiceName
	}
	return "Kore"
}

func (e *cloudEngine) EstimateDuration(text, lang string) float64 {
	charsPerSec := charsPerSecForLang(lang)
	return (&durationCalculator{
		charsPerSec: charsPerSec,
		multiplier:  e.cfg.SpeechRateMultiplier,
	}).seconds(text)
}

var geminiVoices = []string{
	"Zephyr", "Puck", "Charon", "Kore", "Fenrir", "Leda", "Orus", "Aoede",
	"Callirrhoe", "Autonoe", "Enceladus", "Iapetus", "Umbriel", "Algieba",
	"Despina", "Erinome", "Algenib", "Rasalgethi", "Laomedeia", "Achernar",
	"Alnilam", "Schedar", "Gacrux", "Pulcherrima", "Achird", "Zubenelgenubi",
	"Vindemiatrix", "Sadachbia", "Sadaltager", "Sulafat",
}

func (e *cloudEngine) ListVoices(lang string) ([]string, error) {
	return geminiVoices, nil
}

func langMap(lang string) string {
	return lang
}
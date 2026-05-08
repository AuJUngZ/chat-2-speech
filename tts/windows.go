package tts

import (
	"context"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type windowsEngine struct {
	cfg EngineConfig
}

const CLSID_SpVoice = "SAPI.SpVoice"

func (e *windowsEngine) Speak(ctx context.Context, text, lang string) error {
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject(CLSID_SpVoice)
	if err != nil {
		return err
	}
	defer unknown.Release()

	voice, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer voice.Release()

	if voiceName := e.voiceNameForLang(lang); voiceName != "" {
		voicesVariant, err := oleutil.CallMethod(voice, "GetVoices")
		if err == nil {
			voices := voicesVariant.ToIDispatch()
			countVariant, _ := oleutil.GetProperty(voices, "Count")
			count := int(countVariant.Val)
			for i := 0; i < count; i++ {
				itemVariant, _ := oleutil.CallMethod(voices, "Item", i)
				item := itemVariant.ToIDispatch()
				descVariant, _ := oleutil.CallMethod(item, "GetDescription")
				if descVariant.ToString() == voiceName {
					oleutil.PutPropertyRef(voice, "Voice", item)
					itemVariant.Clear()
					break
				}
				itemVariant.Clear()
			}
			voicesVariant.Clear()
		}
	}

	rate := e.rateForLang(lang)
	oleutil.PutProperty(voice, "Rate", rate)

	// SPF_ASYNC = 1, SPF_PURGEBEFORESPEAK = 2
	// Start speaking asynchronously
	_, err = oleutil.CallMethod(voice, "Speak", text, 1)
	if err != nil {
		return err
	}

	// Poll for completion or cancellation
	for {
		select {
		case <-ctx.Done():
			// Interrupt speech: Speak empty string with SPF_PURGEBEFORESPEAK
			oleutil.CallMethod(voice, "Speak", "", 1|2)
			return ctx.Err()
		default:
			// WaitUntilDone returns true if finished, false if timeout
			res, err := oleutil.CallMethod(voice, "WaitUntilDone", 100) // 100ms timeout
			if err != nil {
				return err
			}
			if res.Val != 0 { // In COM, True is often -1 or non-zero
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (e *windowsEngine) ListVoices(lang string) ([]string, error) {
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject(CLSID_SpVoice)
	if err != nil {
		return nil, err
	}
	defer unknown.Release()

	voice, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	defer voice.Release()

	voicesVariant, err := oleutil.CallMethod(voice, "GetVoices")
	if err != nil {
		return nil, err
	}
	defer voicesVariant.Clear()

	return e.filterVoicesByLang(voicesVariant.ToIDispatch(), lang)
}

func (e *windowsEngine) EstimateDuration(text, lang string) float64 {
	return (&durationCalculator{
		charsPerSec: charsPerSecForLang(lang),
		multiplier:  e.cfg.SpeechRateMultiplier,
	}).seconds(text)
}

func (e *windowsEngine) voiceNameForLang(lang string) string {
	if lang == "th" {
		return e.cfg.ThaiVoiceName
	}
	return e.cfg.EnglishVoiceName
}

func (e *windowsEngine) rateForLang(lang string) int32 {
	// SAPI rate is from -10 to 10. 0 is normal.
	// SpeechRateMultiplier 1.0 -> 0
	// 0.5 -> -5? 2.0 -> 5?
	// Let's use a simple mapping: (multiplier - 1.0) * 10
	rate := (e.cfg.SpeechRateMultiplier - 1.0) * 10.0
	if rate > 10 {
		rate = 10
	}
	if rate < -10 {
		rate = -10
	}
	return int32(rate)
}

func (e *windowsEngine) filterVoicesByLang(voices *ole.IDispatch, lang string) ([]string, error) {
	countVariant, err := oleutil.GetProperty(voices, "Count")
	if err != nil {
		return nil, err
	}
	defer countVariant.Clear()
	count := int(countVariant.Val)

	wantLCID := e.lcidForLang(lang)
	var names []string
	for i := 0; i < count; i++ {
		itemVariant, err := oleutil.CallMethod(voices, "Item", i)
		if err != nil {
			continue
		}

		item := itemVariant.ToIDispatch()
		langVariant, err := oleutil.CallMethod(item, "GetLanguage")
		if err == nil {
			langID := int(langVariant.Val)
			if wantLCID == 0 || langID == wantLCID {
				descriptionVariant, err := oleutil.CallMethod(item, "GetDescription")
				if err == nil {
					names = append(names, descriptionVariant.ToString())
					descriptionVariant.Clear()
				}
			}
			langVariant.Clear()
		}
		itemVariant.Clear()
	}
	return names, nil
}

func (e *windowsEngine) lcidForLang(lang string) int {
	switch lang {
	case "th":
		return 0x041E
	case "en":
		return 0x0409
	default:
		return 0
	}
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	dl "github.com/takanoriyanagitani/go-detect-language-ql/detector/lingua"
)

func TestDetermineLanguageMode(t *testing.T) {
	t.Run("defaults to ModeAll", func(t *testing.T) {
		mode, err := determineLanguageMode("", false, false, false)
		assert.NoError(t, err)
		assert.IsType(t, dl.ModeAll{}, mode)
	})

	t.Run("uses ModeAll when -all is set", func(t *testing.T) {
		mode, err := determineLanguageMode("", false, false, true)
		assert.NoError(t, err)
		assert.IsType(t, dl.ModeAll{}, mode)
	})

	t.Run("uses ModeAllSpoken when -all-spoken is set", func(t *testing.T) {
		mode, err := determineLanguageMode("", false, true, false)
		assert.NoError(t, err)
		assert.IsType(t, dl.ModeAllSpoken{}, mode)
	})

	t.Run("uses ModeLangs when -languages is set", func(t *testing.T) {
		mode, err := determineLanguageMode("EN,JA", true, false, false)
		assert.NoError(t, err)
		assert.IsType(t, dl.ModeLangs{}, mode)
	})

	t.Run("errors when multiple flags are set", func(t *testing.T) {
		_, err := determineLanguageMode("EN,JA", true, true, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mutually exclusive")
	})

	t.Run("propagates error for single language", func(t *testing.T) {
		_, err := determineLanguageMode("EN", true, false, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 2 languages")
	})

	t.Run("propagates error for invalid language", func(t *testing.T) {
		_, err := determineLanguageMode("EN,invalid", true, false, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language")
	})
}

package text2lang_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pemistahl/lingua-go"
	"github.com/stretchr/testify/assert"
	dlang "github.com/takanoriyanagitani/go-detect-language-ql"
	"github.com/takanoriyanagitani/go-detect-language-ql/detector/lingua"
)

// mockConfidenceValue implements lingua.ConfidenceValue for testing
type mockConfidenceValue struct {
	lang  lingua.Language
	value float64
}

func (m mockConfidenceValue) Language() lingua.Language {
	return m.lang
}

func (m mockConfidenceValue) Value() float64 {
	return m.value
}

func ptr[T any](v T) *T { return &v }

func TestApplyFiltersAndGetStats(t *testing.T) {
	confidenceValues := []lingua.ConfidenceValue{
		mockConfidenceValue{lang: lingua.English, value: 0.9},
		mockConfidenceValue{lang: lingua.French, value: 0.7},
		mockConfidenceValue{lang: lingua.German, value: 0.5},
		mockConfidenceValue{lang: lingua.Spanish, value: 0.3},
	}
	supportedLangs := []lingua.Language{lingua.English, lingua.French, lingua.German, lingua.Spanish, lingua.Italian}

	t.Run("no filters", func(t *testing.T) {
		req := dlang.Request{ID: uuid.New(), Text: "test"}
		detected, stats := text2lang.ApplyFiltersAndGetStats(confidenceValues, req, supportedLangs)

		assert.Len(t, detected, 4)
		assert.Equal(t, int64(5), stats.TotalSupportedLanguages)
		assert.Equal(t, int64(4), stats.DetectedLanguages)
		assert.Equal(t, int64(0), stats.SkippedLanguages)
	})

	t.Run("with minConfidence", func(t *testing.T) {
		req := dlang.Request{ID: uuid.New(), Text: "test", MinConfidence: ptr(0.6)}
		detected, stats := text2lang.ApplyFiltersAndGetStats(confidenceValues, req, supportedLangs)

		assert.Len(t, detected, 2)
		assert.Equal(t, "English", detected[0].Language.Display)
		assert.Equal(t, "French", detected[1].Language.Display)
		assert.Equal(t, int64(5), stats.TotalSupportedLanguages)
		assert.Equal(t, int64(4), stats.DetectedLanguages)
		assert.Equal(t, int64(2), stats.SkippedLanguages)
	})

	t.Run("with maxLanguages", func(t *testing.T) {
		req := dlang.Request{ID: uuid.New(), Text: "test", MaxLanguages: ptr(int64(3))}
		detected, stats := text2lang.ApplyFiltersAndGetStats(confidenceValues, req, supportedLangs)

		assert.Len(t, detected, 3)
		assert.Equal(t, int64(5), stats.TotalSupportedLanguages)
		assert.Equal(t, int64(4), stats.DetectedLanguages)
		assert.Equal(t, int64(1), stats.SkippedLanguages)
	})

	t.Run("with both filters", func(t *testing.T) {
		req := dlang.Request{ID: uuid.New(), Text: "test", MinConfidence: ptr(0.4), MaxLanguages: ptr(int64(2))}
		detected, stats := text2lang.ApplyFiltersAndGetStats(confidenceValues, req, supportedLangs)

		assert.Len(t, detected, 2)
		assert.Equal(t, "English", detected[0].Language.Display)
		assert.Equal(t, "French", detected[1].Language.Display)
		assert.Equal(t, int64(5), stats.TotalSupportedLanguages)
		assert.Equal(t, int64(4), stats.DetectedLanguages)
		assert.Equal(t, int64(2), stats.SkippedLanguages)
	})
}

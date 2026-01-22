package text2lang

import (
	"context"

	"github.com/google/uuid"
)

type Language struct {
	IsoCode639_1 string `json:"iso_code_639_1"`
	IsoCode639_3 string `json:"iso_code_639_3"`
	Display      string `json:"display"`
}

type Confidence = float64

type DetectedLanguageInfo struct {
	Language   `json:"language"`
	Confidence `json:"confidence"`
}

type Request struct {
	ID            uuid.UUID `json:"id"`
	Text          string    `json:"text"`
	MinConfidence *float64  `json:"minConfidence"`
	MaxLanguages  *int64    `json:"maxLanguages"`
}

type Stats struct {
	TotalSupportedLanguages int64
	DetectedLanguages       int64
	SkippedLanguages        int64
}

type Response struct {
	ID       uuid.UUID              `json:"id"`
	Detected []DetectedLanguageInfo `json:"detected"`
	Stats    Stats
}

type Detector func(context.Context, Request) (Response, error)

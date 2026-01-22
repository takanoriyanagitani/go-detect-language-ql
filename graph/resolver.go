package graph

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dlang "github.com/takanoriyanagitani/go-detect-language-ql"
	"github.com/takanoriyanagitani/go-detect-language-ql/graph/model"
)

var (
	ErrInvalidRequestId error = errors.New("invalid request id")
)

type Resolver struct {
	dlang.Detector
}

func (r *Resolver) DetectLanguage(
	ctx context.Context,
	input *model.DetectLanguageInput,
) (*model.DetectLanguageOutput, error) {
	reqId, idErr := uuid.Parse(input.RequestID)
	if nil != idErr {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRequestId, idErr)
	}

	req := dlang.Request{
		ID:            reqId,
		Text:          input.Text,
		MinConfidence: input.MinConfidence,
		MaxLanguages:  input.MaxLanguages,
	}
	res, detErr := r.Detector(ctx, req)

	detected := make([]*model.DetectedLanguage, 0, len(res.Detected))
	for _, info := range res.Detected {
		var lang dlang.Language = info.Language
		var conf float64 = info.Confidence

		var code1 string = lang.IsoCode639_1
		var code3 string = lang.IsoCode639_3
		var disp string = lang.Display

		detected = append(detected, &model.DetectedLanguage{
			Iso639i:    code1,
			Iso639iii:  code3,
			Language:   disp,
			Confidence: conf,
		})
	}

	stats := &model.Stats{
		TotalSupportedLanguages: res.Stats.TotalSupportedLanguages,
		DetectedLanguages:       res.Stats.DetectedLanguages,
		SkippedLanguages:        res.Stats.SkippedLanguages,
	}

	return &model.DetectLanguageOutput{
		RequestID: input.RequestID,
		Detected:  detected,
		Stats:     stats,
	}, detErr
}

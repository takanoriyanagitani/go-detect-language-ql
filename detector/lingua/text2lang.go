package text2lang

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	lg "github.com/pemistahl/lingua-go"
	dlang "github.com/takanoriyanagitani/go-detect-language-ql"
)

var (
	ErrUnsupportedLanguage         error = errors.New("unsupported language")
	ErrLinguaInsufficientLanguages error = errors.New("lingua-go requires at least 2 languages for custom list")
)

type Config struct {
	SupportedLanguageMode LanguageMode

	MinimumRelativeDistance sql.Null[float64]
	Preload                 bool
	LowAccuracyMode         bool
}

type LanguageMode interface {
	Builder() lg.LanguageDetectorBuilder
	Languages() []lg.Language
}

type ModeAll struct{}

func (a ModeAll) Builder() lg.LanguageDetectorBuilder { return AllLanguages() }
func (a ModeAll) Languages() []lg.Language            { return lg.AllLanguages() }

type ModeAllSpoken struct{}

func (s ModeAllSpoken) Builder() lg.LanguageDetectorBuilder { return AllSpokenLanguages() }
func (s ModeAllSpoken) Languages() []lg.Language            { return lg.AllSpokenLanguages() }

type ModeLangs struct{ langs []lg.Language }

func (l ModeLangs) Builder() lg.LanguageDetectorBuilder { return FromLanguages(l.langs...) }
func (l ModeLangs) Languages() []lg.Language            { return l.langs }

var ConfigDefault Config = Config{
	SupportedLanguageMode: ModeAll{},
	MinimumRelativeDistance: sql.Null[float64]{
		V:     0.0,
		Valid: false,
	},
	Preload:         false,
	LowAccuracyMode: false,
}

func (c Config) ToLanguageDetector() lg.LanguageDetector {
	var bldr lg.LanguageDetectorBuilder = c.
		SupportedLanguageMode.
		Builder()

	if c.Preload {
		bldr = bldr.WithPreloadedLanguageModels()
	}

	if c.LowAccuracyMode {
		bldr = bldr.WithLowAccuracyMode()
	}

	if c.MinimumRelativeDistance.Valid {
		var v float64 = c.MinimumRelativeDistance.V
		bldr = bldr.WithMinimumRelativeDistance(v)
	}

	return bldr.Build()
}

func (c Config) ToDetector() dlang.Detector {
	detector := c.ToLanguageDetector()
	supportedLangs := c.SupportedLanguageMode.Languages()

	return func(ctx context.Context, req dlang.Request) (dlang.Response, error) {
		var text string = req.Text
		var linguaResults []lg.ConfidenceValue = detector.
			ComputeLanguageConfidenceValues(text)

		detectedFinal, stats := ApplyFiltersAndGetStats(linguaResults, req, supportedLangs)

		return dlang.Response{
			ID:       req.ID,
			Detected: detectedFinal,
			Stats:    stats,
		}, nil
	}
}

func AllLanguages() lg.LanguageDetectorBuilder {
	return lg.
		NewLanguageDetectorBuilder().
		FromAllLanguages()
}

func AllSpokenLanguages() lg.LanguageDetectorBuilder {
	return lg.
		NewLanguageDetectorBuilder().
		FromAllSpokenLanguages()
}

func FromLanguages(langs ...lg.Language) lg.LanguageDetectorBuilder {
	return lg.
		NewLanguageDetectorBuilder().
		FromLanguages(langs...)
}

func FromCodes1(codes ...lg.IsoCode639_1) lg.LanguageDetectorBuilder {
	return lg.
		NewLanguageDetectorBuilder().
		FromIsoCodes639_1(codes...)
}

func FromCodes3(codes ...lg.IsoCode639_3) lg.LanguageDetectorBuilder {
	return lg.
		NewLanguageDetectorBuilder().
		FromIsoCodes639_3(codes...)
}

func CreateLanguageMapper() func(string) (lg.Language, error) {
	return createSingleMapper(func(l lg.Language) string { return l.String() })
}

type languageKeyExtractor func(lg.Language) string

func createSingleMapper(keyExtractor languageKeyExtractor) func(string) (lg.Language, error) {
	mapper := map[string]lg.Language{}
	for _, lang := range lg.AllLanguages() {
		mapper[keyExtractor(lang)] = lang
	}
	return func(input string) (lg.Language, error) {
		lng, found := mapper[input]
		switch found {
		case true:
			return lng, nil
		default:
			return 0, fmt.Errorf("%w: %s", ErrUnsupportedLanguage, input)
		}
	}
}

func CreateCode1Mapper() func(string) (lg.Language, error) {
	return createSingleMapper(func(l lg.Language) string { return l.IsoCode639_1().String() })
}

func CreateCode3Mapper() func(string) (lg.Language, error) {
	return createSingleMapper(func(l lg.Language) string { return l.IsoCode639_3().String() })
}

func CreateCombinedMapper() func(string) (lg.Language, error) {
	nameMapper := CreateLanguageMapper()
	code1Mapper := CreateCode1Mapper()
	code3Mapper := CreateCode3Mapper()

	return func(input string) (lg.Language, error) {
		upperInput := strings.ToUpper(input)
		lang, err := code1Mapper(upperInput)
		if err == nil {
			return lang, nil
		}
		lang, err = code3Mapper(upperInput)
		if err == nil {
			return lang, nil
		}
		lang, err = nameMapper(upperInput)
		if err == nil {
			return lang, nil
		}
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedLanguage, input)
	}
}

func StringsToModeLangs(langs []string) (ModeLangs, error) {
	if len(langs) < 2 {
		return ModeLangs{}, ErrLinguaInsufficientLanguages
	}

	mapper := CreateCombinedMapper()
	lgLangs := make([]lg.Language, 0, len(langs))

	for _, langStr := range langs {
		lang, err := mapper(langStr)
		if err != nil {
			return ModeLangs{}, err
		}
		lgLangs = append(lgLangs, lang)
	}

	return ModeLangs{langs: lgLangs}, nil
}

func ApplyFiltersAndGetStats(
	results []lg.ConfidenceValue,
	req dlang.Request,
	supportedLangs []lg.Language,
) ([]dlang.DetectedLanguageInfo, dlang.Stats) {
	var minConfidence float64 = 0.0
	if req.MinConfidence != nil {
		minConfidence = *req.MinConfidence
	}

	var maxLanguages int64 = 0
	if req.MaxLanguages != nil {
		maxLanguages = *req.MaxLanguages
	}

	var detectedCountBeforeFilter int64 = int64(len(results))
	var skippedDueToConfidence int64 = 0

	var processedResults []dlang.DetectedLanguageInfo

	for _, result := range results {
		if result.Value() >= minConfidence {
			var lang lg.Language = result.Language()
			processedResults = append(processedResults, dlang.DetectedLanguageInfo{
				Language: dlang.Language{
					IsoCode639_1: lang.IsoCode639_1().String(),
					IsoCode639_3: lang.IsoCode639_3().String(),
					Display:      lang.String(),
				},
				Confidence: result.Value(),
			})
		} else {
			skippedDueToConfidence++
		}
	}

	var detectedFinal []dlang.DetectedLanguageInfo
	var skippedDueToMaxLanguages int64 = 0

	if maxLanguages > 0 && int64(len(processedResults)) > maxLanguages {
		detectedFinal = processedResults[:maxLanguages]
		skippedDueToMaxLanguages = int64(len(processedResults)) - maxLanguages
	} else {
		detectedFinal = processedResults
	}

	var totalSupported int64 = int64(len(supportedLangs))

	stats := dlang.Stats{
		TotalSupportedLanguages: totalSupported,
		DetectedLanguages:       detectedCountBeforeFilter,
		SkippedLanguages:        skippedDueToConfidence + skippedDueToMaxLanguages,
	}

	return detectedFinal, stats
}

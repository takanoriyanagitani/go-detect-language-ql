package main

import (
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	ql "github.com/99designs/gqlgen/graphql"
	gh "github.com/99designs/gqlgen/graphql/handler"
	he "github.com/99designs/gqlgen/graphql/handler/extension"
	ht "github.com/99designs/gqlgen/graphql/handler/transport"
	dlang "github.com/takanoriyanagitani/go-detect-language-ql"
	dl "github.com/takanoriyanagitani/go-detect-language-ql/detector/lingua"
	gr "github.com/takanoriyanagitani/go-detect-language-ql/graph"
	"github.com/takanoriyanagitani/go-detect-language-ql/internal/flagvalue"
)

var (
	ErrMutuallyExclusiveFlags = errors.New("flags -languages, -all-spoken, and -all are mutually exclusive")
)

var (
	// Server config.
	serverPort   int
	bindAddr     string
	readTimeout  time.Duration
	writeTimeout time.Duration

	// Detector config.
	preload         bool
	lowAccuracyMode bool
	minRelDist      flagvalue.NullFloat64
	languages       string
	allSpoken       bool
	allLangs        bool

	// Logging.
	logFormat string
)

func determineLanguageMode(
	languages string,
	languagesSet, allSpokenSet, allLangsSet bool,
) (dl.LanguageMode, error) {
	flagCount := 0
	if languagesSet {
		flagCount++
	}
	if allSpokenSet {
		flagCount++
	}
	// It's a default, so we only count it if it's explicitly set for exclusivity check.
	if allLangsSet {
		flagCount++
	}

	if flagCount > 1 {
		return nil, ErrMutuallyExclusiveFlags
	}

	if languagesSet {
		langs := strings.Split(languages, ",")
		if len(langs) == 1 && langs[0] == "" {
			// case where -languages="" is passed
			return dl.ModeAll{}, nil
		}
		return dl.StringsToModeLangs(langs)
	}

	if allSpokenSet {
		return dl.ModeAllSpoken{}, nil
	}

	return dl.ModeAll{}, nil
}

// setupLogger initializes and sets the default slog logger based on the logFormat.
func setupLogger(logFormat string) *slog.Logger {
	var handler slog.Handler
	switch logFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, nil)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, nil)
	default:
		slog.Error("invalid log format", "format", logFormat)
		os.Exit(1)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// setupHTTPServer initializes and returns an *http.Server instance.
func setupHTTPServer(bindAddr string, serverPort int, readTimeout, writeTimeout time.Duration) *http.Server {
	hsv := &http.Server{
		Addr:           bindAddr + ":" + strconv.Itoa(serverPort),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	return hsv
}

// getAndLogLanguageMode determines the language mode based on global flags and logs its configuration.
func getAndLogLanguageMode() (dl.LanguageMode, error) {
	languagesSet, allSpokenSet, allLangsSet := false, false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "languages":
			languagesSet = true
		case "all-spoken":
			allSpokenSet = true
		case "all":
			allLangsSet = true
		}
	})

	langMode, err := determineLanguageMode(languages, languagesSet, allSpokenSet, allLangsSet)
	if err != nil {
		return nil, err
	}

	if mode, ok := langMode.(dl.ModeLangs); ok {
		slog.Info("language mode configured", "mode", "custom")
		var langNames []string
		for _, l := range mode.Languages() {
			langNames = append(langNames, l.String())
		}
		for _, name := range langNames {
			slog.Info("custom language", "language", name)
		}
	} else if _, ok := langMode.(dl.ModeAllSpoken); ok {
		slog.Info("language mode configured", "mode", "all-spoken")
	} else {
		slog.Info("language mode configured", "mode", "all")
	}

	return langMode, nil
}

// configureDetectorAndLog creates the detector configuration and logs its settings.
func configureDetectorAndLog(langMode dl.LanguageMode) dl.Config {
	cfg := dl.Config{
		Preload:                 preload,
		LowAccuracyMode:         lowAccuracyMode,
		MinimumRelativeDistance: minRelDist.Nullable,
		SupportedLanguageMode:   langMode,
	}

	slog.Info("config",
		"type", "detector",
		"preload", cfg.Preload,
		"low_accuracy_mode", cfg.LowAccuracyMode,
		"min_relative_distance", minRelDist.String(),
	)
	return cfg
}

//nolint:gochecknoinits
func init() {
	// Server flags.
	flag.IntVar(&serverPort, "port", 12281, "Port for the GraphQL server to listen on.")
	flag.StringVar(&bindAddr, "bind-addr", "127.0.0.1", "Address to bind the server to (e.g., 127.0.0.1, 0.0.0.0).")
	flag.DurationVar(&readTimeout, "read-timeout", 10*time.Second, "Max duration for reading request (body included).")
	flag.DurationVar(&writeTimeout, "write-timeout", 10*time.Second, "Max duration before timing out response writes.")

	// Detector flags.
	flag.BoolVar(&preload, "preload", false, "Preload all language models for faster detection.")
	flag.BoolVar(&lowAccuracyMode, "low-accuracy", false, "Enable low-accuracy mode for faster, less precise detection.")
	flag.Var(&minRelDist, "min-rel-dist", "Set the minimum relative distance between languages (0.0 to 0.99).")
	flag.StringVar(&languages, "languages", "", "Comma-separated list of languages to detect (e.g., 'en,jpn,ENGLISH').")
	flag.BoolVar(&allSpoken, "all-spoken", false, "Use all spoken languages for detection.")
	flag.BoolVar(&allLangs, "all", false, "Use all languages (default, mutually exclusive).")

	// Logging.
	flag.StringVar(&logFormat, "log-format", "json", "Log format (json or text).")
}

func main() {
	flag.Parse() // Parse flags first

	// Setup logger.
	setupLogger(logFormat)

	// Validate port.
	if serverPort < 0 || serverPort > 65535 {
		slog.Error("invalid port number", "port", serverPort, "range", "0-65535")
		os.Exit(1)
	}

	langMode, err := getAndLogLanguageMode()
	if err != nil {
		slog.Error("failed to process language mode", "error", err)
		os.Exit(1)
	}

	var cfg dl.Config = configureDetectorAndLog(langMode)

	slog.Info("config",
		"type", "network",
		"port", serverPort,
		"bind_addr", bindAddr,
	)
	slog.Info("config",
		"type", "timeouts",
		"read", readTimeout,
		"write", writeTimeout,
	)

	slog.Info("loading...")
	var det dlang.Detector = cfg.ToDetector()
	slog.Info("loaded.")
	var res *gr.Resolver = &gr.Resolver{Detector: det}

	var gc gr.Config = gr.Config{Resolvers: res}
	var sc ql.ExecutableSchema = gr.NewExecutableSchema(gc)
	var svr *gh.Server = gh.New(sc)

	svr.AddTransport(ht.GET{})
	svr.AddTransport(ht.POST{})
	svr.Use(he.Introspection{})

	http.Handle("/query", svr)

	hsv := setupHTTPServer(bindAddr, serverPort, readTimeout, writeTimeout)

	slog.Info("ready.")

	err = hsv.ListenAndServe()
	if err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

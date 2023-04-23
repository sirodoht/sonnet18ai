package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/geoah/go-llm"
	"github.com/sashabaranov/go-openai"
	"github.com/sirodoht/sonnet18ai/internal"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"moul.io/chizap"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	databaseDSN := os.Getenv("DATABASE_DSN")
	if databaseDSN == "" {
		databaseDSN = "sonnet18.sqlite"
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // nolint: errcheck

	db, err := gorm.Open(
		sqlite.Open(databaseDSN),
		&gorm.Config{},
	)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	// enable debug mode
	if debugMode {
		logger.Info("enable debug mode")
		db = db.Debug()
	}

	// Construct a new store
	store := internal.NewSQLStore(db)

	// Construct a new llm client
	client := openai.NewClient(os.Getenv("OPENAI_TOKEN"))
	evaluators := map[string]llm.Evaluator{
		"gpt3p5": llm.NewChatGPT3p5(client),
		"gpt4":   llm.NewChatGPT4(client),
		// "llama7b": llm.NewLlama("./models/7B/ggml-model-f32.bin"),
	}

	llmClient := llm.NewService(
		os.Getenv("PREFIX"),
		evaluators,
	)

	// Construct a new router
	handlers := internal.NewHandlers(logger, store, llmClient)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(chizap.New(logger, &chizap.Opts{
		WithReferer:   true,
		WithUserAgent: true,
	}))

	// register handlers
	handlers.Register(r)

	// serve
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	fmt.Println("Listening on", port, "...")
	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

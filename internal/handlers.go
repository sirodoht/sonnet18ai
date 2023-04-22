package internal

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

type Handlers struct {
	logger *zap.Logger
	store  *SQLStore
}

type TemplateValues struct {
	Document *Document
	Error    string
}

func NewHandlers(logger *zap.Logger, store *SQLStore) *Handlers {
	return &Handlers{
		logger: logger,
		store:  store,
	}
}

func (handlers *Handlers) Register(r *chi.Mux) {
	// register handlers
	r.Get("/", handlers.HandleIndex)

	// static files
	fs := http.FileServer(http.FS(staticFiles))
	r.Handle("/static/*", fs)
}

func (handlers *Handlers) HandleIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFS(templateFiles,
		"internal/templates/layout.html",
		"internal/templates/index.html",
	)
	if err != nil {
		panic(err)
	}

	err = t.Execute(w, nil)
	if err != nil {
		panic(err)
	}
	handlers.logger.Info("render index success")
}

package internal

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strconv"

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

func NewHandlers(logger *zap.Logger, store *SQLStore) *Handlers {
	return &Handlers{
		logger: logger,
		store:  store,
	}
}

func (handlers *Handlers) Register(r *chi.Mux) {
	// register handlers
	r.Get("/", handlers.HandleIndex)
	r.Get("/text/{id}", handlers.HandleDocument)
	r.Post("/text/{id}/update", handlers.HandleDocumentUpdate)

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
		"templates/layout.html",
		"templates/index.html",
	)
	if err != nil {
		panic(err)
	}

	documents, err := handlers.store.GetDocuments(r.Context())
	if err != nil {
		handlers.logger.Error("could not get documents")
		panic(err)
	}

	values := struct {
		Documents []*Document
	}{
		Documents: documents,
	}
	err = t.Execute(w, values)
	if err != nil {
		panic(err)
	}
	handlers.logger.Info("render index success")
}

func (handlers *Handlers) HandleDocument(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFS(templateFiles,
		"templates/layout.html",
		"templates/document.html",
	)
	if err != nil {
		panic(err)
	}

	idString := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		handlers.logger.Error("could not parse id")
		panic(err)
	}
	uID := uint(id)

	document, err := handlers.store.GetDocument(r.Context(), uID)
	if err != nil {
		handlers.logger.Error("could not get document")
		panic(err)
	}
	values := struct {
		Document *Document
	}{
		Document: document,
	}
	err = t.Execute(w, values)
	if err != nil {
		panic(err)
	}
}

func (handlers *Handlers) HandleDocumentUpdate(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	uID := uint(id)
	if err != nil {
		handlers.logger.With(
			zap.Error(err),
		).Error("cannot find document")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type RequestBody struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	var rb RequestBody
	err = json.Unmarshal(b, &rb)
	if err != nil {
		handlers.logger.With(
			zap.Error(err),
		).Error("failed to parse input data")
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// update document on database
	err = handlers.store.UpdateDocument(
		r.Context(),
		uID,
		*rb.Title,
		*rb.Body,
	)
	if err != nil {
		handlers.logger.Error("could not update document")
		panic(err)
	}

	// respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

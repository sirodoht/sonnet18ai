package internal

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	llm "github.com/geoah/go-llm"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

type Handlers struct {
	logger    *zap.Logger
	store     *SQLStore
	llmClient *llm.Service
}

func NewHandlers(
	logger *zap.Logger,
	store *SQLStore,
	llmClient *llm.Service,
) *Handlers {
	return &Handlers{
		logger:    logger,
		store:     store,
		llmClient: llmClient,
	}
}

func (handlers *Handlers) Register(r *chi.Mux) {
	// register handlers
	r.Get("/", handlers.HandleIndex)
	r.Get("/text/new", handlers.HandleDocumentNew)
	r.Post("/text/delete", handlers.HandleDocumentDelete)
	r.Get("/text/{id}", handlers.HandleDocument)
	r.Post("/text/{id}/update", handlers.HandleDocumentUpdate)
	r.Post("/api/v1/evaluate", handlers.HandleEvaluate)

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
	DocumentBody := strings.ReplaceAll(document.Body, "\n", "<br>")
	values := struct {
		Document     *Document
		DocumentBody template.HTML
		Token        string
	}{
		Document:     document,
		DocumentBody: template.HTML(DocumentBody),
		Token:        os.Getenv("LLMAPI_TOKEN"),
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

func (handlers *Handlers) HandleDocumentNew(
	w http.ResponseWriter,
	r *http.Request,
) {
	// create document on database
	id, err := handlers.store.CreateDocument(r.Context())
	if err != nil {
		handlers.logger.Error("could not update document")
		panic(err)
	}

	// redirect to document
	http.Redirect(w, r, fmt.Sprintf("/text/%d", id), http.StatusSeeOther)
}

func (handlers *Handlers) HandleDocumentDelete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	uID := uint(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// delete document on database
	err = handlers.store.DeleteDocument(r.Context(), uID)
	if err != nil {
		handlers.logger.Error("could not update document")
		panic(err)
	}

	// redirect to document
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type (
	EvaluateRequest struct {
		Prompt string `json:"prompt"`
		Model  string `json:"model"`
	}
	EvaluateResponse struct {
		Result string `json:"result"`
	}
)

func (handlers *Handlers) HandleEvaluate(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token != os.Getenv("LLMAPI_TOKEN") {
		fmt.Println("got", token, "expected", os.Getenv("LLMAPI_TOKEN"))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := handlers.llmClient.Evaluate(r.Context(), req.Model, req.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := EvaluateResponse{Result: result}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

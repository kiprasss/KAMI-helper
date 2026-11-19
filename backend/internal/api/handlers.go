package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"backend/internal/forum"
	"backend/internal/kb"

	"github.com/go-chi/chi/v5"
)

const maxQueryLength = 200

type Handler struct {
	store   *kb.Store
	fetcher *forum.Fetcher
	images  *imageCache
}

func NewHandler(store *kb.Store, fetcher *forum.Fetcher) *Handler {
	return &Handler{store: store, fetcher: fetcher, images: newImageCache()}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/servers", h.servers)
	r.Get("/status", h.status)
	r.Get("/search", h.search)
	r.Get("/rules/{id}", h.rule)
	r.Get("/image", h.image)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Ошибка отправки ответа: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) servers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": h.store.Servers()})
}

func (h *Handler) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": h.store.Status()})
}

type searchResult struct {
	ID            string  `json:"id"`
	Kind          string  `json:"kind"`
	Point         string  `json:"point"`
	DocumentTitle string  `json:"document_title"`
	Breadcrumbs   string  `json:"breadcrumbs"`
	DocumentType  string  `json:"document_type"`
	Snippet       string  `json:"snippet"`
	Punishment    string  `json:"punishment"`
	URL           string  `json:"url"`
	Score         float64 `json:"score"`
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	server := strings.TrimSpace(r.URL.Query().Get("server"))

	if query == "" {
		writeError(w, http.StatusBadRequest, "Введите поисковый запрос")
		return
	}

	if utf8.RuneCountInString(query) > maxQueryLength {
		writeError(w, http.StatusBadRequest, "Слишком длинный запрос")
		return
	}

	if !h.store.HasServer(server) {
		writeError(w, http.StatusBadRequest, "Неизвестный сервер")
		return
	}

	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "Лимит должен быть от 1 до 100")
			return
		}
		limit = parsed
	}

	hits := h.store.Search(query, server)
	total := len(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}

	terms := kb.QueryTerms(query)
	results := make([]searchResult, 0, len(hits))

	for _, hit := range hits {
		e := hit.Entry
		results = append(results, searchResult{
			ID:            e.ID,
			Kind:          e.Kind,
			Point:         e.Point,
			DocumentTitle: e.Title,
			Breadcrumbs:   e.Breadcrumbs,
			DocumentType:  e.DocType,
			Snippet:       kb.Snippet(e.Text, terms),
			Punishment:    e.Punishment,
			URL:           e.PostURL,
			Score:         hit.Score,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": results,
		"meta": map[string]int{
			"total": total,
			"limit": limit,
		},
	})
}

func (h *Handler) rule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	entry := h.store.Get(id)
	if entry == nil {
		writeError(w, http.StatusNotFound, "Правило не найдено. Возможно, база обновилась — повторите поиск.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": entry})
}

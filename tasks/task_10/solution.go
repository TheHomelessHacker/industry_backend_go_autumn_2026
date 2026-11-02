package main

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"encoding/json"
	"slices"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

// Инициализация
func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if len(title) == 0 {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	id := strconv.FormatUint(r.seq, 10)
	task := Task{
		ID:        id,
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[id] = task
	return task, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.tasks[id]
	if ok {
		return task, true
	}
	return Task{}, false
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	list := make([]Task, 0)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, task := range r.tasks {
		if task.Done == done {
			list = append(list, task)
		}
	}
	slices.SortFunc(list, func(a, b Task) int {
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return list
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[id]
	if ok {
		if task.Done != done {
			task.UpdatedAt = r.clock.Now()
			task.Done = done
			r.tasks[id] = task
		}
		return task, nil
	}
	return Task{}, ErrNotFound
}

type httpHandler struct{ 
	repo TaskRepo
	mux *http.ServeMux
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	h := &httpHandler{repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", h.handleCreate)
	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) { h.handleGet(w, r, r.PathValue("id")) })
	mux.HandleFunc("GET /tasks", h.handleList)
	mux.HandleFunc("PATCH /tasks/{id}", func(w http.ResponseWriter, r *http.Request) { h.handlePatch(w, r, r.PathValue("id")) })
	h.mux = mux
	return h
}

func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing data")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w,r)
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(req.Title)
	if errors.Is(err, ErrInvalidTitle) {
		http.Error(w, "invalid title", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	doneValues := r.URL.Query()["done"]
	done := false
	
	if len(doneValues) > 1 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(doneValues) == 1 {
		if doneValues[0] == "true" {
			done = true
		} else if doneValues[0] != "false" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	writeJSON(w, http.StatusOK, tasks)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Done == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
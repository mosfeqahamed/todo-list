package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"todo-list/internal/middleware"
	"todo-list/internal/models"
)

const maxTitleLen = 200

type TodoHandler struct {
	DB *pgxpool.Pool
}

type todoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func (req *todoRequest) validate() string {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return "title is required"
	}
	if len(req.Title) > maxTitleLen {
		return "title must be at most 200 characters"
	}
	return ""
}

const todoColumns = `id, user_id, title, description, completed, created_at, updated_at`

func scanTodo(row pgx.Row) (models.Todo, error) {
	var t models.Todo
	err := row.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (h *TodoHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	rows, err := h.DB.Query(r.Context(),
		`SELECT `+todoColumns+` FROM todos WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		log.Printf("list todos: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	todos := []models.Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			log.Printf("scan todo: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("iterate todos: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, todos)
}

func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req todoRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	t, err := scanTodo(h.DB.QueryRow(r.Context(),
		`INSERT INTO todos (user_id, title, description, completed) VALUES ($1, $2, $3, $4)
		 RETURNING `+todoColumns,
		userID, req.Title, req.Description, req.Completed))
	if err != nil {
		log.Printf("create todo: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func (h *TodoHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	t, err := scanTodo(h.DB.QueryRow(r.Context(),
		`SELECT `+todoColumns+` FROM todos WHERE id = $1 AND user_id = $2`, id, userID))
	if h.handleRowErr(w, err, "get todo") {
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req todoRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	t, err := scanTodo(h.DB.QueryRow(r.Context(),
		`UPDATE todos SET title = $1, description = $2, completed = $3, updated_at = NOW()
		 WHERE id = $4 AND user_id = $5
		 RETURNING `+todoColumns,
		req.Title, req.Description, req.Completed, id, userID))
	if h.handleRowErr(w, err, "update todo") {
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	tag, err := h.DB.Exec(r.Context(),
		`DELETE FROM todos WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		log.Printf("delete todo: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid todo id")
		return 0, false
	}
	return id, true
}

// handleRowErr writes an error response and returns true if err is non-nil.
func (h *TodoHandler) handleRowErr(w http.ResponseWriter, err error, op string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "todo not found")
		return true
	}
	log.Printf("%s: %v", op, err)
	writeError(w, http.StatusInternalServerError, "internal error")
	return true
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"chai-shop/internal/i18n"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Meta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

type MenuItem struct {
	ID       int     `json:"id"`
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	Price    float64 `json:"price" validate:"required,gt=0"`
	ImageURL string  `json:"image_url,omitempty"`
}

type App struct {
	Conn      *pgxpool.Pool
	Validate  *validator.Validate
	Localizer *i18n.Localizer
}

func (app *App) HandleMenu(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if limit > 100 {
		app.writeError(w, r, http.StatusBadRequest, "limit_exceeded")
		return
	}

	offset := (page - 1) * limit

	// URL থেকে সব parameter নাও
	search := r.URL.Query().Get("search")
	minPriceStr := r.URL.Query().Get("min_price")
	maxPriceStr := r.URL.Query().Get("max_price")

	// WHERE clause dynamically build করো
	conditions := []string{}
	args := []any{}

	// search condition
	if search != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search) + "%"
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", len(args)+1))
		args = append(args, pattern)
	}

	// min_price condition
	if minPriceStr != "" {
		if val, err := strconv.ParseFloat(minPriceStr, 64); err == nil && val >= 0 {
			conditions = append(conditions, fmt.Sprintf("price >= $%d", len(args)+1))
			args = append(args, val)
		}
	}

	// max_price condition
	if maxPriceStr != "" {
		if val, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && val >= 0 {
			conditions = append(conditions, fmt.Sprintf("price <= $%d", len(args)+1))
			args = append(args, val)
		}
	}

	// WHERE clause বানাও
	// conditions খালি হলে where = "" → সব item দেখাবে
	// conditions আছে হলে where = "WHERE name ILIKE $1 AND price >= $2"
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// COUNT query — কতটা item আছে মোট
	var total int
	countArgs := append([]any{}, args...)
	err := app.Conn.QueryRow(
		context.Background(),
		fmt.Sprintf("SELECT COUNT(*) FROM menu_items %s", where),
		countArgs...,
	).Scan(&total)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}

	// LIMIT আর OFFSET এর জন্য args এ যোগ করো
	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, limit, offset)

	rows, err := app.Conn.Query(
		context.Background(),
		fmt.Sprintf(
			"SELECT id, name, price, COALESCE(image_url, '') FROM menu_items %s ORDER BY id LIMIT $%d OFFSET $%d",
			where, limitArg, offsetArg,
		),
		args...,
	)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}
	defer rows.Close()

	items := make([]MenuItem, 0)
	for rows.Next() {
		var item MenuItem
		err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.ImageURL)
		if err != nil {
			app.writeError(w, r, http.StatusInternalServerError, "db_read_error")
			return
		}
		items = append(items, item)
	}

	totalPages := (total + limit - 1) / limit

	app.writeJSONMeta(w, r, http.StatusOK, "success", items, &Meta{
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

func (app *App) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	var item MenuItem
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
		return
	}

	if !app.validateMenuItem(w, r, item) {
		return
	}

	err = app.Conn.QueryRow(
		context.Background(),
		"INSERT INTO menu_items (name, price) VALUES ($1, $2) RETURNING id",
		item.Name, item.Price,
	).Scan(&item.ID)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}

	app.writeJSON(w, r, http.StatusCreated, "item_created", item)
}

func (app *App) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var item MenuItem
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
		return
	}

	if !app.validateMenuItem(w, r, item) {
		return
	}

	tag, err := app.Conn.Exec(
		context.Background(),
		"UPDATE menu_items SET name=$1, price=$2 WHERE id=$3",
		item.Name, item.Price, id,
	)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}
	if tag.RowsAffected() == 0 {
		app.writeError(w, r, http.StatusNotFound, "item_not_found")
		return
	}

	app.writeJSON(w, r, http.StatusOK, "item_updated", item)
}

func (app *App) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_id")
		return
	}

	_, err = app.Conn.Exec(context.Background(), "DELETE FROM menu_items WHERE id=$1", id)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}

	app.writeJSON(w, r, http.StatusOK, "item_deleted", nil)
}

func (app *App) validateMenuItem(w http.ResponseWriter, r *http.Request, item MenuItem) bool {
	err := app.Validate.Struct(item)
	if err != nil {
		errors := err.(validator.ValidationErrors)
		for _, e := range errors {
			switch e.Field() {
			case "Name":
				app.writeError(w, r, http.StatusBadRequest, "name_invalid")
				return false
			case "Price":
				app.writeError(w, r, http.StatusBadRequest, "price_invalid")
				return false
			}
		}
	}
	return true
}

func (app *App) writeJSON(w http.ResponseWriter, r *http.Request, status int, key string, data any) {
	lang := i18n.DetectLang(r.Header.Get("Accept-Language"))
	message := app.Localizer.T(lang, key)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: status < 400,
		Message: message,
		Data:    data,
	})
}

func (app *App) writeJSONMeta(w http.ResponseWriter, r *http.Request, status int, key string, data any, meta *Meta) {
	lang := i18n.DetectLang(r.Header.Get("Accept-Language"))
	message := app.Localizer.T(lang, key)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: status < 400,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func (app *App) writeError(w http.ResponseWriter, r *http.Request, status int, key string) {
	app.writeJSON(w, r, status, key, nil)
}
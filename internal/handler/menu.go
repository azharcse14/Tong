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

type Translation struct {
    Lang string `json:"lang"`
    Name string `json:"name" validate:"required,min=2,max=100"`
}

type MenuItem struct {
    ID           int           `json:"id"`
    Price        float64       `json:"price" validate:"required,gt=0"`
    ImageURL     string        `json:"image_url,omitempty"`
    Translations []Translation `json:"translations,omitempty"`
    Name         string        `json:"name,omitempty"` // response এ current lang এর নাম
}

type MenuItemRequest struct {
    NameBn string  `json:"name_bn" validate:"required,min=2,max=100"`
    NameEn string  `json:"name_en" validate:"required,min=2,max=100"`
    Price  float64 `json:"price" validate:"required,gt=0"`
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

    // user কোন ভাষায় আছে সেটা detect করো
    lang := i18n.DetectLang(r.Header.Get("Accept-Language"))

    search := r.URL.Query().Get("search")
    minPriceStr := r.URL.Query().Get("min_price")
    maxPriceStr := r.URL.Query().Get("max_price")

    conditions := []string{}
    args := []any{}

    // search — সেই ভাষার translation এ খুঁজবে
    if search != "" {
        pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search) + "%"
        conditions = append(conditions, fmt.Sprintf(
            "EXISTS (SELECT 1 FROM menu_item_translations t WHERE t.item_id = m.id AND t.lang = $%d AND t.name ILIKE $%d)",
            len(args)+1, len(args)+2,
        ))
        args = append(args, lang, pattern)
    }

    if minPriceStr != "" {
        if val, err := strconv.ParseFloat(minPriceStr, 64); err == nil && val >= 0 {
            conditions = append(conditions, fmt.Sprintf("m.price >= $%d", len(args)+1))
            args = append(args, val)
        }
    }

    if maxPriceStr != "" {
        if val, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && val >= 0 {
            conditions = append(conditions, fmt.Sprintf("m.price <= $%d", len(args)+1))
            args = append(args, val)
        }
    }

    where := ""
    if len(conditions) > 0 {
        where = "WHERE " + strings.Join(conditions, " AND ")
    }

    var total int
    countArgs := append([]any{}, args...)
    err := app.Conn.QueryRow(
        context.Background(),
        fmt.Sprintf("SELECT COUNT(*) FROM menu_items m %s", where),
        countArgs...,
    ).Scan(&total)
    if err != nil {
        app.writeError(w, r, http.StatusInternalServerError, "db_error")
        return
    }

    limitArg := len(args) + 1
    offsetArg := len(args) + 2
    args = append(args, limit, offset)

    // menu_items এর সাথে translation JOIN করো
    // সেই ভাষার নাম না থাকলে যেকোনো একটা নাম দেখাবে (COALESCE)
    rows, err := app.Conn.Query(
        context.Background(),
        fmt.Sprintf(`
            SELECT m.id, m.price, COALESCE(m.image_url, ''),
                   COALESCE(
                       (SELECT name FROM menu_item_translations WHERE item_id = m.id AND lang = '%s'),
                       (SELECT name FROM menu_item_translations WHERE item_id = m.id LIMIT 1),
                       ''
                   ) as name
            FROM menu_items m %s
            ORDER BY m.id
            LIMIT $%d OFFSET $%d
        `, lang, where, limitArg, offsetArg),
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
        err := rows.Scan(&item.ID, &item.Price, &item.ImageURL, &item.Name)
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
    var req MenuItemRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
        return
    }
    if err := app.Validate.Struct(req); err != nil {
        app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
        return
    }

    // প্রথমে menu_items এ insert করো, id পাও
    var itemID int
    err = app.Conn.QueryRow(
        context.Background(),
        "INSERT INTO menu_items (price) VALUES ($1) RETURNING id",
        req.Price,
    ).Scan(&itemID)
    if err != nil {
        app.writeError(w, r, http.StatusInternalServerError, "db_error")
        return
    }

    // তারপর দুটো translation insert করো
    _, err = app.Conn.Exec(
        context.Background(),
        "INSERT INTO menu_item_translations (item_id, lang, name) VALUES ($1, 'bn', $2), ($1, 'en', $3)",
        itemID, req.NameBn, req.NameEn,
    )
    if err != nil {
        app.writeError(w, r, http.StatusInternalServerError, "db_error")
        return
    }

    app.writeJSON(w, r, http.StatusCreated, "item_created", map[string]any{
        "id":      itemID,
        "price":   req.Price,
        "name_bn": req.NameBn,
        "name_en": req.NameEn,
    })
}

func (app *App) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        app.writeError(w, r, http.StatusBadRequest, "invalid_id")
        return
    }
    var req MenuItemRequest
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
        return
    }
    if err := app.Validate.Struct(req); err != nil {
        app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
        return
    }

    tag, err := app.Conn.Exec(
        context.Background(),
        "UPDATE menu_items SET price=$1 WHERE id=$2",
        req.Price, id,
    )
    if err != nil {
        app.writeError(w, r, http.StatusInternalServerError, "db_error")
        return
    }
    if tag.RowsAffected() == 0 {
        app.writeError(w, r, http.StatusNotFound, "item_not_found")
        return
    }

    // translation upsert — আগে থাকলে update, না থাকলে insert
    _, err = app.Conn.Exec(
        context.Background(),
        `INSERT INTO menu_item_translations (item_id, lang, name) VALUES ($1, 'bn', $2), ($1, 'en', $3)
         ON CONFLICT (item_id, lang) DO UPDATE SET name = EXCLUDED.name`,
        id, req.NameBn, req.NameEn,
    )
    if err != nil {
        app.writeError(w, r, http.StatusInternalServerError, "db_error")
        return
    }

    app.writeJSON(w, r, http.StatusOK, "item_updated", map[string]any{
        "id":      id,
        "price":   req.Price,
        "name_bn": req.NameBn,
        "name_en": req.NameEn,
    })
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
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

func (app *App) Register(w http.ResponseWriter, r *http.Request) {
	// ধাপ ১: Request থেকে data নাও
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
		return
	}

	// ধাপ ২: Validation
	if err := app.Validate.Struct(req); err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
		return
	}

	// ধাপ ৩: Password hash করো
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}

	// ধাপ ৪: Database এ save করো
	var userID int
	err = app.Conn.QueryRow(
		context.Background(),
		"INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id",
		req.Name, req.Email, string(hashedPassword),
	).Scan(&userID)

	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "email_exists")
		return
	}

	go sendWelcomeEmail(req.Email, req.Name)

	app.writeJSON(w, r, http.StatusCreated, "register_success", map[string]any{
		"id":    userID,
		"name":  req.Name,
		"email": req.Email,
	})
}


type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (app *App) Login(w http.ResponseWriter, r *http.Request) {
	// ধাপ ১: Request থেকে data নাও
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_payload")
		return
	}

	// ধাপ ২: Database থেকে user খোঁজো
	var userID int
	var name, hashedPassword string
	err = app.Conn.QueryRow(
		context.Background(),
		"SELECT id, name, password FROM users WHERE email=$1",
		req.Email,
	).Scan(&userID, &name, &hashedPassword)

	if err != nil {
		app.writeError(w, r, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	// ধাপ ৩: Password মেলাও
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password))
	if err != nil {
		app.writeError(w, r, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	// ধাপ ৪: JWT token বানাও
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"name":    name,
		"email":   req.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}

	app.writeJSON(w, r, http.StatusOK, "login_success", map[string]string{
		"token": tokenString,
	})
}
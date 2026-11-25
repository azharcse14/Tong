package main

import (
	"fmt"
	"net/http"
	"os"

	"chai-shop/internal/db"
	"chai-shop/internal/handler"
	"chai-shop/internal/i18n"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(".env file পাওয়া যায়নি")
	}

	dbURL := os.Getenv("DB_URL")
	port := os.Getenv("PORT")

	conn, err := db.Connect(dbURL)
	if err != nil {
		fmt.Println("Database connect হয়নি:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Database connected!")

	localizer, err := i18n.NewLocalizer(map[string]string{
    "en": "internal/i18n/en.json",
    "bn": "internal/i18n/bn.json",
	})

	if err != nil {
		fmt.Println("i18n load হয়নি:", err)
		return
	}

	app := &handler.App{
		Conn:      conn,
		Validate:  validator.New(),
		Localizer: localizer,
	}

	mux := http.NewServeMux()
	// Public
	mux.HandleFunc("GET /menu", app.HandleMenu)
	mux.Handle("POST /menu", app.AuthMiddleware(http.HandlerFunc(app.CreateMenuItem)))
	mux.Handle("PUT /menu/{id}", app.AuthMiddleware(http.HandlerFunc(app.UpdateMenuItem)))
	mux.Handle("DELETE /menu/{id}", app.AuthMiddleware(http.HandlerFunc(app.DeleteMenuItem)))
	mux.Handle("POST /menu/{id}/image", app.AuthMiddleware(http.HandlerFunc(app.UploadMenuImage)))

	// Auth
	mux.HandleFunc("POST /auth/register", app.Register)
	mux.HandleFunc("POST /auth/login", app.Login)

	// Docs
	mux.HandleFunc("GET /docs", handler.SwaggerUI)
	mux.HandleFunc("GET /swagger.yaml", handler.SwaggerYAML)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	wrappedMux := handler.Logger(handler.Recoverer(app.RateLimit(mux)))
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: wrappedMux,
	}

	fmt.Println("দোকান খুলেছে! Port", port, "এ চলছে...")
	server.ListenAndServe()
}
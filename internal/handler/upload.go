package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func (app *App) UploadMenuImage(w http.ResponseWriter, r *http.Request) {
	// ID নাও
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "invalid_id")
		return
	}

	// Max 5MB
	r.ParseMultipartForm(5 << 20)

	// Form থেকে file নাও
	file, header, err := r.FormFile("image")
	if err != nil {
		app.writeError(w, r, http.StatusBadRequest, "image_required")
		return
	}
	defer file.Close()

	// Extension check করো
	ext := filepath.Ext(header.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		app.writeError(w, r, http.StatusBadRequest, "invalid_image_type")
		return
	}

	// আসল content check (শুধু extension না)
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if ct := http.DetectContentType(buf[:n]); ct != "image/jpeg" && ct != "image/png" {
		app.writeError(w, r, http.StatusBadRequest, "invalid_image_type")
		return
	}
	file.Seek(0, io.SeekStart)

	// Unique filename বানাও
	filename := fmt.Sprintf("%d_%d%s", id, time.Now().UnixNano(), ext)
	savePath := filepath.Join("uploads", filename)

	// File save করো
	dst, err := os.Create(savePath)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}
	defer dst.Close()

	io.Copy(dst, file)

	// Database এ URL save করো
	imageURL := "/uploads/" + filename
	tag, err := app.Conn.Exec(
		context.Background(),
		"UPDATE menu_items SET image_url=$1 WHERE id=$2",
		imageURL, id,
	)
	if err != nil {
		app.writeError(w, r, http.StatusInternalServerError, "db_error")
		return
	}
	if tag.RowsAffected() == 0 {
		os.Remove(savePath)
		app.writeError(w, r, http.StatusNotFound, "item_not_found")
		return
	}

	app.writeJSON(w, r, http.StatusOK, "image_uploaded", map[string]string{
		"image_url": imageURL,
	})
}
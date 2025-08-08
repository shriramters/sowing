package controller

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/niklasfasching/go-org/org"
)

// Misc provides miscellaneous handlers
type Misc struct {
	DB *sql.DB
}

// Register registers the misc routes
func (m *Misc) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /_preview", m.preview)
	mux.HandleFunc("POST /upload", m.upload)
}

func (m *Misc) preview(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	htmlContentString, err := org.New().Parse(strings.NewReader(string(body)), "").Write(org.NewHTMLWriter())
	if err != nil {
		log.Printf("Error converting org-mode content to HTML: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlContentString))
}

func (m *Misc) upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "The uploaded file is too big.", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, _ := io.ReadAll(file)
	hash := sha256.Sum256(fileBytes)
	uniqueFilename := fmt.Sprintf("%s-%d%s",
		hex.EncodeToString(hash[:16]),
		time.Now().Unix(),
		filepath.Ext(handler.Filename))

	dst, err := os.Create(filepath.Join("uploads", uniqueFilename))
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := dst.Write(fileBytes); err != nil {
			http.Error(w, "Error writing the file", http.StatusInternalServerError)
			return
		}

	_, err = m.DB.Exec(
		"INSERT INTO attachments (filename, unique_filename, mime_type, size) VALUES (?, ?, ?, ?)",
		handler.Filename, uniqueFilename, handler.Header.Get("Content-Type"), handler.Size)
	if err != nil {
		log.Printf("Error saving attachment to database: %v", err)
		http.Error(w, "Error saving file metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"url": "/uploads/%s"}`, uniqueFilename)
}

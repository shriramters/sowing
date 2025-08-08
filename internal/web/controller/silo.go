package controller

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"sowing/internal/models"
	"sowing/internal/web/viewmodels"
)

// Silo provides silo handlers
type Silo struct {
	DB        *sql.DB
	Templates map[string]*template.Template
}

// Register registers the silo routes
func (s *Silo) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.list)
	mux.HandleFunc("POST /", s.create)
}

func (s *Silo) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query("SELECT id, slug, name, archived_at, cover_image FROM silos WHERE archived_at IS NULL")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer rows.Close()

	var silos []models.Silo
	for rows.Next() {
		var silo models.Silo
		if err := rows.Scan(&silo.ID, &silo.Slug, &silo.Name, &silo.ArchivedAt, &silo.CoverImage); err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		silos = append(silos, silo)
	}

	user := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silos:       silos,
		ShowSidebar: false,
		CurrentUser: user,
		IsLoggedIn:  user != nil,
	}

	err = s.Templates["index.html"].ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println(err)
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, "Internal Server Error", 500)
		}
	}
}

func (s *Silo) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	name := r.PostFormValue("name")
	slug := r.PostFormValue("slug")

	if name == "" || slug == "" {
		http.Error(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	var coverImageURL *string

	file, handler, err := r.FormFile("cover_image")
	if err != nil && err != http.ErrMissingFile {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}

	if err != http.ErrMissingFile {
		defer file.Close()
		fileBytes, _ := io.ReadAll(file)
		hash := sha256.Sum256(fileBytes)
		uniqueFilename := fmt.Sprintf("%s-%d%s", hex.EncodeToString(hash[:16]), time.Now().Unix(), filepath.Ext(handler.Filename))

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

		url := "/uploads/" + uniqueFilename
		coverImageURL = &url
	}

	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "INSERT INTO silos (name, slug, cover_image) VALUES (?, ?, ?)", name, slug, coverImageURL)
	if err != nil {
		log.Printf("Error creating silo: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	siloID, _ := res.LastInsertId()

	res, err = tx.ExecContext(ctx, "INSERT INTO pages (silo_id, slug, title, current_revision_id) VALUES (?, 'home', 'Home', -1)", siloID)
	if err != nil {
		log.Printf("Error creating home page: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	pageID, _ := res.LastInsertId()

	initialContent := fmt.Sprintf("* Welcome to the %s Silo!", name)
	res, err = tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, 1, 'Initial creation', ?)", pageID, initialContent)
	if err != nil {
		log.Printf("Error creating initial revision: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	revisionID, _ := res.LastInsertId()

	_, err = tx.ExecContext(ctx, "UPDATE pages SET current_revision_id = ? WHERE id = ?", revisionID, pageID)
	if err != nil {
		log.Printf("Error updating page with revision ID: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

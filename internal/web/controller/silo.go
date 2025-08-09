package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sowing/internal/models"
	"sowing/internal/silo"
	"sowing/internal/web/viewmodels"
	"time"
)

// Silo provides silo handlers
type Silo struct {
	SiloRepo  *silo.Repository
	Templates map[string]*template.Template
}

// Register registers the silo routes
func (s *Silo) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.list)
	mux.HandleFunc("POST /", s.create)
}

func (s *Silo) list(w http.ResponseWriter, r *http.Request) {
	silos, err := s.SiloRepo.List()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silos:       silos,
		ShowSidebar: false,
		CurrentUser: user,
		IsLoggedIn:  user != nil,
	}

	err = s.Templates["index.html"].ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println(err)
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

	err = s.SiloRepo.Create(name, slug, coverImageURL)
	if err != nil {
		log.Printf("Error creating silo: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

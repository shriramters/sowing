package web

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/niklasfasching/go-org/org"
	"sowing/internal/models"
)

// PageData is a unified struct to hold all possible data for any page.
// This ensures that the templates always receive a consistent data structure.
type PageData struct {
	ShowSidebar bool
	Silos       []models.Silo
	Silo        models.Silo
	Page        models.Page
	Content     template.HTML
}

// Homepage now acts as a router. It serves the silo list for the root path
// and delegates to the wiki page handler for wiki paths.
func Homepage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	wikiPageHandler := viewWikiPage(db, templates)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		// Root path shows the list of silos.
		if path == "/" {
			rows, err := db.Query("SELECT id, slug, name, archived_at, cover_image FROM silos WHERE archived_at IS NULL")
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

			data := PageData{
				Silos:       silos,
				ShowSidebar: false,
			}

			// Use the "index.html" template set to execute the layout.
			err = templates["index.html"].ExecuteTemplate(w, "layout.html", data)
			if err != nil {
				log.Println(err)
				if w.Header().Get("Content-Type") == "" {
					http.Error(w, "Internal Server Error", 500)
				}
			}
			return
		}

		// Paths like /{silo-slug}/wiki/... are handled by the wiki page handler.
		if len(parts) >= 2 && parts[1] == "wiki" {
			wikiPageHandler(w, r)
			return
		}

		http.NotFound(w, r)
	}
}

// viewWikiPage handles rendering a single wiki page.
func viewWikiPage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		siloSlug := parts[0]
		pagePath := parts[2:]

		if len(pagePath) != 1 {
			http.Error(w, "Hierarchical pages not yet implemented", http.StatusNotImplemented)
			return
		}
		pageSlug := pagePath[0]

		var silo models.Silo
		err := db.QueryRow("SELECT id, slug, name FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID, &silo.Slug, &silo.Name)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		var page models.Page
		err = db.QueryRow("SELECT id, title, current_revision_id FROM pages WHERE silo_id = ? AND slug = ? AND parent_id IS NULL", silo.ID, pageSlug).Scan(&page.ID, &page.Title, &page.CurrentRevisionID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		page.Slug = pageSlug

		var revision models.Revision
		err = db.QueryRow("SELECT content FROM revisions WHERE id = ?", page.CurrentRevisionID).Scan(&revision.Content)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		htmlContentBytes, err := org.New().Parse(strings.NewReader(revision.Content), "").Write(org.NewHTMLWriter())
		if err != nil {
			log.Printf("Error converting org-mode content to HTML: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		data := PageData{
			Silo:        silo,
			Page:        page,
			Content:     template.HTML(htmlContentBytes),
			ShowSidebar: true,
		}

		// Use the "view.html" template set to execute the layout.
		err = templates["view.html"].ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			log.Println(err)
			if w.Header().Get("Content-Type") == "" {
				http.Error(w, "Internal Server Error", 500)
			}
		}
	}

}

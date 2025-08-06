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
// SiloPages is now a tree structure instead of a flat list.
type PageData struct {
	ShowSidebar bool
	Silos       []models.Silo
	Silo        models.Silo
	Page        models.Page    // The current page being viewed
	SiloPages   []*models.Page // The page tree for the sidebar
	Content     template.HTML
}

// buildPageTree takes a flat list of pages (already sorted by position)
// and organizes them into a hierarchical tree.
func buildPageTree(pages []models.Page) []*models.Page {
	pageMap := make(map[int]*models.Page)
	for i := range pages {
		p := pages[i]
		pageMap[p.ID] = &p
	}

	var rootPages []*models.Page
	// Iterate over the original sorted slice to maintain order.
	for _, p := range pages {
		pageNode := pageMap[p.ID]
		if pageNode.ParentID == nil {
			rootPages = append(rootPages, pageNode)
		} else {
			parent, ok := pageMap[*pageNode.ParentID]
			if ok {
				parent.Children = append(parent.Children, pageNode)
			}
		}
	}

	// After building the tree, construct the path for each node recursively.
	var constructPath func(pages []*models.Page, basePath string)
	constructPath = func(pages []*models.Page, basePath string) {
		for _, page := range pages {
			if basePath == "" {
				page.Path = page.Slug
			} else {
				page.Path = basePath + "/" + page.Slug
			}
			if len(page.Children) > 0 {
				constructPath(page.Children, page.Path)
			}
		}
	}

	constructPath(rootPages, "") // Start with an empty base path for root pages

	return rootPages
}

// Homepage acts as a router. It serves the silo list for the root path
// and delegates to the wiki page handler for wiki paths.
func Homepage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	wikiPageHandler := viewWikiPage(db, templates)
	editWikiPageHandler := editWikiPage(db, templates)

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

		// Paths like /{silo-slug}/wiki/.../edit are handled by the edit page handler.
		if len(parts) >= 3 && parts[len(parts)-1] == "edit" {
			editWikiPageHandler(w, r)
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

// findPageByPath iteratively queries the database to find a page by its hierarchical path.
func findPageByPath(db *sql.DB, siloID int, path []string) (models.Page, error) {
	// If the path is empty, no page can be found.
	if len(path) == 0 {
		return models.Page{}, sql.ErrNoRows
	}

	var page models.Page
	var parentID *int // Starts as NULL for the root page

	for _, slug := range path {
		var err error
		var query string
		if parentID == nil {
			query = "SELECT id, title, current_revision_id, slug, parent_id FROM pages WHERE silo_id = ? AND slug = ? AND parent_id IS NULL"
			err = db.QueryRow(query, siloID, slug).Scan(&page.ID, &page.Title, &page.CurrentRevisionID, &page.Slug, &page.ParentID)
		} else {
			query = "SELECT id, title, current_revision_id, slug, parent_id FROM pages WHERE silo_id = ? AND slug = ? AND parent_id = ?"
			err = db.QueryRow(query, siloID, slug, *parentID).Scan(&page.ID, &page.Title, &page.CurrentRevisionID, &page.Slug, &page.ParentID)
		}

		if err != nil {
			return models.Page{}, err // Return error if page not found or other DB error
		}

		// The current page's ID becomes the next iteration's parent ID.
		pageID := page.ID
		parentID = &pageID
	}
	return page, nil
}

// viewWikiPage handles rendering a single wiki page.
func viewWikiPage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		siloSlug := parts[0]
		pagePath := parts[2:]

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

		page, err := findPageByPath(db, silo.ID, pagePath)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		// The findPageByPath function doesn't populate the Path field, so we do it here.
		page.Path = strings.Join(pagePath, "/")

		var revision models.Revision
		err = db.QueryRow("SELECT content FROM revisions WHERE id = ?", page.CurrentRevisionID).Scan(&revision.Content)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		// Fetch all pages in the current silo, ordered by their position.
		rows, err := db.Query("SELECT id, slug, title, parent_id, position FROM pages WHERE silo_id = ? AND archived_at IS NULL ORDER BY position ASC", silo.ID)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		defer rows.Close()

		var allSiloPages []models.Page
		for rows.Next() {
			var p models.Page
			if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.ParentID, &p.Position); err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", 500)
				return
			}
			allSiloPages = append(allSiloPages, p)
		}

		pageTree := buildPageTree(allSiloPages)

		htmlContentBytes, err := org.New().Parse(strings.NewReader(revision.Content), "").Write(org.NewHTMLWriter())
		if err != nil {
			log.Printf("Error converting org-mode content to HTML: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		data := PageData{
			Silo:        silo,
			Page:        page,
			SiloPages:   pageTree,
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

// editWikiPage handles rendering the wiki page editor.
func editWikiPage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		siloSlug := parts[0]
		pagePath := parts[2 : len(parts)-1] // Exclude the "/edit" part

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

		page, err := findPageByPath(db, silo.ID, pagePath)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		// The findPageByPath function doesn't populate the Path field, so we do it here.
		page.Path = strings.Join(pagePath, "/")

		var revision models.Revision
		err = db.QueryRow("SELECT content FROM revisions WHERE id = ?", page.CurrentRevisionID).Scan(&revision.Content)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		// Fetch all pages in the current silo to build the sidebar tree.
		rows, err := db.Query("SELECT id, slug, title, parent_id, position FROM pages WHERE silo_id = ? AND archived_at IS NULL ORDER BY position ASC", silo.ID)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		defer rows.Close()

		var allSiloPages []models.Page
		for rows.Next() {
			var p models.Page
			if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.ParentID, &p.Position); err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", 500)
				return
			}
			allSiloPages = append(allSiloPages, p)
		}

		pageTree := buildPageTree(allSiloPages)

		data := PageData{
			Silo:        silo,
			Page:        page,
			SiloPages:   pageTree, // Pass the page tree to the template
			Content:     template.HTML(revision.Content),
			ShowSidebar: true, // Explicitly enable the sidebar
		}

		// Use the "edit.html" template set to execute the layout.
		err = templates["edit.html"].ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			log.Println(err)
			if w.Header().Get("Content-Type") == "" {
				http.Error(w, "Internal Server Error", 500)
			}
		}
	}
}

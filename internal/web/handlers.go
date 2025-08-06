package web

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
	"strconv"
	"strings"
	"time"

	"sowing/internal/auth"
	"github.com/niklasfasching/go-org/org"
	"sowing/internal/models"
)

// PageData is a unified struct to hold all possible data for any page.
// SiloPages is now a tree structure instead of a flat list.
type PageData struct {
	ShowSidebar  bool
	Silos        []models.Silo
	Silo         models.Silo
	Page         models.Page    // The current page being viewed
	SiloPages    []*models.Page // The page tree for the sidebar
	Content      template.HTML
	AllSiloPages []models.Page // For the parent dropdown on the new page
	ParentID     int           // The pre-selected parent on the new page
	CurrentUser  *models.User
	IsLoggedIn   bool
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

// previewHandler takes raw org-mode text and returns the rendered HTML.
func previewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

// createSiloHandler handles the creation of a new silo and its initial home page.
func createSiloHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Start a transaction
		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		defer tx.Rollback() // Rollback on error

		// 1. Insert the silo
		res, err := tx.ExecContext(ctx, "INSERT INTO silos (name, slug, cover_image) VALUES (?, ?, ?)", name, slug, coverImageURL)
		if err != nil {
			log.Printf("Error creating silo: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		siloID, _ := res.LastInsertId()

		// 2. Insert the home page (with a temporary revision ID)
		res, err = tx.ExecContext(ctx, "INSERT INTO pages (silo_id, slug, title, current_revision_id) VALUES (?, 'home', 'Home', -1)", siloID)
		if err != nil {
			log.Printf("Error creating home page: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		pageID, _ := res.LastInsertId()

		// 3. Insert the initial revision
		// Assuming user ID 1 is the default author for now
		initialContent := fmt.Sprintf("* Welcome to the %s Silo!", name)
		res, err = tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, 1, 'Initial creation', ?)", pageID, initialContent)
		if err != nil {
			log.Printf("Error creating initial revision: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		revisionID, _ := res.LastInsertId()

		// 4. Update the page with the correct revision ID
		_, err = tx.ExecContext(ctx, "UPDATE pages SET current_revision_id = ? WHERE id = ?", revisionID, pageID)
		if err != nil {
			log.Printf("Error updating page with revision ID: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// uploadHandler handles file uploads from the editor.
func uploadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the multipart form with a 10 MB size limit
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

		// Generate a unique filename based on the file's content and the current time
		// to prevent filename collisions and caching issues.
		fileBytes, _ := io.ReadAll(file)
		hash := sha256.Sum256(fileBytes)
		uniqueFilename := fmt.Sprintf("%s-%d%s",
			hex.EncodeToString(hash[:16]),
			time.Now().Unix(),
			filepath.Ext(handler.Filename))

		// Create the destination file on the server's disk.
		dst, err := os.Create(filepath.Join("uploads", uniqueFilename))
		if err != nil {
			http.Error(w, "Error saving the file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Write the file content to the destination file.
		if _, err := dst.Write(fileBytes); err != nil {
				http.Error(w, "Error writing the file", http.StatusInternalServerError)
				return
			}

		// For now, we won't associate the upload with a page_id.
		// In the future, you could pass the page_id from the editor.
		_, err = db.Exec(
			"INSERT INTO attachments (filename, unique_filename, mime_type, size) VALUES (?, ?, ?, ?)",
			handler.Filename, uniqueFilename, handler.Header.Get("Content-Type"), handler.Size)
		if err != nil {
			log.Printf("Error saving attachment to database: %v", err)
			http.Error(w, "Error saving file metadata", http.StatusInternalServerError)
			return
		}

		// Return the URL of the uploaded file as a JSON response.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"url": "/uploads/%s"}`, uniqueFilename)
	}
}

// loginHandler handles the login page.
func loginHandler(authService *auth.Service, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")
			_, err := authService.Login(w, r, username, password)
			if err != nil {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		err := templates["login.html"].ExecuteTemplate(w, "layout.html", nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
		}
	}
}

// logoutHandler handles the logout page.
func logoutHandler(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authService.Logout(w, r)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// registerHandler handles the registration page.
func registerHandler(authService *auth.Service, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			displayName := r.FormValue("display_name")
			password := r.FormValue("password")

			_, err := authService.RegisterUser(w, r, username, displayName, password)
			if err != nil {
				http.Error(w, "Registration failed", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		err := templates["register.html"].ExecuteTemplate(w, "layout.html", nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
		}
	}
}

// listSilosHandler handles displaying the list of silos (the homepage content).
func listSilosHandler(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This handler is only for GET requests to list silos.
		// POST requests for silo creation are handled by siloCreateHandler.
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

		user := r.Context().Value("user").(*models.User)
		data := PageData{
			Silos:       silos,
			ShowSidebar: false,
			CurrentUser: user,
			IsLoggedIn:  user != nil,
		}

		err = templates["index.html"].ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			log.Println(err)
			if w.Header().Get("Content-Type") == "" {
				http.Error(w, "Internal Server Error", 500)
			}
		}
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

// getPagePathByID recursively finds the full path of a page given its ID.
func getPagePathByID(db *sql.DB, pageID int) (string, error) {
	var slug string
	var parentID sql.NullInt64
	err := db.QueryRow("SELECT slug, parent_id FROM pages WHERE id = ?", pageID).Scan(&slug, &parentID)
	if err != nil {
		return "", err
	}

	if !parentID.Valid {
		return slug, nil
	}

	parentPath, err := getPagePathByID(db, int(parentID.Int64))
	if err != nil {
		return "", err
	}

	return parentPath + "/" + slug, nil
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

		htmlContentString, err := org.New().Parse(strings.NewReader(revision.Content), "").Write(org.NewHTMLWriter())
		if err != nil {
			log.Printf("Error converting org-mode content to HTML: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		user, _ := r.Context().Value("user").(*models.User)
		data := PageData{
			Silo:        silo,
			Page:        page,
			SiloPages:   pageTree,
			Content:     template.HTML(htmlContentString),
			ShowSidebar: true,
			CurrentUser: user,
			IsLoggedIn:  user != nil,
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

		user, _ := r.Context().Value("user").(*models.User)
		data := PageData{
			Silo:        silo,
			Page:        page,
			SiloPages:   pageTree, // Pass the page tree to the template
			Content:     template.HTML(revision.Content),
			ShowSidebar: true, // Explicitly enable the sidebar
			CurrentUser: user,
			IsLoggedIn:  user != nil,
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

// newWikiPage handles rendering the form for a new wiki page.
func newWikiPage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		siloSlug := parts[0]

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

		// Fetch all pages in the current silo to build the sidebar and parent dropdown.
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

		// Check for a pre-selected parent from the query parameter.
		parentID := 0
		parentIDStr := r.URL.Query().Get("parent")
		if parentIDStr != "" {
			parentID, _ = strconv.Atoi(parentIDStr)
		}

		user, _ := r.Context().Value("user").(*models.User)
		data := PageData{
			Silo:         silo,
			SiloPages:    pageTree,
			AllSiloPages: allSiloPages,
			ParentID:     parentID,
			ShowSidebar:  true,
			CurrentUser:  user,
			IsLoggedIn:   user != nil,
		}

		err = templates["new.html"].ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			log.Println(err)
			if w.Header().Get("Content-Type") == "" {
				http.Error(w, "Internal Server Error", 500)
			}
		}
	}
}

// createPageHandler handles the POST request for creating a new page.
func createPageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		siloSlug := parts[0]

		var silo models.Silo
		err := db.QueryRow("SELECT id FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		title := r.PostFormValue("title")
		slug := r.PostFormValue("slug")
		content := r.PostFormValue("content")
		comment := r.PostFormValue("comment")
		parentID, _ := strconv.Atoi(r.PostFormValue("parent"))

		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		defer tx.Rollback()

		var parentIDPtr *int
		if parentID != 0 {
			parentIDPtr = &parentID
		}

		res, err := tx.ExecContext(ctx, "INSERT INTO pages (silo_id, parent_id, slug, title, current_revision_id) VALUES (?, ?, ?, ?, -1)", silo.ID, parentIDPtr, slug, title)
		if err != nil {
			log.Printf("Error creating page: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		pageID, _ := res.LastInsertId()

		res, err = tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, ?, ?, ?)", pageID, user.ID, comment, content)
		if err != nil {
			log.Printf("Error creating revision: %v", err)
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

		var redirectPath string
		if parentID != 0 {
			parentPath, err := getPagePathByID(db, parentID)
			if err != nil {
				log.Printf("Error getting parent path for redirect: %v", err)
				http.Error(w, "Internal Server Error", 500)
				return
			}
			redirectPath = parentPath + "/" + slug
		} else {
			redirectPath = slug
		}

		http.Redirect(w, r, fmt.Sprintf("/%s/wiki/%s", siloSlug, redirectPath), http.StatusSeeOther)
	}
}

// savePageHandler handles the POST request for editing a page.
func savePageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		siloSlug := parts[0]
		pagePath := parts[2 : len(parts)-1]

		var silo models.Silo
		err := db.QueryRow("SELECT id FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		page, err := findPageByPath(db, silo.ID, pagePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}
		content := r.PostFormValue("content")
		comment := r.PostFormValue("comment")

		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		defer tx.Rollback()

		res, err := tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, ?, ?, ?)", page.ID, user.ID, comment, content)
		if err != nil {
			log.Printf("Error creating revision: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		revisionID, _ := res.LastInsertId()

		_, err = tx.ExecContext(ctx, "UPDATE pages SET current_revision_id = ? WHERE id = ?", revisionID, page.ID)
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

		http.Redirect(w, r, fmt.Sprintf("/%s/wiki/%s", siloSlug, strings.Join(pagePath, "/")), http.StatusSeeOther)
	}
}

// Homepage sets up the main router with protected and unprotected routes.
func Homepage(db *sql.DB, templates map[string]*template.Template) http.HandlerFunc {
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo)

	// Define all raw (unwrapped) handlers
	loginH := loginHandler(authService, templates)
	logoutH := logoutHandler(authService)
	
	previewH := previewHandler
	uploadH := uploadHandler(db)
	siloCreateH := createSiloHandler(db)
	listSilosH := listSilosHandler(db, templates)
	viewWikiPageH := viewWikiPage(db, templates)
	editWikiPageH := editWikiPage(db, templates)
	savePageH := savePageHandler(db)
	newWikiPageH := newWikiPage(db, templates)
	createPageH := createPageHandler(db)

	// Create a main mux
	mainMux := http.NewServeMux()

	// Register unprotected routes directly to the main mux
	mainMux.HandleFunc("/login", loginH)
	mainMux.HandleFunc("/logout", logoutH)
	

	// Create a new mux for protected routes
	protectedMux := http.NewServeMux()

	// Register all protected handlers to the protected mux
	protectedMux.HandleFunc("/_preview", previewH)
	protectedMux.HandleFunc("/upload", uploadH)
	protectedMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		// Handle POST for silo creation on the root path
		if path == "/" && r.Method == http.MethodPost {
			siloCreateH.ServeHTTP(w, r)
			return
		}

		// Handle GET for listing silos on the root path
		if path == "/" && r.Method == http.MethodGet {
			listSilosH.ServeHTTP(w, r)
			return
		}

		// Paths like /{silo-slug}/new are for creating a new page.
		if len(parts) == 2 && parts[1] == "new" {
			if r.Method == http.MethodPost {
				createPageH.ServeHTTP(w, r)
			} else {
				newWikiPageH.ServeHTTP(w, r)
			}
			return
		}

		// Paths like /{silo-slug}/wiki/.../edit are handled by the edit page handler.
		if len(parts) >= 3 && parts[len(parts)-1] == "edit" {
			if r.Method == http.MethodPost {
				savePageH.ServeHTTP(w, r)
			} else {
				editWikiPageH.ServeHTTP(w, r)
			}
			return
		}

		// Paths like /{silo-slug}/wiki/... are handled by the wiki page handler.
		if len(parts) >= 2 && parts[1] == "wiki" {
			viewWikiPageH.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})

	// Wrap the protected mux with the RequireLogin middleware
	protectedHandler := authService.RequireLogin(protectedMux)

	// Register the protected handler as the catch-all for the main mux
	mainMux.Handle("/", protectedHandler)

	// Apply the WithUser middleware as the outermost middleware
	return authService.WithUser(mainMux).ServeHTTP
}

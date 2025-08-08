package controller

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sowing/internal/models"
	"sowing/internal/web/viewmodels"

	"github.com/niklasfasching/go-org/org"
)

// Page provides page handlers
type Page struct {
	DB        *sql.DB
	Templates map[string]*template.Template
}

// Register registers the page routes
func (p *Page) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{siloSlug}/wiki/{pagePath...}", p.view)
	mux.HandleFunc("GET /{siloSlug}/new", p.new)
	mux.HandleFunc("POST /{siloSlug}/new", p.create)
	mux.HandleFunc("GET /{siloSlug}/edit/{pagePath...}", p.edit)
	mux.HandleFunc("POST /{siloSlug}/edit/{pagePath...}", p.save)
	mux.HandleFunc("POST /{siloSlug}/delete/{pagePath...}", p.delete)
}

func (p *Page) view(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")
	pagePath := r.PathValue("pagePath")

	var silo models.Silo
	err := p.DB.QueryRow("SELECT id, slug, name FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID, &silo.Slug, &silo.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	page, err := findPageByPath(p.DB, silo.ID, strings.Split(pagePath, "/"))
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	page.Path = pagePath

	var revision models.Revision
	err = p.DB.QueryRow("SELECT content FROM revisions WHERE id = ?", page.CurrentRevisionID).Scan(&revision.Content)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	rows, err := p.DB.Query("SELECT id, slug, title, parent_id, position FROM pages WHERE silo_id = ? AND archived_at IS NULL ORDER BY position ASC", silo.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer rows.Close()

	var allSiloPages []models.Page
	for rows.Next() {
		var page models.Page
		if err := rows.Scan(&page.ID, &page.Slug, &page.Title, &page.ParentID, &page.Position); err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		allSiloPages = append(allSiloPages, page)
	}

	pageTree := buildPageTree(allSiloPages)

	htmlContentString, err := org.New().Parse(strings.NewReader(revision.Content), "").Write(org.NewHTMLWriter())
	if err != nil {
		log.Printf("Error converting org-mode content to HTML: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silo:        silo,
		Page:        page,
		SiloPages:   pageTree,
		Content:     template.HTML(htmlContentString),
		ShowSidebar: true,
		CurrentUser: user,
		IsLoggedIn:  user != nil,
	}

	err = p.Templates["view.html"].ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println(err)
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, "Internal Server Error", 500)
		}
	}
}

func (p *Page) new(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")

	var silo models.Silo
	err := p.DB.QueryRow("SELECT id, slug, name FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID, &silo.Slug, &silo.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	rows, err := p.DB.Query("SELECT id, slug, title, parent_id, position FROM pages WHERE silo_id = ? AND archived_at IS NULL ORDER BY position ASC", silo.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer rows.Close()

	var allSiloPages []models.Page
	for rows.Next() {
		var page models.Page
		if err := rows.Scan(&page.ID, &page.Slug, &page.Title, &page.ParentID, &page.Position); err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		allSiloPages = append(allSiloPages, page)
	}

	pageTree := buildPageTree(allSiloPages)

	parentID := 0
	parentIDStr := r.URL.Query().Get("parent")
	if parentIDStr != "" {
		parentID, _ = strconv.Atoi(parentIDStr)
	}

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silo:         silo,
		SiloPages:    pageTree,
		AllSiloPages: allSiloPages,
		ParentID:     parentID,
		ShowSidebar:  true,
		CurrentUser:  user,
		IsLoggedIn:   user != nil,
	}

	err = p.Templates["new.html"].ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println(err)
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, "Internal Server Error", 500)
		}
	}
}

func (p *Page) create(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")

	var silo models.Silo
	err := p.DB.QueryRow("SELECT id FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID)
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
	tx, err := p.DB.BeginTx(ctx, nil)
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
		parentPath, err := getPagePathByID(p.DB, parentID)
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

func (p *Page) edit(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")
	pagePath := r.PathValue("pagePath")

	var silo models.Silo
	err := p.DB.QueryRow("SELECT id, slug, name FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID, &silo.Slug, &silo.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	page, err := findPageByPath(p.DB, silo.ID, strings.Split(pagePath, "/"))
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	page.Path = pagePath

	var revision models.Revision
	err = p.DB.QueryRow("SELECT content FROM revisions WHERE id = ?", page.CurrentRevisionID).Scan(&revision.Content)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	rows, err := p.DB.Query("SELECT id, slug, title, parent_id, position FROM pages WHERE silo_id = ? AND archived_at IS NULL ORDER BY position ASC", silo.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer rows.Close()

	var allSiloPages []models.Page
	for rows.Next() {
		var page models.Page
		if err := rows.Scan(&page.ID, &page.Slug, &page.Title, &page.ParentID, &page.Position); err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		allSiloPages = append(allSiloPages, page)
	}

	pageTree := buildPageTree(allSiloPages)

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silo:        silo,
		Page:        page,
		SiloPages:   pageTree,
		Content:     template.HTML(revision.Content),
		ShowSidebar: true,
		CurrentUser: user,
		IsLoggedIn:  user != nil,
	}

	err = p.Templates["edit.html"].ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println(err)
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, "Internal Server Error", 500)
		}
	}
}

func (p *Page) save(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")
	pagePath := r.PathValue("pagePath")

	var silo models.Silo
	err := p.DB.QueryRow("SELECT id FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	page, err := findPageByPath(p.DB, silo.ID, strings.Split(pagePath, "/"))
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
	tx, err := p.DB.BeginTx(ctx, nil)
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

	http.Redirect(w, r, fmt.Sprintf("/%s/wiki/%s", siloSlug, pagePath), http.StatusSeeOther)
}

func (p *Page) delete(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")
	pagePath := r.PathValue("pagePath")

	var silo models.Silo
	err := p.DB.QueryRow("SELECT id FROM silos WHERE slug = ?", siloSlug).Scan(&silo.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	page, err := findPageByPath(p.DB, silo.ID, strings.Split(pagePath, "/"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	_, err = p.DB.Exec("UPDATE pages SET archived_at = ? WHERE id = ?", time.Now(), page.ID)
	if err != nil {
		log.Printf("Error archiving page: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s/wiki/home", siloSlug), http.StatusSeeOther)
}

// findPageByPath iteratively queries the database to find a page by its hierarchical path.
func findPageByPath(db *sql.DB, siloID int, path []string) (models.Page, error) {
	if len(path) == 0 {
		return models.Page{}, sql.ErrNoRows
	}

	var page models.Page
	var parentID *int

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
			return models.Page{}, err
		}

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

// buildPageTree takes a flat list of pages (already sorted by position)
// and organizes them into a hierarchical tree.
func buildPageTree(pages []models.Page) []*models.Page {
	pageMap := make(map[int]*models.Page)
	for i := range pages {
		p := pages[i]
		pageMap[p.ID] = &p
	}

	var rootPages []*models.Page
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

	constructPath(rootPages, "")

	return rootPages
}

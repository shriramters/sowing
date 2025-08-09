package controller

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"sowing/internal/models"
	"sowing/internal/page"
	"sowing/internal/silo"
	"sowing/internal/web/viewmodels"

	"github.com/niklasfasching/go-org/org"
)

// Page provides page handlers
type Page struct {
	PageRepo  *page.Repository
	SiloRepo  *silo.Repository
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

	silo, err := p.SiloRepo.FindBySlug(siloSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	page, err := p.PageRepo.FindByPath(silo.ID, strings.Split(pagePath, "/"))
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

	content, err := p.PageRepo.GetRevisionContent(page.CurrentRevisionID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	allSiloPages, err := p.PageRepo.ListBySilo(silo.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	pageTree := buildPageTree(allSiloPages)

	htmlContentString, err := org.New().Parse(strings.NewReader(content), "").Write(org.NewHTMLWriter())
	if err != nil {
		log.Printf("Error converting org-mode content to HTML: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silo:        *silo,
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
	}
}

func (p *Page) new(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")

	silo, err := p.SiloRepo.FindBySlug(siloSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	allSiloPages, err := p.PageRepo.ListBySilo(silo.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	pageTree := buildPageTree(allSiloPages)

	parentID := 0
	parentIDStr := r.URL.Query().Get("parent")
	if parentIDStr != "" {
		parentID, _ = strconv.Atoi(parentIDStr)
	}

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silo:         *silo,
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
	}
}

func (p *Page) create(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")

	silo, err := p.SiloRepo.FindBySlug(siloSlug)
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

	page := &models.Page{
		SiloID: silo.ID,
		Slug:   slug,
		Title:  title,
	}
	if parentID != 0 {
		page.ParentID = &parentID
	}

	revision := &models.Revision{
		AuthorID: user.ID,
		Comment:  &comment,
		Content:  content,
	}

	_, err = p.PageRepo.Create(r.Context(), page, revision)
	if err != nil {
		log.Printf("Error creating page: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	var redirectPath string
	if parentID != 0 {
		parentPath, err := p.PageRepo.GetPathByID(parentID)
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

	silo, err := p.SiloRepo.FindBySlug(siloSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	page, err := p.PageRepo.FindByPath(silo.ID, strings.Split(pagePath, "/"))
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

	content, err := p.PageRepo.GetRevisionContent(page.CurrentRevisionID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	allSiloPages, err := p.PageRepo.ListBySilo(silo.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	pageTree := buildPageTree(allSiloPages)

	user, _ := r.Context().Value("user").(*models.User)
	data := viewmodels.PageData{
		Silo:        *silo,
		Page:        page,
		SiloPages:   pageTree,
		Content:     template.HTML(content),
		ShowSidebar: true,
		CurrentUser: user,
		IsLoggedIn:  user != nil,
	}

	err = p.Templates["edit.html"].ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println(err)
	}
}

func (p *Page) save(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")
	pagePath := r.PathValue("pagePath")

	silo, err := p.SiloRepo.FindBySlug(siloSlug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	page, err := p.PageRepo.FindByPath(silo.ID, strings.Split(pagePath, "/"))
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

	revision := &models.Revision{
		AuthorID: user.ID,
		Comment:  &comment,
		Content:  content,
	}

	err = p.PageRepo.CreateRevision(r.Context(), revision, page.ID)
	if err != nil {
		log.Printf("Error creating revision: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s/wiki/%s", siloSlug, pagePath), http.StatusSeeOther)
}

func (p *Page) delete(w http.ResponseWriter, r *http.Request) {
	siloSlug := r.PathValue("siloSlug")
	pagePath := r.PathValue("pagePath")

	silo, err := p.SiloRepo.FindBySlug(siloSlug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	page, err := p.PageRepo.FindByPath(silo.ID, strings.Split(pagePath, "/"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = p.PageRepo.Delete(page.ID)
	if err != nil {
		log.Printf("Error archiving page: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s/wiki/home", siloSlug), http.StatusSeeOther)
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

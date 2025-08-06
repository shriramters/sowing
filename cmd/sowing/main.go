package main

import (
	"database/sql"
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"

	"sowing/internal/database"
	"sowing/internal/web"
)

type Application struct {
	DB        *sql.DB
	Templates map[string]*template.Template
}

func main() {
	var dsn = flag.String("dsn", "sowing.db", "The database connection string.")
	flag.Parse()

	db, err := database.New(*dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	log.Println("database migrated")

	// Create a directory for file uploads if it doesn't exist.
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatal(err)
	}

	handleAdminCommands(db)

	if len(flag.Args()) > 0 && flag.Arg(0) == "admin" {
		os.Exit(0)
	}

	// Create a map to hold the different, isolated template sets.
	templates := make(map[string]*template.Template)

	// Create a FuncMap to add helper functions to the templates.
	// The "dict" function allows passing multiple, named values to a template,
	// which is perfect for complex or recursive templates.
	funcMap := template.FuncMap{
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

	// Create a template set for the index page, including the new FuncMap.
	templates["index.html"] = template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"internal/web/templates/layout.html",
		"internal/web/templates/index.html",
		"internal/web/templates/sidebar.html",
	))

	// Create a separate template set for the wiki view page, also with the FuncMap.
	templates["view.html"] = template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"internal/web/templates/layout.html",
		"internal/web/templates/view.html",
		"internal/web/templates/sidebar.html",
	))

	// Create a template set for the wiki edit page.
	templates["edit.html"] = template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"internal/web/templates/layout.html",
		"internal/web/templates/edit.html",
		"internal/web/templates/sidebar.html",
	))

	// Create a template set for the new wiki page.
	templates["new.html"] = template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"internal/web/templates/layout.html",
		"internal/web/templates/new.html",
		"internal/web/templates/sidebar.html",
	))

	app := &Application{
		DB:        db,
		Templates: templates,
	}

	// Create a new ServeMux to explicitly control routing.
	mux := http.NewServeMux()

	// Register the specific file servers first.
	mux.Handle("/static/", http.StripPrefix("/static/", web.StaticFileServer()))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// Register the main application handler for all other routes.
	mux.HandleFunc("/", web.Homepage(app.DB, app.Templates))

	log.Println("starting server on :8080")
	// Use the new mux as the handler.
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

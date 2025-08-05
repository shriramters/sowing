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
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
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

	app := &Application{
		DB:        db,
		Templates: templates,
	}

	// Pass the map of templates to the handler.
	http.HandleFunc("/", web.Homepage(app.DB, app.Templates))
	http.Handle("/static/", http.StripPrefix("/static/", web.StaticFileServer()))

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

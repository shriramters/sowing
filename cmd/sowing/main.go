package main

import (
	"database/sql"
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
	Templates *template.Template
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

	templates, err := template.ParseGlob("internal/web/templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	app := &Application{
		DB:        db,
		Templates: templates,
	}

	http.HandleFunc("/", web.Homepage(app.DB, app.Templates))
	http.Handle("/static/", http.StripPrefix("/static/", web.StaticFileServer()))

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
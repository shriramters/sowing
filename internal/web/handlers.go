package web

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"sowing/internal/models"
)

func Homepage(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		templates.ExecuteTemplate(w, "layout.html", struct{ Silos []models.Silo }{Silos: silos})
	}
}

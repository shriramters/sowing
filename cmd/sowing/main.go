package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"sowing/internal/auth"
	"sowing/internal/database"
	"sowing/internal/models"
	"sowing/internal/web"

	"golang.org/x/crypto/bcrypt"
)



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

	// Get session key from environment variable
	sessionKey := os.Getenv("SOWING_SESSION_KEY")
	if sessionKey == "" {
		log.Fatal("SOWING_SESSION_KEY environment variable not set")
	}

	// Initialize the session store
	if err := auth.InitSessionStore(sessionKey); err != nil {
		log.Fatal(err)
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

	// Create a template set for the login page.
	templates["login.html"] = template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"internal/web/templates/layout.html",
		"internal/web/templates/login.html",
		"internal/web/templates/sidebar.html",
	))

	// Create a template set for the register page.
	templates["register.html"] = template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"internal/web/templates/layout.html",
		"internal/web/templates/register.html",
		"internal/web/templates/sidebar.html",
	))

	server := web.NewServer(db, templates)

	if err := http.ListenAndServe(":8080", server); err != nil {
		log.Fatal(err)
	}

	
}
func handleAdminCommands(db *sql.DB) {
	args := flag.Args()
	if len(args) == 0 || args[0] != "admin" {
		return
	}

	// Shift args to remove "admin"
	args = args[1:]

	if len(args) == 0 {
		fmt.Println("Usage: sowing admin <command>")
		os.Exit(1)
	}

	switch args[0] {
	case "create-user":
		createCmd := flag.NewFlagSet("create-user", flag.ExitOnError)
		username := createCmd.String("username", "", "The username for the new user.")
		displayName := createCmd.String("display-name", "", "The display name for the new user.")
		password := createCmd.String("password", "", "The password for the new user.")
		createCmd.Parse(args[1:])

		if *username == "" || *displayName == "" || *password == "" {
			fmt.Println("Username, display name, and password are required.")
			os.Exit(1)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Error hashing password: %v", err)
		}
		passwordHash := string(hashedPassword)

		authRepo := auth.NewRepository(db)
		user := &models.User{
			Username:    *username,
			DisplayName: *displayName,
		}
		identity := &models.Identity{
			Provider:       "local",
			ProviderUserID: *username,
			PasswordHash:   &passwordHash,
		}

		if err := authRepo.CreateUser(user, identity); err != nil {
			log.Fatalf("Error creating user: %v", err)
		}

		fmt.Println("User created successfully.")
		os.Exit(0)
	case "create-silo":
		siloCmd := flag.NewFlagSet("create-silo", flag.ExitOnError)
		name := siloCmd.String("name", "", "The name of the new silo.")
		slug := siloCmd.String("slug", "", "The slug for the new silo.")
		siloCmd.Parse(args[1:])

		if *name == "" || *slug == "" {
			fmt.Println("Name and slug are required.")
			os.Exit(1)
		}

		_, err := db.Exec("INSERT INTO silos (name, slug) VALUES (?, ?)", *name, *slug)
		if err != nil {
			log.Fatalf("Error creating silo: %v", err)
		}

		fmt.Println("Silo created successfully.")
		os.Exit(0)
	default:
		fmt.Println("Unknown admin command:", args[0])
		os.Exit(1)
	}
}
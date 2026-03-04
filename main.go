package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"podpal/internal/downloader"
	"podpal/web"
)

func main() {
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	dl := downloader.New("cache")
	handler := web.New(tmpl, dl)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Static files
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// App routes
	r.Mount("/", handler.Routes())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Podpal starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"fitnesscoach/db"
	"fitnesscoach/handlers"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize DB
	if err := db.InitDB(); err != nil {
		log.Fatal("❌ Database connection failed:", err)
	}

	// Serve static files (CSS, JS, etc.)
	fs := http.FileServer(http.Dir("resources"))
	http.Handle("/resources/", http.StripPrefix("/resources/", fs))

	// Routes
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/home", handlers.HomePageHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Start server
	fmt.Println("✅ Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

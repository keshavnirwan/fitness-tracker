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
	http.HandleFunc("/userinfo", handlers.UserInfoHandler)
	http.HandleFunc("/userdash", handlers.UserDashHandler)
	http.HandleFunc("/weight", handlers.WeightHandler)
	http.HandleFunc("/cardio", handlers.CardioHandler)
	http.HandleFunc("/update-progress", handlers.UpdateProgressHandler)
	http.HandleFunc("/coachdash", handlers.HandleConnections)
	go handlers.HandleMessages()

	// Start server
	fmt.Println("✅ Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

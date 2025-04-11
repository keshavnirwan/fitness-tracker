package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fitnesscoach/db"
	"fmt"
	"net/http"
	"text/template"

	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte(generateRandomString(32)))

type WebPageData struct {
	WebsiteTitle        string
	H1Heading           string
	BodyParagraphText   string
	PostResponseMessage string
}

func templateRender(w http.ResponseWriter, data WebPageData, file string) {
	tmpl, err := template.ParseFiles("templates/" + file + ".html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func templateRenderMap(w http.ResponseWriter, m map[string]string, file string) {
	tmpl, err := template.ParseFiles("templates/" + file + ".html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, m)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	page := WebPageData{
		WebsiteTitle:      "Register",
		H1Heading:         "Create an Account",
		BodyParagraphText: "Sign up to get started!",
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		email := r.FormValue("applicantemail")
		password := r.FormValue("password")
		role := r.FormValue("role")

		_, err := db.CreateUser(username, email, password, role)
		if err != nil {
			page.PostResponseMessage = "Registration failed. Try a different username."
		} else {
			page.PostResponseMessage = fmt.Sprintf("✅ Successfully registered! Welcome, %s.", username)
		}
	}

	templateRender(w, page, "register")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	webPageData := WebPageData{
		WebsiteTitle:      "Login Page",
		H1Heading:         "Enter Your Login Details",
		BodyParagraphText: "",
	}

	if r.Method == http.MethodGet {
		templateRender(w, webPageData, "login")
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		isValid, role, err := db.ValidateUser(username, password)
		if err != nil || !isValid {
			webPageData.PostResponseMessage = "❌ Wrong username or password. Please try again."
			templateRender(w, webPageData, "login")
			return
		}

		session, _ := store.Get(r, "fitnesscoach.com")
		session.Values["authenticatedUser"] = true
		session.Values["username"] = username
		session.Values["role"] = role
		session.Save(r, w)

		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "fitnesscoach.com")
	isAuthenticated, ok := session.Values["authenticatedUser"].(bool)
	if !ok || !isAuthenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := session.Values["username"].(string)
	role := session.Values["role"].(string)

	data := map[string]string{
		"WebsiteTitle":    "Fitness Dashboard",
		"HomePageHeading": "Welcome to Your Dashboard",
		"WelcomeMessage":  fmt.Sprintf("Welcome, %s!", username),
	}

	if role == "coach" {
		templateRenderMap(w, data, "coachdash")
	} else {
		templateRenderMap(w, data, "userdash")
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "fitnesscoach.com")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "fallbacksecret"
	}
	return hex.EncodeToString(bytes)
}

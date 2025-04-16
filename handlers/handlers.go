package handlers

import (
	"crypto/rand"

	"encoding/hex"
	"fitnesscoach/db"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
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

		fullName := r.FormValue("full_name")
		ageStr := r.FormValue("age")
		gender := r.FormValue("gender")
		heightStr := r.FormValue("height_cm")
		weightStr := r.FormValue("weight_kg")

		age, _ := strconv.Atoi(ageStr)
		height, _ := strconv.ParseFloat(heightStr, 64)
		weight, _ := strconv.ParseFloat(weightStr, 64)

		userID, err := db.CreateUser(username, email, password, role)
		if err != nil {
			page.PostResponseMessage = "Registration failed. Try a different username or email."
		} else {
			err := db.SaveUserInfoByID(userID, fullName, age, gender, height, weight)
			if err != nil {
				page.PostResponseMessage = "User registered, but failed to save personal info."
			} else {
				page.PostResponseMessage = fmt.Sprintf("✅ Successfully registered! Welcome, %s.", username)
			}
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

	if role == "coach" {
		data := map[string]string{
			"WebsiteTitle":    "Coach Dashboard",
			"HomePageHeading": "Welcome Coach",
			"WelcomeMessage":  fmt.Sprintf("Hello, Coach %s!", username),
		}
		templateRenderMap(w, data, "coachdash")
		return
	}

	userInfo, err := db.GetUserInfoByUsername(username)
	if err != nil || userInfo.FullName == "" {
		data := map[string]string{
			"WebsiteTitle": "Complete Your Profile",
			"Username":     username,
		}
		templateRenderMap(w, data, "userinfoform")
		return
	}

	http.Redirect(w, r, "/userdash", http.StatusSeeOther)
}

func UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	session, _ := store.Get(r, "fitnesscoach.com")
	username := session.Values["username"].(string)

	fullName := r.FormValue("full_name")
	age, _ := strconv.Atoi(r.FormValue("age"))
	gender := r.FormValue("gender")
	height, _ := strconv.ParseFloat(r.FormValue("height_cm"), 64)
	weight, _ := strconv.ParseFloat(r.FormValue("weight_kg"), 64)

	err := db.SaveUserInfo(username, fullName, age, gender, height, weight)
	if err != nil {
		http.Error(w, "Failed to save user info", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/userdash?updated=true", http.StatusSeeOther)
}

func UserDashHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "fitnesscoach.com")
	isAuthenticated, ok := session.Values["authenticatedUser"].(bool)
	if !ok || !isAuthenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := session.Values["username"].(string)
	userInfo, err := db.GetUserInfoByUsername(username)
	if err != nil {
		http.Error(w, "Unable to load dashboard", http.StatusInternalServerError)
		return
	}

	message := ""
	if r.URL.Query().Get("updated") == "true" {
		message = "Information updated successfully!"
	}

	data := map[string]string{
		"WebsiteTitle": "User Dashboard",
		"FullName":     userInfo.FullName,
		"Age":          strconv.Itoa(userInfo.Age),
		"Gender":       userInfo.Gender,
		"HeightCM":     fmt.Sprintf("%.1f", userInfo.Height),
		"WeightKG":     fmt.Sprintf("%.1f", userInfo.Weight),
		"Username":     username,
		"Message":      message,
	}

	templateRenderMap(w, data, "userdash")
}

func WeightHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "fitnesscoach.com")
	isAuthenticated, ok := session.Values["authenticatedUser"].(bool)
	if !ok || !isAuthenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]string{
		"WebsiteTitle": "Weight Training Goals",
	}
	templateRenderMap(w, data, "weight")
}

func CardioHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "fitnesscoach.com")
	isAuthenticated, ok := session.Values["authenticatedUser"].(bool)
	if !ok || !isAuthenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]string{
		"WebsiteTitle": "Cardio Training Goals",
	}
	templateRenderMap(w, data, "cardio")
}

func UpdateProgressHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username := session.Values["username"].(string)

	userID, err := db.GetUserIDByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	workout := r.FormValue("workout") == "true"
	meals := r.FormValue("meals") == "true"
	water := r.FormValue("water") == "true"

	err = db.SaveOrUpdateProgress(userID, workout, meals, water)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// WebSocket Chat Implementation

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[string]*websocket.Conn)     // username -> connection
var connections = make(map[*websocket.Conn]string) // connection -> username
var broadcast = make(chan Message)

type Message struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	session, _ := store.Get(r, "fitnesscoach.com")
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		ws.Close()
		return
	}

	clients[username] = ws
	connections[ws] = username
	defer func() {
		delete(clients, username)
		delete(connections, ws)
		ws.Close()
	}()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		msg.Sender = username
		broadcast <- msg
	}
}

func HandleMessages() {
	for {
		msg := <-broadcast
		receiverConn, ok := clients[msg.Receiver]
		if ok {
			err := receiverConn.WriteJSON(msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				receiverConn.Close()
				delete(clients, msg.Receiver)
			}
		} else {
			log.Printf("User %s is not connected", msg.Receiver)
		}
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

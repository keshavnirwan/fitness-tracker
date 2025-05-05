package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fitnesscoach/db"
	"fmt"
	"io"
	"io/ioutil"
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

func GetAllUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	users, err := db.GetAllUserInfo()
	if err != nil {
		log.Println("❌ Failed to fetch user info:", err)
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
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

// CoachChatHandler serves the coachchat.html template
func CoachChatHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the template
	tmpl, err := template.ParseFiles("templates/coachchat.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	// Render the template
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
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

func UpdateProfilePageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/update-profile.html")
		if err != nil {
			http.Error(w, "Failed to load page", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	} else if r.Method == http.MethodPost {
		UpdateProfileHandler(w, r)
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var profileData struct {
		FullName string  `json:"fullName"`
		Age      int     `json:"age"`
		Gender   string  `json:"gender"`
		Height   float64 `json:"height"`
		Weight   float64 `json:"weight"`
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log the raw request body for debugging
	log.Printf("Raw Request Body: %s", body)

	// Decode the JSON into the struct
	err = json.Unmarshal(body, &profileData)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log the decoded profile data
	log.Printf("Decoded Profile Data: %+v", profileData)

	session, _ := store.Get(r, "fitnesscoach.com")
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = db.SaveUserInfo(username, profileData.FullName, profileData.Age, profileData.Gender, profileData.Height, profileData.Weight)
	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated successfully"})
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

// Message structure for WebSocket communication
type Message struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`
}

// HandleConnections handles WebSocket connections for both coach and user
func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Retrieve username from session
	session, _ := store.Get(r, "fitnesscoach.com")
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		ws.Close()
		return
	}

	// Register the connection
	clients[username] = ws
	connections[ws] = username
	defer func() {
		delete(clients, username)
		delete(connections, ws)
		ws.Close()
	}()

	// Listen for incoming messages
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		msg.Sender = username // Set the sender to the current user
		broadcast <- msg      // Send the message to the broadcast channel
	}
}

// HandleMessages handles broadcasting messages to specific users
func HandleMessages() {
	for {
		msg := <-broadcast

		// Get sender and receiver IDs
		senderID, err1 := db.GetUserIDByUsername(msg.Sender)
		receiverID, err2 := db.GetUserIDByUsername(msg.Receiver)

		if err1 != nil || err2 != nil {
			log.Printf("❌ Failed to get user IDs: %v, %v", err1, err2)
			continue
		}

		// Save message to the database
		err := db.SendMessage(senderID, receiverID, msg.Content)
		if err != nil {
			log.Printf("❌ Failed to save message: %v", err)
		} else {
			log.Printf("💬 Message saved to DB: %s -> %s", msg.Sender, msg.Receiver)
		}

		// Deliver message to the receiver if online
		if receiverConn, ok := clients[msg.Receiver]; ok {
			if err := receiverConn.WriteJSON(msg); err != nil {
				log.Printf("WebSocket write error: %v", err)
				receiverConn.Close()
				delete(clients, msg.Receiver)
			}
		} else {
			log.Printf("📭 %s is not connected", msg.Receiver)
		}
	}
}
func ChatHistoryHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "fitnesscoach.com")
	currentUsername := session.Values["username"].(string)

	senderID, err := db.GetUserIDByUsername(currentUsername)
	if err != nil {
		log.Printf("❌ Invalid sender: %v", err)
		http.Error(w, "Invalid sender", http.StatusBadRequest)
		return
	}

	receiver := r.URL.Query().Get("receiver")
	if receiver == "" {
		log.Println("❌ Receiver username is missing")
		http.Error(w, "Receiver username is required", http.StatusBadRequest)
		return
	}

	receiverID, err := db.GetUserIDByUsername(receiver)
	if err != nil {
		log.Printf("❌ Invalid receiver: %v", err)
		http.Error(w, "Invalid receiver", http.StatusBadRequest)
		return
	}

	log.Printf("🔍 Fetching messages between senderID: %d and receiverID: %d", senderID, receiverID)

	messages, err := db.GetMessagesBetweenUsers(senderID, receiverID)
	if err != nil {
		log.Printf("❌ Failed to fetch chat history: %v", err)
		http.Error(w, "Failed to fetch chat history", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Messages fetched: %v", messages)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
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

const cohereAPIKey = "aflWffC10AK0waH2h7KkeMXkRoR8Igj8Y2ofiGKw"
const cohereURL = "https://api.cohere.ai/v1/generate"

type CohereRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type CohereResponse struct {
	Generations []struct {
		Text string `json:"text"`
	} `json:"generations"`
}

func AiChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/chat.html"))
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		prompt := r.FormValue("prompt")

		reqBody := CohereRequest{
			Model:       "command", // Or "command-nightly"
			Prompt:      prompt,
			MaxTokens:   100,
			Temperature: 0.7,
		}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", cohereURL, bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+cohereAPIKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Cohere API request failed", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		respBody, _ := ioutil.ReadAll(resp.Body)
		var cohereResp CohereResponse
		json.Unmarshal(respBody, &cohereResp)

		tmpl := template.Must(template.ParseFiles("templates/chat.html"))
		tmpl.Execute(w, map[string]string{
			"Response": cohereResp.Generations[0].Text,
		})
	}
}

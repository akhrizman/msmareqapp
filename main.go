package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}
	LoadConfig()

	initDB()
	initSessionStore()

	r := mux.NewRouter()

	// public
	r.HandleFunc("/", HomeGetHandler).Methods("GET")
	r.HandleFunc("/login", LoginPageGetHandler).Methods("GET")
	r.HandleFunc("/login", LoginHandler).Methods("POST")

	// change password (requires being logged in)
	r.Handle("/change-password", RequireLogin(http.HandlerFunc(ChangePasswordPageHandler))).Methods("GET")
	r.Handle("/change-password", RequireLogin(http.HandlerFunc(ChangePasswordHandler))).Methods("POST")

	// authenticated pages
	auth := r.PathPrefix("/").Subrouter()
	auth.Use(func(next http.Handler) http.Handler { return RequireLogin(next) })
	auth.HandleFunc("/logout", LogoutHandler).Methods("GET")
	auth.HandleFunc("/profile", ProfilePageHandler).Methods("GET")
	auth.HandleFunc("/profile", ProfileUpdateFormHandler).Methods("POST")
	auth.HandleFunc("/testing", TestingRequirementsPageHandler).Methods("GET")
	auth.HandleFunc("/forms", FormsPageHandler).Methods("GET")

	// admin
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(func(next http.Handler) http.Handler { return RequireLogin(next) })
	admin.Use(func(next http.Handler) http.Handler { return RequireAdmin(next) })

	admin.HandleFunc("/add-user", AddUserPageHandler).Methods("GET")
	admin.HandleFunc("/add-user", AddUserFormHandler).Methods("POST")

	admin.HandleFunc("/manage-users", ManageUsersPageHandler).Methods("GET")
	admin.HandleFunc("/manage-users", ManageUsersFormHandler).Methods("POST")

	// RestAPI Endpoints
	admin.HandleFunc("/user", UserGetHandler).Methods("GET")
	admin.HandleFunc("/reset", ResetUserPasswordHandler).Methods("POST")
	auth.HandleFunc("/rank", RankGet).Methods("GET")
	auth.HandleFunc("/form", FormGet).Methods("GET")

	// static (for local css/js)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Println("Starting server at :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		return
	}
}

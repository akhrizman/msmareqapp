package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	initDB()
	initSessionStore()

	r := mux.NewRouter()

	// public
	r.HandleFunc("/", HomeGetHandler).Methods("GET")

	r.HandleFunc("/login", LoginPageGetHandler).Methods("GET")
	r.HandleFunc("/login", LoginHandler).Methods("POST")

	// change password (requires logged in)
	r.Handle("/change-password", RequireLogin(http.HandlerFunc(ChangePasswordGet))).Methods("GET")
	r.Handle("/change-password", RequireLogin(http.HandlerFunc(ChangePasswordPost))).Methods("POST")

	// authenticated pages
	auth := r.PathPrefix("/").Subrouter()
	auth.Use(func(next http.Handler) http.Handler { return RequireLogin(next) })

	auth.HandleFunc("/logout", LogoutGetHandler).Methods("GET")

	auth.HandleFunc("/profile", ProfileGet).Methods("GET")
	auth.HandleFunc("/profile", ProfilePost).Methods("POST")

	auth.HandleFunc("/testing", TestingGet).Methods("GET")

	auth.HandleFunc("/forms", FormsGet).Methods("GET")

	// admin
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(func(next http.Handler) http.Handler { return RequireLogin(next) })
	admin.Use(func(next http.Handler) http.Handler { return RequireAdmin(next) })

	admin.HandleFunc("/add-student", AddStudentGet).Methods("GET")
	admin.HandleFunc("/add-student", AddStudentPost).Methods("POST")

	admin.HandleFunc("/manage-students", ManageStudentsGet).Methods("GET")
	admin.HandleFunc("/manage-students", ManageStudentsPost).Methods("POST")

	// RestAPI Endpoints
	auth.HandleFunc("/rank", RankGet).Methods("GET")

	// static (if you want to serve local css/js)
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Println("Starting server at :8080")
	http.ListenAndServe(":8080", r)
}

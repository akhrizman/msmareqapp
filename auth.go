package main

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store *sessions.CookieStore

const sessionName = "msmareq-session"

func initSessionStore() {
	secret := os.Getenv("SESSION_KEY")
	if secret == "" {
		secret = "dev-secret-please-change" // change in production!
	}
	store = sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		HttpOnly: true,
		Secure:   false, // set true if HTTPS
		Path:     "/",
		MaxAge:   60 * 60 * 24, // 1 day
	}
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPasswordHash(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

// Password policy: Capital, number, special, min 8.
func ValidatePasswordPolicy(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if match, _ := regexp.MatchString(`[A-Z]`, pw); !match {
		return errors.New("password must contain at least one uppercase letter")
	}
	if match, _ := regexp.MatchString(`[0-9]`, pw); !match {
		return errors.New("password must contain at least one number")
	}
	if match, _ := regexp.MatchString(`[!@#~$%^&*()_\-+={}|\[\]\\:;\"'<>,.?/]`, pw); !match {
		return errors.New("password must contain at least one special character")
	}
	return nil
}

func LoginUser(w http.ResponseWriter, r *http.Request, username string) error {
	session, _ := store.Get(r, sessionName)
	session.Values["username"] = username
	session.Save(r, w)
	// update last_login_date
	_, err := db.Exec("UPDATE user SET last_login_date = ? WHERE username = ?", time.Now(), username)
	return err
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	delete(session.Values, "username")
	session.Options.MaxAge = -1
	session.Save(r, w)
}

func CurrentUser(r *http.Request) (*User, error) {
	session, _ := store.Get(r, sessionName)
	v := session.Values["username"]
	if v == nil {
		return nil, errors.New("not logged in")
	}
	username := v.(string)
	u, err := GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, errors.New("user inactive")
	}
	return u, nil
}

// middleware
func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := CurrentUser(r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := CurrentUser(r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if !u.IsAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// helper to parse int from nullable rank id +1 logic
func nextRankIDForUser(u *User) (int, error) {
	// if student_rank_id is null, assume next = 1
	if !u.StudentRankID.Valid {
		return 1, nil
	}
	return int(u.StudentRankID.Int64) + 1, nil
}

func parseBoolFromForm(r *http.Request, name string) bool {
	// checkbox handling
	return r.FormValue(name) == "on" || r.FormValue(name) == "1" || r.FormValue(name) == "true"
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var templateFuncs = template.FuncMap{
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid dict call: odd number of args")
		}
		m := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			m[key] = values[i+1]
		}
		return m, nil
	},
}

var templates = template.Must(
	template.New("").Funcs(templateFuncs).ParseFiles(
		"templates/navbar.gohtml",
		"templates/layout_top.gohtml",
		"templates/layout_bottom.gohtml",
		"templates/home.gohtml",
		"templates/login.gohtml",
		"templates/profile.gohtml",
		"templates/testing_requirements.gohtml",
		"templates/forms.gohtml",
		"templates/change_password.gohtml",
		"templates/add_user.gohtml",
		"templates/manage_users.gohtml",
	),
)

func render(w http.ResponseWriter, name string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("error rendering template: %v", err)
	}
}

func HomeGetHandler(w http.ResponseWriter, r *http.Request) {
	render(w, "home", map[string]interface{}{})
}

func LoginPageGetHandler(w http.ResponseWriter, r *http.Request) {
	render(w, "login", map[string]interface{}{})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Login form: %v", err)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	user, err := GetUserByUsername(username)
	if err != nil {
		render(w, "login", map[string]interface{}{"Error": "invalid credentials"})
		return
	}
	if err := CheckPasswordHash(user.PasswordHash, password); err != nil {
		render(w, "login", map[string]interface{}{"Error": "invalid credentials"})
		return
	}
	// login success
	err = LoginUser(w, r, username)
	if err != nil {
		log.Printf("error logging in: %v", err)
	}
	if user.ForcePasswordChange {
		http.Redirect(w, r, "/change-password", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/testing", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	LogoutUser(w, r)
	render(w, "login", map[string]interface{}{})
}

// ChangePasswordPageHandler Shows change password form on first login
func ChangePasswordPageHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	render(w, "change_password", map[string]interface{}{"User": u})
}

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Change Password form: %v", err)
	}
	pw := r.FormValue("password")
	pw2 := r.FormValue("password2")
	if pw != pw2 {
		render(w, "change_password", map[string]interface{}{"User": u, "Error": "passwords do not match"})
		return
	}
	if err := ValidatePasswordPolicy(pw); err != nil {
		render(w, "change_password", map[string]interface{}{"User": u, "Error": err.Error()})
		return
	}
	hash, err := HashPassword(pw)
	if err != nil {
		render(w, "change_password", map[string]interface{}{"User": u, "Error": "server error"})
		return
	}
	if err := UpdateUserPassword(u.Username, hash, false); err != nil {
		render(w, "change_password", map[string]interface{}{"User": u, "Error": "server error"})
		return
	}
	http.Redirect(w, r, "/testing", http.StatusSeeOther)
}

func ProfilePageHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	render(w, "profile", map[string]interface{}{"User": u})
}

func ProfileUpdateFormHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Profile Update form: %v", err)
	}
	fn := r.FormValue("first_name")
	ln := r.FormValue("last_name")
	if fn == "" || ln == "" {
		render(w, "profile", map[string]interface{}{"User": u, "Error": "first and last required"})
		return
	}
	u.FirstName = fn
	u.LastName = ln
	if err := UpdateUserProfile(u); err != nil {
		render(w, "profile", map[string]interface{}{"User": u, "Error": "server error"})
		return
	}
	// handle password change optional
	newpw := r.FormValue("new_password")
	if newpw != "" {
		if err := ValidatePasswordPolicy(newpw); err != nil {
			render(w, "profile", map[string]interface{}{"User": u, "Error": err.Error()})
			return
		}
		hash, _ := HashPassword(newpw)
		if err := UpdateUserPassword(u.Username, hash, false); err != nil {
			render(w, "profile", map[string]interface{}{"User": u, "Error": "server error"})
			return
		}
	}
	render(w, "profile", map[string]interface{}{"User": u, "Success": "Profile updated"})
}

// TestingRequirementsPageHandler - Loads the next rank's Testing Requirements
func TestingRequirementsPageHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	allRanks, _ := GetAllTestableRanks()
	nextID, _ := nextRankIDForUser(u)
	// Build dropdown depending on allow_full_access
	var dropdown []StudentRank
	if u.AllowFullAccess {
		dropdown = allRanks
	} else {
		for _, rr := range allRanks {
			if rr.ID <= nextID {
				dropdown = append(dropdown, rr)
			}
		}
	}
	// find selected rank (preset to rank with id == nextID if exists)
	var selected *StudentRank
	for _, rr := range dropdown {
		if rr.ID == nextID {
			tmp := rr
			selected = &tmp
		}
	}
	// content below is requirements column of selected
	reqText := ""
	if selected != nil {
		reqText = selected.Requirements
	}
	render(w, "testing_requirements", map[string]interface{}{
		"User":     u,
		"Dropdown": dropdown,
		"Selected": selected,
		"Req":      reqText,
	})
}

// FormsPageHandler - Loads the next rank's form requirement
func FormsPageHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	allRanks, _ := GetAllTestableRanks()
	nextID, _ := nextRankIDForUser(u)

	var ranksToShow []StudentRank
	if u.AllowFullAccess {
		ranksToShow = allRanks
	} else {
		for _, rr := range allRanks {
			if rr.ID <= nextID {
				ranksToShow = append(ranksToShow, rr)
			}
		}
	}

	// default select the nextID rank and load that rank's form
	var selected *StudentRank
	for _, rr := range ranksToShow {
		if rr.ID == nextID {
			tmp := rr
			selected = &tmp
		}
	}
	var form *Form
	if selected != nil && selected.FormID.Valid {
		formID := int(selected.FormID.Int64)
		form, _ = GetFormByID(formID)
	}
	render(w, "forms", map[string]interface{}{
		"User":     u,
		"Ranks":    ranksToShow,
		"Selected": selected,
		"Form":     form,
	})
}

// RankGet Rank API to retrieve requirements
func RankGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	rank, err := GetRankByID(id)
	if err == sql.ErrNoRows {
		http.Error(w, "Rank not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(rank)
	if err != nil {
		log.Printf("Error encoding rank: %v", err)
	}
}

func AddUserPageHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	ranks, _ := GetAllRanks()
	render(w, "add_user", map[string]interface{}{"User": u, "Ranks": ranks})
}

func AddUserFormHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Add User form: %v", err)
	}
	first := r.FormValue("first_name")
	last := r.FormValue("last_name")
	username := r.FormValue("username")
	rankID, _ := strconv.Atoi(r.FormValue("rank_id"))
	allow := parseBoolFromForm(r, "allow_full_access")
	if username == "" || first == "" || last == "" {
		ranks, _ := GetAllRanks()
		render(w, "add_user", map[string]interface{}{"User": u, "Ranks": ranks, "Error": "missing fields"})
		return
	}
	newUser := &User{
		Username:        username,
		FirstName:       first,
		LastName:        last,
		IsAdmin:         false,
		IsActive:        true,
		AllowFullAccess: allow,
		StudentRankID:   sqlNullInt(rankID),
	}
	// default password = FirstLastMSMA$123 and force password change
	defaultPwd := first + last + "MSMA$123"
	hashed, _ := HashPassword(defaultPwd)
	if err := CreateUser(newUser, hashed); err != nil {
		ranks, _ := GetAllRanks()
		render(w, "add_user", map[string]interface{}{"User": u, "Ranks": ranks, "Error": "could not create user: " + err.Error()})
		return
	}
	ranks, _ := GetAllRanks()
	render(w, "add_user", map[string]interface{}{"User": u, "Ranks": ranks, "Success": fmt.Sprintf("User created with default password: %s", defaultPwd)})
}

func ManageUsersPageHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	users, _ := GetAllUsersExcept(u.Username)
	ranks, _ := GetAllRanks()
	render(w, "manage_users", map[string]interface{}{"User": u, "Users": users, "Ranks": ranks})
}

func ManageUsersFormHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Manage User form: %v", err)
	}
	target := r.FormValue("selected_username")
	// load the user
	targetUser, err := GetUserByUsername(target)
	if err != nil {
		users, _ := GetAllUsersExcept(u.Username)
		ranks, _ := GetAllRanks()
		render(w, "manage_users", map[string]interface{}{"User": u, "Users": users, "Ranks": ranks, "Error": "cannot find user"})
		return
	}
	// update fields
	targetUser.FirstName = r.FormValue("first_name")
	targetUser.LastName = r.FormValue("last_name")
	targetUser.IsAdmin = parseBoolFromForm(r, "is_admin")
	targetUser.IsActive = parseBoolFromForm(r, "is_active")
	targetUser.AllowFullAccess = parseBoolFromForm(r, "allow_full_access")
	rankID, _ := strconv.Atoi(r.FormValue("rank_id"))
	targetUser.StudentRankID = sqlNullInt(rankID)
	if err := UpdateUserAdminDetails(targetUser); err != nil {
		users, _ := GetAllUsersExcept(u.Username)
		ranks, _ := GetAllRanks()
		render(w, "manage_users", map[string]interface{}{"User": u, "Users": users, "Ranks": ranks, "Error": "update failed"})
		return
	}
	if r.FormValue("reset_password") == "on" {
		_, err := ResetUserPasswordToDefault(targetUser.Username, targetUser.FirstName, targetUser.LastName)
		if err != nil {
			// ignore for now
		}
	}
	users, _ := GetAllUsersExcept(u.Username)
	ranks, _ := GetAllRanks()
	render(w, "manage_users", map[string]interface{}{"User": u, "Users": users, "Ranks": ranks, "Success": "Updated"})
}

// small helper
func sqlNullInt(v int) (ni sql.NullInt64) {
	if v <= 0 {
		ni.Valid = false
		return
	}
	ni.Valid = true
	ni.Int64 = int64(v)
	return
}

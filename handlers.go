package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Regex for username enumeration
var trailingNumberRegex = regexp.MustCompile(`^(.*?)(\d+)$`)

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
		"templates/belt.gohtml",
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
		"templates/edit_forms.gohtml",
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

	// BLOCK INACTIVE USERS (mimic invalid password)
	if !user.IsActive {
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
	render(w, "home", map[string]interface{}{})
}

// ChangePasswordPageHandler Shows change password form on first login
func ChangePasswordPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	render(w, "change_password", map[string]interface{}{"User": user})
}

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Change Password form: %v", err)
	}
	pw := r.FormValue("password")
	pw2 := r.FormValue("password2")
	if pw != pw2 {
		render(w, "change_password", map[string]interface{}{"User": user, "Error": "passwords do not match"})
		return
	}
	if err := ValidatePasswordPolicy(pw); err != nil {
		render(w, "change_password", map[string]interface{}{"User": user, "Error": err.Error()})
		return
	}
	hash, err := HashPassword(pw)
	if err != nil {
		render(w, "change_password", map[string]interface{}{"User": user, "Error": "server error"})
		return
	}
	if err := UpdateUserPassword(user.Username, hash, false); err != nil {
		render(w, "change_password", map[string]interface{}{"User": user, "Error": "server error"})
		return
	}
	http.Redirect(w, r, "/testing", http.StatusSeeOther)
}

func ProfilePageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	render(w, "profile", map[string]interface{}{"User": user})
}

func ProfileUpdateFormHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Profile Update form: %v", err)
	}
	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	if firstName == "" || lastName == "" {
		render(w, "profile", map[string]interface{}{"User": user, "Error": "first and last required"})
		return
	}
	user.FirstName = firstName
	user.LastName = lastName
	if err := UpdateUserProfile(user); err != nil {
		render(w, "profile", map[string]interface{}{"User": user, "Error": "server error"})
		return
	}
	// handle password change optional
	newpw := r.FormValue("new_password")
	if newpw != "" {
		if err := ValidatePasswordPolicy(newpw); err != nil {
			render(w, "profile", map[string]interface{}{"User": user, "Error": err.Error()})
			return
		}
		hash, _ := HashPassword(newpw)
		if err := UpdateUserPassword(user.Username, hash, false); err != nil {
			render(w, "profile", map[string]interface{}{"User": user, "Error": "server error"})
			return
		}
	}
	render(w, "profile", map[string]interface{}{"User": user, "Success": "Profile updated"})
}

// TestingRequirementsPageHandler - Loads the next rank's Testing Requirements
func TestingRequirementsPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	allRanks, _ := GetAllTestableRanks()
	nextID, _ := nextRankIDForUser(user)

	// Clamp selected rank to 17 if user's rank is greater than 17
	selectedID := nextID
	if user.StudentRankID.Valid && user.StudentRankID.Int64 > 17 {
		selectedID = 17 // highest testable rank
	}

	// Build dropdown depending on allow_full_access
	var dropdown []StudentRank
	if user.AllowFullAccess {
		dropdown = allRanks
	} else {
		for _, rr := range allRanks {
			if rr.ID <= selectedID {
				dropdown = append(dropdown, rr)
			}
		}
	}

	// find selected rank (preset to rank with id == nextID if exists) unless user's next rank is not testable
	var selected *StudentRank

	for _, rr := range allRanks {
		tmp := rr
		if rr.ID == selectedID {
			selected = &tmp
		}
	}

	// content below is requirements column of selected
	reqText := ""
	if selected != nil {
		reqText = selected.Requirements
	}

	nextRank, _ := GetRankByID(nextID)

	render(w, "testing_requirements", map[string]interface{}{
		"User":     user,
		"Dropdown": dropdown,
		"Selected": selected,
		"NextRank": nextRank, // actual next rank
		"Req":      reqText,
	})
}

// FormsPageHandler - Loads the next rank's form requirement
func FormsPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	allRanks, _ := GetAllTestableRanks()
	nextID, _ := nextRankIDForUser(user)

	// Clamp selected rank to 17 if user's rank is greater than 17
	selectedID := nextID
	if user.StudentRankID.Valid && user.StudentRankID.Int64 > 17 {
		selectedID = 17 // highest testable rank
	}

	var ranksToShow []StudentRank
	if user.AllowFullAccess {
		ranksToShow = allRanks
	} else {
		for _, rr := range allRanks {
			if rr.ID <= selectedID {
				ranksToShow = append(ranksToShow, rr)
			}
		}
	}

	// default select the nextID rank and load that rank's form
	//idParam := r.URL.Query().Get("formId")
	//if idParam == "" {
	//	http.Error(w, "Missing id parameter", http.StatusBadRequest)
	//	return
	//}
	//
	//id, err := strconv.Atoi(idParam)
	//if err != nil {
	//	http.Error(w, "Invalid id parameter", http.StatusBadRequest)
	//	return
	//}

	var selected *StudentRank
	for _, rr := range ranksToShow {
		if rr.ID == selectedID {
			tmp := rr
			selected = &tmp
		}
	}
	var form *Form
	if selected != nil && selected.FormID.Valid {
		formID := int(selected.FormID.Int64)
		form, _ = GetFormByID(formID)
	}

	nextForm, _ := GetFormByRankID(nextID)

	render(w, "forms", map[string]interface{}{
		"User":     user,
		"Ranks":    ranksToShow,
		"Selected": selected,
		"NextForm": nextForm,
		"Form":     form,
	})
}

func AddUserPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	ranks, _ := GetAllRanks()
	render(w, "add_user", map[string]interface{}{"User": user, "Ranks": ranks})
}

func AddUserFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/manage-users", http.StatusSeeOther)
		return
	}

	user, _ := CurrentUser(r)

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Add User form: %v", err)
	}

	first := strings.TrimSpace(r.FormValue("first_name"))
	last := strings.TrimSpace(r.FormValue("last_name"))

	if first == "" || last == "" {
		ranks, _ := GetAllRanks()
		render(w, "add_user", map[string]interface{}{
			"User":  user,
			"Ranks": ranks,
			"Error": "missing fields",
		})
		return
	}

	baseUsername := strings.ToLower(lettersOnly(first) + "." + lettersOnly(last))
	username := GenerateValidUsername(baseUsername)

	rankID, _ := strconv.Atoi(r.FormValue("rank_id"))
	allow := parseBoolFromForm(r, "allow_full_access")

	if username == "" {
		ranks, _ := GetAllRanks()
		render(w, "add_user", map[string]interface{}{
			"User":  user,
			"Ranks": ranks,
			"Error": "missing fields",
		})
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

	defaultPwd := strings.ToLower(lettersOnly(first+last)) + Config.DefaultPasswordSuffix
	hashed, _ := HashPassword(defaultPwd)

	if err := CreateUser(newUser, hashed); err != nil {
		ranks, _ := GetAllRanks()
		render(w, "add_user", map[string]interface{}{
			"User":  user,
			"Ranks": ranks,
			"Error": "could not create user: " + err.Error(),
		})
		return
	}

	// Redirect after successful POST
	http.Redirect(w, r, "/admin/manage-users?created_user="+url.QueryEscape(username)+
		"&created_password="+url.QueryEscape(defaultPwd), http.StatusSeeOther)
}

func GenerateValidUsername(potentialUsername string) string {
	user, _ := GetUserByUsername(potentialUsername)
	if user != nil {
		base := potentialUsername
		num := 1

		if matches := trailingNumberRegex.FindStringSubmatch(potentialUsername); matches != nil {
			base = matches[1]
			n, _ := strconv.Atoi(matches[2])
			num = n + 1
		}

		// return incremented username
		return GenerateValidUsername(fmt.Sprintf("%s%d", base, num))
	}
	return potentialUsername
}

func ManageUsersPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	users, _ := GetAllUsersExcept(user.Username)
	ranks, _ := GetAllRanks()

	data := map[string]interface{}{
		"User":  user,
		"Users": users,
		"Ranks": ranks,
	}

	createdUser := r.URL.Query().Get("created_user")
	createdPwd := r.URL.Query().Get("created_password")

	if createdUser != "" && createdPwd != "" {
		data["Success"] = fmt.Sprintf(
			"User %s created successfully.  Temporary password: %s",
			createdUser,
			createdPwd,
		)
	}

	render(w, "manage_users", data)
}

func ManageUsersFormHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing Manage User form: %v", err)
	}
	target := r.FormValue("selected_username")
	// load the user
	targetUser, err := GetUserByUsername(target)
	if err != nil {
		users, _ := GetAllUsersExcept(user.Username)
		ranks, _ := GetAllRanks()
		render(w, "manage_users", map[string]interface{}{"User": user, "Users": users, "Ranks": ranks, "Error": "cannot find user"})
		return
	}
	// update fields
	targetUser.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	targetUser.LastName = strings.TrimSpace(r.FormValue("last_name"))
	targetUser.IsAdmin = parseBoolFromForm(r, "is_admin")
	targetUser.IsActive = parseBoolFromForm(r, "is_active")
	targetUser.AllowFullAccess = parseBoolFromForm(r, "allow_full_access")
	rankID, _ := strconv.Atoi(r.FormValue("rank_id"))
	targetUser.StudentRankID = sqlNullInt(rankID)
	if err := UpdateUserAdminDetails(targetUser); err != nil {
		users, _ := GetAllUsersExcept(user.Username)
		ranks, _ := GetAllRanks()
		render(w, "manage_users", map[string]interface{}{"User": user, "Users": users, "Ranks": ranks, "Error": "update failed"})
		return
	}

	users, _ := GetAllUsersExcept(user.Username)
	ranks, _ := GetAllRanks()
	render(w, "manage_users", map[string]interface{}{"User": user, "Users": users, "Ranks": ranks, "Success": targetUser.Username + " Updated"})
}

// RestAPIs

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

// FormForRankGet Form API to retrieve form by rankId
func FormForRankGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("rankId")
	if idParam == "" {
		http.Error(w, "Missing rankId parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "Invalid rankId parameter", http.StatusBadRequest)
		return
	}

	form, err := GetFormByRankID(id)
	if err == sql.ErrNoRows {
		http.Error(w, "Form not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(form)
	if err != nil {
		log.Printf("Error encoding form: %v", err)
	}
}

// FormGet Form API to retrieve form by id
func FormGet(w http.ResponseWriter, r *http.Request) {
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

	form, err := GetFormByID(id)
	if err == sql.ErrNoRows {
		http.Error(w, "Form not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(form)
	if err != nil {
		log.Printf("Error encoding form: %v", err)
	}
}

func BeltGet(w http.ResponseWriter, r *http.Request) {
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

	rank, err := GetBeltDetailsByRankID(id)
	if err == sql.ErrNoRows {
		http.Error(w, "Belt Details not found", http.StatusNotFound)
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

// UserGetHandler UserDTO API to get the user details without password
func UserGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	user, err := GetUserByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	dto := UserDTO{
		Username:        user.Username,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		IsAdmin:         user.IsAdmin,
		IsActive:        user.IsActive,
		AllowFullAccess: user.AllowFullAccess,
	}

	if user.StudentRankID.Valid {
		dto.StudentRankID = int(user.StudentRankID.Int64)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto)
}

func ResetUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
		return
	}

	user, err := GetUserByUsername(username)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	newPwd, err := ResetUserPasswordToDefault(
		user.Username,
		user.FirstName,
		user.LastName,
	)
	if err != nil {
		http.Error(w, `{"error":"reset failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(
		`{"password":"%s"}`,
		newPwd,
	)))
}

func lettersOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
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

func EditFormsPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	forms, err := GetFormNames()

	if err != nil {
		http.Error(w, `{"error":"getting forms list failed"}`, http.StatusInternalServerError)
		return
	}

	render(w, "edit_forms", map[string]interface{}{
		"User":  user,
		"Forms": forms,
	})
}

func EditFormsFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//user, _ := CurrentUser(r)
	err := r.ParseForm()

	if err != nil {
		log.Printf("error parsing Form Update form: %v", err)
	}

	formName := strings.TrimSpace(r.FormValue("formName"))
	if formName == "" {
		render(w, "edit_forms", map[string]interface{}{"Error": "form name required"})
		return
	}

	formDescription := strings.TrimSpace(r.FormValue("formDescription"))
	if formDescription == "" {
		render(w, "edit_forms", map[string]interface{}{"Error": "form description required"})
		return
	}

	formSteps := strings.TrimSpace(r.FormValue("formSteps"))
	if formSteps == "" {
		render(w, "edit_forms", map[string]interface{}{"Error": "form steps required"})
		return
	}

	formId, _ := strconv.Atoi(r.FormValue("formId"))
	form, err := GetFormByID(formId)
	if err != nil {
		log.Printf("error getting form: %v", err)
	}

	form.Name = formName
	form.Description = formDescription
	form.Steps = formSteps

	if err := UpdateForm(form); err != nil {
		render(w, "edit_forms", map[string]interface{}{"Error": "server error"})
		return
	}

	http.Redirect(w, r, "/forms", http.StatusSeeOther)
}

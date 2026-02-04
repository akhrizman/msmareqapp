package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Form struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Steps       string `json:"steps"`
	VideoLink   string `json:"videoLink"`
}

type StudentRank struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Requirements string        `json:"requirements"`
	BeltColor    string        `json:"beltColor"`
	StripeColor  string        `json:"stripeColor"`
	StripeCount  string        `json:"stripeCount"`
	FormID       sql.NullInt64 `json:"formId"`
}

type User struct {
	Username            string
	FirstName           string
	LastName            string
	PasswordHash        string
	IsAdmin             bool
	IsActive            bool
	StudentRankID       sql.NullInt64
	AllowFullAccess     bool
	LastLoginDate       sql.NullTime
	ForcePasswordChange bool
}

func (u User) Initials() string {
	var firstInitial, lastInitial string

	if u.FirstName != "" {
		firstInitial = strings.ToUpper(string([]rune(u.FirstName)[0]))
	}

	if u.LastName != "" {
		lastInitial = strings.ToUpper(string([]rune(u.LastName)[0]))
	}

	if firstInitial == "" && lastInitial == "" {
		return ""
	}

	return firstInitial + "." + lastInitial + "."
}

type UserDTO struct {
	Username        string `json:"username"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	IsAdmin         bool   `json:"is_admin"`
	IsActive        bool   `json:"is_active"`
	AllowFullAccess bool   `json:"allow_full_access"`
	StudentRankID   int    `json:"student_rank_id"`
}

// Queries

func GetUserByUsername(username string) (*User, error) {
	u := &User{}
	row := db.QueryRow(`SELECT username, first_name, last_name, password, is_admin, is_active, student_rank_id, allow_full_access, last_login_date, force_password_change FROM user WHERE username = ?`, username)
	var studentRank sql.NullInt64
	var lastLogin sql.NullTime
	var pass string
	var isAdmin, isActive, allowFull, forcePwd byte
	err := row.Scan(&u.Username, &u.FirstName, &u.LastName, &pass, &isAdmin, &isActive, &studentRank, &allowFull, &lastLogin, &forcePwd)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = pass
	u.IsAdmin = isAdmin == 1
	u.IsActive = isActive == 1
	u.StudentRankID = studentRank
	u.AllowFullAccess = allowFull == 1
	u.LastLoginDate = lastLogin
	u.ForcePasswordChange = forcePwd == 1
	return u, nil
}

func UpdateUserPassword(username, newHash string, forceChange bool) error {
	force := 0
	if forceChange {
		force = 1
	}
	_, err := db.Exec(`UPDATE user SET password = ?, force_password_change = ? WHERE username = ?`, newHash, force, username)
	return err
}

func UpdateUserProfile(u *User) error {
	_, err := db.Exec(`UPDATE user SET first_name = ?, last_name = ? WHERE username = ?`, u.FirstName, u.LastName, u.Username)
	return err
}

func GetAllTestableRanks() ([]StudentRank, error) {
	rows, err := db.Query(`SELECT id, name, description, requirements, form_id FROM student_rank WHERE id BETWEEN 2 AND 17  ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudentRank
	for rows.Next() {
		var r StudentRank
		var formID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Requirements, &formID); err != nil {
			return nil, err
		}
		r.FormID = formID
		out = append(out, r)
	}
	return out, nil
}

func GetAllRanks() ([]StudentRank, error) {
	rows, err := db.Query(`SELECT id, name, description, requirements, form_id FROM student_rank ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudentRank
	for rows.Next() {
		var r StudentRank
		var formID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Requirements, &formID); err != nil {
			return nil, err
		}
		r.FormID = formID
		out = append(out, r)
	}
	return out, nil
}

func GetBeltDetailsByRankID(id int) (*StudentRank, error) {
	r := &StudentRank{}
	err := db.QueryRow(`SELECT id, belt_color, stripe_color, stripe_count FROM student_rank WHERE id = ?`, id).Scan(&r.ID, &r.BeltColor, &r.StripeColor, &r.StripeCount)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func GetRankByID(id int) (*StudentRank, error) {
	r := &StudentRank{}
	var formID sql.NullInt64
	err := db.QueryRow(`SELECT id, name, description, requirements, form_id FROM student_rank WHERE id = ?`, id).Scan(&r.ID, &r.Name, &r.Description, &r.Requirements, &formID)
	if err != nil {
		return nil, err
	}
	r.FormID = formID
	return r, nil
}

func GetFormNames() ([]Form, error) {
	rows, err := db.Query(`SELECT id, name, description FROM form ORDER BY id`)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var forms []Form
	for rows.Next() {
		var form Form
		rows.Scan(&form.ID, &form.Name, &form.Description)
		forms = append(forms, form)
	}

	return forms, nil
}

func GetFormByID(id int) (*Form, error) {
	f := &Form{}
	err := db.QueryRow(`SELECT id, name, description, steps, video_link FROM form WHERE id = ?`, id).Scan(&f.ID, &f.Name, &f.Description, &f.Steps, &f.VideoLink)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func GetFormByRankID(id int) (*Form, error) {
	f := &Form{}
	err := db.QueryRow(`SELECT form.id, form.name, form.description, form.steps, form.video_link from student_rank INNER JOIN form on form.id = student_rank.form_id WHERE student_rank.id = ?`, id).Scan(&f.ID, &f.Name, &f.Description, &f.Steps, &f.VideoLink)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func CreateUser(u *User, passwordHash string) error {
	isAdmin := 0
	if u.IsAdmin {
		isAdmin = 1
	}
	isActive := 1
	if !u.IsActive {
		isActive = 0
	}
	allowFull := 0
	if u.AllowFullAccess {
		allowFull = 1
	}
	var rankID interface{}
	if u.StudentRankID.Valid {
		rankID = u.StudentRankID.Int64
	} else {
		rankID = nil
	}
	_, err := db.Exec(`INSERT INTO user (username, first_name, last_name, password, is_admin, is_active, student_rank_id, allow_full_access, force_password_change) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		u.Username, u.FirstName, u.LastName, passwordHash, isAdmin, isActive, rankID, allowFull)
	return err
}

func GetAllUsersExcept(username string) ([]User, error) {
	rows, err := db.Query(`SELECT username, first_name, last_name, is_admin, is_active, student_rank_id, allow_full_access FROM user WHERE username <> ?`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var rank sql.NullInt64
		var isAdmin, isActive, allowFull byte
		if err := rows.Scan(&u.Username, &u.FirstName, &u.LastName, &isAdmin, &isActive, &rank, &allowFull); err != nil {
			return nil, err
		}
		u.IsAdmin = isAdmin == 1
		u.IsActive = isActive == 1
		u.StudentRankID = rank
		u.AllowFullAccess = allowFull == 1
		out = append(out, u)
	}
	return out, nil
}

func UpdateUserAdminDetails(u *User) error {
	isAdmin := 0
	if u.IsAdmin {
		isAdmin = 1
	}
	isActive := 0
	if u.IsActive {
		isActive = 1
	}
	allowFull := 0
	if u.AllowFullAccess {
		allowFull = 1
	}
	var rankID interface{}
	if u.StudentRankID.Valid {
		rankID = u.StudentRankID.Int64
	} else {
		rankID = nil
	}
	_, err := db.Exec(`UPDATE user SET first_name = ?, last_name = ?, is_admin = ?, is_active = ?, student_rank_id = ?, allow_full_access = ? WHERE username = ?`,
		u.FirstName, u.LastName, isAdmin, isActive, rankID, allowFull, u.Username)
	return err
}

func ResetUserPasswordToDefault(username, firstName, lastName string) (string, error) {
	if username == "" {
		return "", errors.New("username empty")
	}
	newPwd := strings.ToLower(lettersOnly(firstName+lastName)) + Config.DefaultPasswordSuffix
	hashed, err := HashPassword(newPwd)
	if err != nil {
		return "", err
	}
	if err := UpdateUserPassword(username, hashed, true); err != nil {
		return "", err
	}
	return newPwd, nil
}

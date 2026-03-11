package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

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

	// Build ranksToShowInDropdown depending on allow_full_access
	var ranksToShowInDropdown []StudentRank
	if user.AllowFullAccess {
		ranksToShowInDropdown = allRanks
	} else {
		for _, rr := range allRanks {
			if rr.ID <= selectedID {
				ranksToShowInDropdown = append(ranksToShowInDropdown, rr)
			}
		}
	}

	idParam := r.URL.Query().Get("id")
	formId, err := strconv.Atoi(idParam)
	if err != nil {
		formId = 0
	}

	// Set form if provided by url param
	var formToDisplay *Form
	if formId > 0 {
		formToDisplay, err = GetFormByID(formId)
		if err != nil {
			http.Error(w, "Form Not Found", http.StatusBadRequest)
		}
	}

	// default select the nextID rank and load that rank's form
	var selected *StudentRank
	for _, rr := range ranksToShowInDropdown {
		if rr.ID == selectedID {
			tmp := rr
			selected = &tmp
		}
	}

	// set the form to the student's next required form
	if formToDisplay == nil {
		if selected != nil && selected.FormID.Valid {
			formID := int(selected.FormID.Int64)
			formToDisplay, _ = GetFormByID(formID)
		}
	} else {
		// form to display was set by url param so adjust dropdown selection accordingly
		selected, _ = GetRankByFormID(formId) // default to showing 12th gup
	}

	nextForm, _ := GetFormByRankID(nextID)

	render(w, "forms", map[string]interface{}{
		"User":          user,
		"RanksDropdown": ranksToShowInDropdown,
		"Selected":      selected,
		"NextForm":      nextForm,
		"FormToDisplay": formToDisplay,
	})
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

	if err = UpdateForm(form); err != nil {
		render(w, "edit_forms", map[string]interface{}{"Error": "server error"})
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/forms?id=%v", formId), http.StatusSeeOther)
}

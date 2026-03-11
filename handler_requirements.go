package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

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
	rankId, err := strconv.Atoi(idParam)
	if err != nil {
		rankId = 0
	}

	// Set rank if provided by url param
	var rankToDisplay *StudentRank
	if rankId > 0 {
		rankToDisplay, err = GetRankByID(rankId)
		if err != nil {
			http.Error(w, "Student Rank Not Found", http.StatusBadRequest)
		}
	}

	// find selected rank (preset to rank with id == nextID if exists) unless user's next rank is not testable
	var selected *StudentRank
	for _, rr := range ranksToShowInDropdown {
		if rr.ID == selectedID {
			tmp := rr
			selected = &tmp
		}
	}

	// set the requirements to the student's next requirements
	if rankToDisplay == nil {
		if selected != nil {
			rankToDisplay, _ = GetRankByID(selected.ID)
		}
	} else {
		// Rank to display was set by url param so adjust dropdown selection accordingly
		selected, _ = GetRankByID(rankId) // default to showing 12th gup
	}

	// content below is requirements column of selected
	reqText := ""
	if selected != nil {
		reqText = selected.Requirements
	}

	nextRank, _ := GetRankByID(nextID)

	render(w, "requirements", map[string]interface{}{
		"User":                 user,
		"RanksDropdown":        ranksToShowInDropdown,
		"Selected":             selected,
		"NextRank":             nextRank, // actual next rank
		"RequirementToDisplay": reqText,
	})
}

func EditRequirementsPageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r)
	ranks, err := GetAllTestableRanks()

	if err != nil {
		http.Error(w, `{"error":"getting ranks list failed"}`, http.StatusInternalServerError)
		return
	}

	render(w, "edit_requirements", map[string]interface{}{
		"User":  user,
		"Ranks": ranks,
	})
}

func EditRequirementsFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()

	if err != nil {
		log.Printf("error parsing Requirements Update rank: %v", err)
	}

	rankName := strings.TrimSpace(r.FormValue("rankName"))
	if rankName == "" {
		render(w, "edit_requirements", map[string]interface{}{"Error": "rank name required"})
		return
	}

	rankDescription := strings.TrimSpace(r.FormValue("rankDescription"))
	if rankDescription == "" {
		render(w, "edit_requirements", map[string]interface{}{"Error": "rank description required"})
		return
	}

	rankRequirements := strings.TrimSpace(r.FormValue("requirements"))
	if rankRequirements == "" {
		render(w, "edit_requirements", map[string]interface{}{"Error": "rank requirements required"})
		return
	}

	rankId, _ := strconv.Atoi(r.FormValue("rankId"))
	rank, err := GetRankByID(rankId)
	if err != nil {
		log.Printf("error getting rank: %v", err)
	}

	rank.Name = rankName
	rank.Description = rankDescription
	rank.Requirements = rankRequirements

	if err = UpdateRank(rank); err != nil {
		render(w, "edit_requirements", map[string]interface{}{"Error": "server error"})
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/requirements?id=%v", rankId), http.StatusSeeOther)
}

package api

import (
	"net/http"

	"claude-agent-manager/internal/roles"
)

// RoleRoutes serves the agent role catalogue.
type RoleRoutes struct{}

func NewRoleRoutes() *RoleRoutes {
	return &RoleRoutes{}
}

// List returns all roles grouped by category (summaries only, no system prompts).
func (rr *RoleRoutes) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query != "" {
		writeJSON(w, http.StatusOK, roles.Search(query))
		return
	}
	writeJSON(w, http.StatusOK, roles.List())
}

// Get returns a single role by ID including the full system prompt.
func (rr *RoleRoutes) Get(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	role := roles.Get(id)
	if role == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Role not found"})
		return
	}
	writeJSON(w, http.StatusOK, role)
}

// Stats returns basic stats about the role library.
func (rr *RoleRoutes) Stats(w http.ResponseWriter, r *http.Request) {
	cats := roles.List()
	catCounts := make(map[string]int, len(cats))
	for _, c := range cats {
		catCounts[c.Name] = len(c.Roles)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_roles": roles.Count(),
		"categories":  catCounts,
	})
}

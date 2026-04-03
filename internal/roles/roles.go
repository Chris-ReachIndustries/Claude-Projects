package roles

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed roles.json
var rolesJSON []byte

// Role represents a pre-defined agent role with a specialized system prompt.
type Role struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Emoji          string `json:"emoji"`
	Vibe           string `json:"vibe"`
	SuggestedImage string `json:"suggested_image"`
	SystemPrompt   string `json:"system_prompt"`
	Source         string `json:"source"`
}

// RoleSummary is a lightweight version for listing (no system_prompt).
type RoleSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Emoji          string `json:"emoji"`
	Vibe           string `json:"vibe"`
	SuggestedImage string `json:"suggested_image"`
}

var (
	allRoles    []Role
	rolesByID   map[string]*Role
	categories  []CategoryGroup
	loadOnce    sync.Once
)

type CategoryGroup struct {
	Name  string        `json:"name"`
	Roles []RoleSummary `json:"roles"`
}

func load() {
	// Override suggested images for roles that benefit from specialized containers
	imageOverrides := map[string]string{
		// Data/analytics roles → claude-agent-data
		"support-analytics-reporter":       "claude-agent-data",
		"engineering-data-engineer":         "claude-agent-data",
		"engineering-database-optimizer":    "claude-agent-data",
		"specialized-model-qa":             "claude-agent-data",
		"product-feedback-synthesizer":      "claude-agent-data",
		"testing-performance-benchmarker":   "claude-agent-data",
		"testing-test-results-analyzer":     "claude-agent-data",
		// Go development → claude-agent-go
		"engineering-backend-architect":     "claude-agent-go",
		"engineering-software-architect":    "claude-agent-go",
		// Web/testing roles → claude-agent-web
		"testing-reality-checker":           "claude-agent-web",
		"testing-evidence-collector":        "claude-agent-web",
		"testing-api-tester":               "claude-agent-web",
		"testing-accessibility-auditor":     "claude-agent-web",
		"marketing-seo-specialist":          "claude-agent-web",
		// Document generation → claude-agent-printingpress
		"specialized-document-generator":    "claude-agent-printingpress",
		"support-executive-summary-generator": "claude-agent-printingpress",
	}

	loadOnce.Do(func() {
		json.Unmarshal(rolesJSON, &allRoles)
		rolesByID = make(map[string]*Role, len(allRoles))
		catMap := make(map[string][]RoleSummary)
		catOrder := []string{}

		for i := range allRoles {
			r := &allRoles[i]
			// Apply image overrides
			if img, ok := imageOverrides[r.ID]; ok {
				r.SuggestedImage = img
			}
			rolesByID[r.ID] = r

			summary := RoleSummary{
				ID:             r.ID,
				Name:           r.Name,
				Category:       r.Category,
				Description:    r.Description,
				Emoji:          r.Emoji,
				Vibe:           r.Vibe,
				SuggestedImage: r.SuggestedImage,
			}
			if _, exists := catMap[r.Category]; !exists {
				catOrder = append(catOrder, r.Category)
			}
			catMap[r.Category] = append(catMap[r.Category], summary)
		}

		for _, cat := range catOrder {
			categories = append(categories, CategoryGroup{
				Name:  cat,
				Roles: catMap[cat],
			})
		}
	})
}

// Get returns a role by ID, or nil if not found.
func Get(id string) *Role {
	load()
	return rolesByID[id]
}

// List returns all roles grouped by category.
func List() []CategoryGroup {
	load()
	return categories
}

// All returns all roles as summaries.
func All() []RoleSummary {
	load()
	summaries := make([]RoleSummary, len(allRoles))
	for i, r := range allRoles {
		summaries[i] = RoleSummary{
			ID:             r.ID,
			Name:           r.Name,
			Category:       r.Category,
			Description:    r.Description,
			Emoji:          r.Emoji,
			Vibe:           r.Vibe,
			SuggestedImage: r.SuggestedImage,
		}
	}
	return summaries
}

// Search returns roles matching a query (name or description).
func Search(query string) []RoleSummary {
	load()
	q := strings.ToLower(query)
	var results []RoleSummary
	for _, r := range allRoles {
		if strings.Contains(strings.ToLower(r.Name), q) ||
			strings.Contains(strings.ToLower(r.Description), q) ||
			strings.Contains(strings.ToLower(r.Category), q) {
			results = append(results, RoleSummary{
				ID:             r.ID,
				Name:           r.Name,
				Category:       r.Category,
				Description:    r.Description,
				Emoji:          r.Emoji,
				Vibe:           r.Vibe,
				SuggestedImage: r.SuggestedImage,
			})
		}
	}
	return results
}

// Count returns the total number of roles.
func Count() int {
	load()
	return len(allRoles)
}

// BuildCatalogue generates a compact text summary of all roles for the PM prompt.
// This is kept short to avoid bloating the PM's context.
func BuildCatalogue() string {
	load()
	var b strings.Builder
	b.WriteString("## AVAILABLE AGENT ROLES\n\n")
	b.WriteString("When spawning agents, use role_id to assign a specialized role with expert system prompts.\n")
	b.WriteString("If no role_id matches, you can still use a custom role name and prompt.\n\n")

	for _, cat := range categories {
		b.WriteString("### " + cat.Name + "\n")
		for _, r := range cat.Roles {
			b.WriteString("- **" + r.ID + "** — " + r.Name)
			if r.Description != "" {
				desc := r.Description
				if len(desc) > 120 {
					desc = desc[:120] + "..."
				}
				b.WriteString(": " + desc)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

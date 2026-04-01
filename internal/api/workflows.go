package api

import (
	"encoding/json"
	"net/http"

	"claude-agent-manager/internal/db"
)

type WorkflowRoutes struct {
	db             *db.DB
	startWorkflow  func(string) (bool, string)
	pauseWorkflow  func(string) (bool, string)
}

func NewWorkflowRoutes(d *db.DB, start func(string) (bool, string), pause func(string) (bool, string)) *WorkflowRoutes {
	return &WorkflowRoutes{db: d, startWorkflow: start, pauseWorkflow: pause}
}

func (wf *WorkflowRoutes) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, wf.db.GetAllWorkflows())
}

func (wf *WorkflowRoutes) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string                   `json:"name"`
		Steps    []map[string]interface{} `json:"steps"`
		Metadata map[string]interface{}   `json:"metadata"`
	}
	if err := readJSON(r, &body); err != nil || body.Name == "" || len(body.Steps) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and steps required"})
		return
	}

	// Normalize steps
	for i, step := range body.Steps {
		if _, ok := step["name"]; !ok {
			step["name"] = "Unnamed Step"
		}
		if _, ok := step["folder_path"]; !ok {
			step["folder_path"] = ""
		}
		if _, ok := step["prompt"]; !ok {
			step["prompt"] = ""
		}
		if _, ok := step["trigger"]; !ok {
			step["trigger"] = "on_complete"
		}
		if _, ok := step["status"]; !ok {
			step["status"] = "pending"
		}
		body.Steps[i] = step
	}

	id := generateUUID()
	stepsJSON, _ := json.Marshal(body.Steps)
	metaJSON, _ := json.Marshal(body.Metadata)
	if body.Metadata == nil {
		metaJSON = []byte("{}")
	}

	wf.db.Exec("INSERT INTO workflows (id, name, steps, metadata) VALUES (?, ?, ?, ?)", id, body.Name, string(stepsJSON), string(metaJSON))

	workflow := wf.db.GetWorkflow(id)
	writeJSON(w, http.StatusCreated, workflow)
}

func (wf *WorkflowRoutes) Get(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	workflow := wf.db.GetWorkflow(id)
	if workflow == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Workflow not found"})
		return
	}
	writeJSON(w, http.StatusOK, workflow)
}

func (wf *WorkflowRoutes) Start(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	ok, errMsg := wf.startWorkflow(id)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "workflow": wf.db.GetWorkflow(id)})
}

func (wf *WorkflowRoutes) Pause(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	ok, errMsg := wf.pauseWorkflow(id)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "workflow": wf.db.GetWorkflow(id)})
}

func (wf *WorkflowRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if wf.db.GetWorkflow(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Workflow not found"})
		return
	}
	wf.db.DeleteWorkflow(id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

package services

import (
	"encoding/json"
	"log/slog"

	"claude-agent-manager/internal/db"
)

type WorkflowStep struct {
	Name       string  `json:"name"`
	FolderPath string  `json:"folder_path"`
	Prompt     string  `json:"prompt"`
	Trigger    string  `json:"trigger"`
	Condition  *string `json:"condition"`
	AgentID    *string `json:"agent_id"`
	Status     string  `json:"status"`
}

type WorkflowEngine struct {
	db *db.DB
}

func NewWorkflowEngine(d *db.DB) *WorkflowEngine {
	return &WorkflowEngine{db: d}
}

func (we *WorkflowEngine) OnAgentStatusChange(agentID, newStatus string) {
	rows, err := we.db.Query("SELECT * FROM workflows WHERE status = 'running'")
	if err != nil {
		return
	}
	defer rows.Close()

	type wfRow struct {
		ID          string
		Steps       string
		CurrentStep int64
	}

	var workflows []wfRow
	for rows.Next() {
		wf, _ := scanWorkflow(rows)
		if wf != nil {
			workflows = append(workflows, *wf)
		}
	}

	for _, wf := range workflows {
		var steps []WorkflowStep
		if json.Unmarshal([]byte(wf.Steps), &steps) != nil {
			continue
		}

		idx := int(wf.CurrentStep)
		if idx >= len(steps) {
			continue
		}
		currentStep := &steps[idx]
		if currentStep.AgentID == nil || *currentStep.AgentID != agentID {
			continue
		}

		if newStatus == "completed" {
			if currentStep.Condition != nil && *currentStep.Condition != "" {
				agent := we.db.GetAgent(agentID)
				summary, _ := agent["latest_summary"].(string)
				if summary == "" || !contains(summary, *currentStep.Condition) {
					slog.Info("Workflow step condition not met", "workflowId", wf.ID, "step", idx)
					currentStep.Status = "failed"
					we.updateSteps(wf.ID, steps)
					we.db.Exec("UPDATE workflows SET status = 'failed' WHERE id = ?", wf.ID)
					continue
				}
			}
			currentStep.Status = "completed"
			we.updateSteps(wf.ID, steps)
			we.advanceWorkflow(wf.ID)
		} else if newStatus == "archived" && currentStep.Status == "running" {
			currentStep.Status = "failed"
			we.updateSteps(wf.ID, steps)
			we.db.Exec("UPDATE workflows SET status = 'failed', completed_at = datetime('now') WHERE id = ?", wf.ID)
			slog.Warn("Workflow failed: agent archived during step", "workflowId", wf.ID, "agentId", agentID)
		}
	}
}

func (we *WorkflowEngine) advanceWorkflow(workflowID string) {
	wf := we.db.GetWorkflow(workflowID)
	if wf == nil {
		return
	}
	status, _ := wf["status"].(string)
	if status != "running" {
		return
	}

	var steps []WorkflowStep
	stepsStr, _ := wf["steps"].(string)
	if json.Unmarshal([]byte(stepsStr), &steps) != nil {
		return
	}

	currentStep, _ := wf["current_step"].(int64)
	nextIdx := int(currentStep) + 1

	if nextIdx >= len(steps) {
		we.db.Exec("UPDATE workflows SET status = 'completed', completed_at = datetime('now'), current_step = ? WHERE id = ?", nextIdx, workflowID)
		slog.Info("Workflow completed", "workflowId", workflowID)
		return
	}

	next := &steps[nextIdx]
	next.Status = "running"
	we.db.CreateLaunchRequest("new", next.FolderPath, nil, nil)
	we.updateSteps(workflowID, steps)
	we.db.Exec("UPDATE workflows SET current_step = ? WHERE id = ?", nextIdx, workflowID)
	slog.Info("Workflow advancing", "workflowId", workflowID, "step", nextIdx, "name", next.Name)
}

func (we *WorkflowEngine) StartWorkflow(workflowID string) (bool, string) {
	wf := we.db.GetWorkflow(workflowID)
	if wf == nil {
		return false, "Workflow not found"
	}
	status, _ := wf["status"].(string)
	if status == "running" {
		return false, "Workflow already running"
	}
	if status == "completed" {
		return false, "Workflow already completed"
	}

	var steps []WorkflowStep
	stepsStr, _ := wf["steps"].(string)
	if json.Unmarshal([]byte(stepsStr), &steps) != nil {
		return false, "Invalid steps JSON"
	}
	if len(steps) == 0 {
		return false, "Workflow has no steps"
	}

	currentStep, _ := wf["current_step"].(int64)
	startIdx := int(currentStep)
	if startIdx >= len(steps) {
		return false, "All steps already processed"
	}

	steps[startIdx].Status = "running"
	we.db.CreateLaunchRequest("new", steps[startIdx].FolderPath, nil, nil)
	we.updateSteps(workflowID, steps)
	we.db.Exec("UPDATE workflows SET status = 'running', started_at = datetime('now'), current_step = ? WHERE id = ?", startIdx, workflowID)

	slog.Info("Workflow started", "workflowId", workflowID, "step", startIdx)
	return true, ""
}

func (we *WorkflowEngine) PauseWorkflow(workflowID string) (bool, string) {
	wf := we.db.GetWorkflow(workflowID)
	if wf == nil {
		return false, "Workflow not found"
	}
	status, _ := wf["status"].(string)
	if status != "running" {
		return false, "Workflow is not running"
	}
	we.db.Exec("UPDATE workflows SET status = 'paused' WHERE id = ?", workflowID)
	return true, ""
}

func (we *WorkflowEngine) updateSteps(workflowID string, steps []WorkflowStep) {
	b, _ := json.Marshal(steps)
	we.db.Exec("UPDATE workflows SET steps = ? WHERE id = ?", string(b), workflowID)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// scanWorkflow extracts workflow fields from a row scanner
func scanWorkflow(rows interface{ Scan(dest ...interface{}) error; Columns() ([]string, error) }) (*struct {
	ID          string
	Steps       string
	CurrentStep int64
}, error) {
	// We need to scan all columns but only care about id, steps, current_step
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	result := &struct {
		ID          string
		Steps       string
		CurrentStep int64
	}{}

	for i, col := range cols {
		switch col {
		case "id":
			if v, ok := values[i].(string); ok {
				result.ID = v
			}
		case "steps":
			if v, ok := values[i].(string); ok {
				result.Steps = v
			}
		case "current_step":
			if v, ok := values[i].(int64); ok {
				result.CurrentStep = v
			}
		}
	}
	return result, nil
}

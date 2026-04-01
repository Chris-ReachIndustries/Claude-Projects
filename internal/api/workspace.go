package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-agent-manager/internal/db"
)

type WorkspaceRoutes struct {
	db *db.DB
}

func NewWorkspaceRoutes(d *db.DB) *WorkspaceRoutes {
	return &WorkspaceRoutes{db: d}
}

const workspaceMount = "/workspaces"

// ListFiles lists files in a project workspace directory
func (ws *WorkspaceRoutes) ListFiles(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "."
	}

	fullPath := filepath.Clean(filepath.Join(workspaceMount, reqPath))
	if !strings.HasPrefix(fullPath, workspaceMount) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied"})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil || !info.IsDir() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Directory not found"})
		return
	}

	dirEntries, err := os.ReadDir(fullPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list files"})
		return
	}

	type fileEntry struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		IsDir   bool   `json:"is_dir"`
		Size    int64  `json:"size"`
		ModTime string `json:"mod_time"`
	}

	var entries []fileEntry
	for _, e := range dirEntries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		eInfo, _ := e.Info()
		size := int64(0)
		modTime := ""
		if eInfo != nil {
			size = eInfo.Size()
			modTime = eInfo.ModTime().UTC().Format("2006-01-02 15:04:05")
		}

		relPath, _ := filepath.Rel(workspaceMount, filepath.Join(fullPath, e.Name()))
		entries = append(entries, fileEntry{
			Name:    e.Name(),
			Path:    strings.ReplaceAll(relPath, "\\", "/"),
			IsDir:   e.IsDir(),
			Size:    size,
			ModTime: modTime,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	if entries == nil {
		entries = []fileEntry{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":  reqPath,
		"files": entries,
	})
}

// ReadFile returns the content of a workspace file
func (ws *WorkspaceRoutes) ReadFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}

	fullPath := filepath.Clean(filepath.Join(workspaceMount, filePath))
	if !strings.HasPrefix(fullPath, workspaceMount) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied"})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "File not found"})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Path is a directory"})
		return
	}

	f, err := os.Open(fullPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Cannot read file"})
		return
	}
	defer f.Close()

	// Detect content type from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".md", ".txt", ".csv", ".log":
		contentType = "text/plain; charset=utf-8"
	case ".json":
		contentType = "application/json"
	case ".html":
		contentType = "text/html"
	case ".pdf":
		contentType = "application/pdf"
	case ".py", ".js", ".ts", ".go", ".sh", ".yaml", ".yml", ".toml", ".css", ".sql":
		contentType = "text/plain; charset=utf-8"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+filepath.Base(filePath)+"\"")
	io.Copy(w, io.LimitReader(f, 10*1024*1024)) // 10MB limit
}

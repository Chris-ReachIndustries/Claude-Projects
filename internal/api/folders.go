package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var hiddenDirs = map[string]bool{
	"AppData": true, "Application Data": true, "Local Settings": true,
	"MicrosoftEdgeBackups": true, "NetHood": true, "PrintHood": true,
	"Recent": true, "SendTo": true, "Start Menu": true, "Templates": true,
	"Cookies": true, "ntuser.dat": true, "NTUSER.DAT": true, "ntuser.ini": true,
	"$Recycle.Bin": true, "node_modules": true, ".cache": true, ".npm": true,
	".nuget": true, ".vscode-server": true, "__pycache__": true, ".git": true,
	"All Users": true, "Default": true, "Default User": true, "Public": true,
}

func isHidden(name string) bool {
	if hiddenDirs[name] {
		return true
	}
	if strings.HasPrefix(name, ".") && name != ".claude" {
		return true
	}
	if strings.HasPrefix(name, "$") {
		return true
	}
	return false
}

func HandleFolders(hostHomeMount string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedPath := r.URL.Query().Get("path")

		resolved := filepath.Clean(filepath.Join(hostHomeMount, requestedPath))
		if !strings.HasPrefix(resolved, hostHomeMount) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied: path outside user home"})
			return
		}

		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Directory not found"})
			return
		}

		entries, err := os.ReadDir(resolved)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read directory"})
			return
		}

		type folderEntry struct {
			Name        string `json:"name"`
			Path        string `json:"path"`
			HasChildren bool   `json:"hasChildren"`
		}

		var folders []folderEntry
		for _, e := range entries {
			if !e.IsDir() || isHidden(e.Name()) {
				continue
			}
			fullPath := filepath.Join(resolved, e.Name())
			relPath, _ := filepath.Rel(hostHomeMount, fullPath)

			hasChildren := false
			if children, err := os.ReadDir(fullPath); err == nil {
				for _, c := range children {
					if c.IsDir() && !isHidden(c.Name()) {
						hasChildren = true
						break
					}
				}
			}

			folders = append(folders, folderEntry{
				Name:        e.Name(),
				Path:        strings.ReplaceAll(relPath, "\\", "/"),
				HasChildren: hasChildren,
			})
		}

		sort.Slice(folders, func(i, j int) bool {
			return folders[i].Name < folders[j].Name
		})

		if folders == nil {
			folders = []folderEntry{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"current": requestedPath,
			"folders": folders,
		})
	}
}

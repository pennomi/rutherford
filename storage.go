package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

var storageHostPath string
var storagePattern *regexp.Regexp

func InitStorage() {
	storageHostPath = os.Getenv("STORAGE_HOST_PATH")
	patternStr := os.Getenv("STORAGE_PATH_PATTERN")
	if patternStr == "" {
		patternStr = `pvc-[a-f0-9-]+_(?P<namespace>[^_]+)_(?P<name>.+)`
	}
	storagePattern = regexp.MustCompile(patternStr)
}

type StorageUsageEntry struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UsedBytes int64  `json:"usedBytes"`
}

func HandleStorageUsage(auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		err := auth.ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if storageHostPath == "" {
			w.Write([]byte("[]"))
			return
		}

		entries, err := os.ReadDir(storageHostPath)
		if err != nil {
			w.Write([]byte("[]"))
			return
		}

		var results []StorageUsageEntry
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name, namespace := parseStorageDir(entry.Name())
			if name == "" {
				continue
			}
			size := dirSize(filepath.Join(storageHostPath, entry.Name()))
			results = append(results, StorageUsageEntry{
				Name:      name,
				Namespace: namespace,
				UsedBytes: size,
			})
		}

		json.NewEncoder(w).Encode(results)
	}
}

func parseStorageDir(dirname string) (name string, namespace string) {
	matches := storagePattern.FindStringSubmatch(dirname)
	if matches == nil {
		return "", ""
	}

	nsIdx := storagePattern.SubexpIndex("namespace")
	nameIdx := storagePattern.SubexpIndex("name")

	if nsIdx < 0 || nameIdx < 0 || nsIdx >= len(matches) || nameIdx >= len(matches) {
		return "", ""
	}

	return matches[nameIdx], matches[nsIdx]
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

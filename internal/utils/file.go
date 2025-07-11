package utils

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func SanitizeFilename(name string) string {
	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")

	// Remove anything that's not a-zA-Z0-9._-
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	safe := reg.ReplaceAllString(name, "")

	// Prevent names like ".env" or ".htaccess"
	if strings.HasPrefix(safe, ".") {
		safe = "file" + safe
	}

	if len(safe) == 0 {
		return "file"
	}
	return safe
}

// CleanOrphanedFiles deletes files in the given uploadDir that are not present in the database.
func CleanOrphanedFiles(db *sql.DB, uploadDir string) error {
	// 1. Read all files in upload dir
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		return fmt.Errorf("could not read upload dir: %w", err)
	}

	// 2. Get all storage paths from DB
	rows, err := db.Query(`SELECT storage_path FROM files`)
	if err != nil {
		return fmt.Errorf("could not query file records: %w", err)
	}
	defer rows.Close()

	dbPaths := make(map[string]bool)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue // skip bad rows
		}
		base := filepath.Base(path) // normalize just filename
		dbPaths[base] = true
	}

	// 3. Compare disk files to DB entries
	for _, f := range files {
		if !f.Type().IsRegular() {
			continue
		}
		if _, ok := dbPaths[f.Name()]; !ok {
			fullPath := filepath.Join(uploadDir, f.Name())
			err := os.Remove(fullPath)
			if err != nil {
				fmt.Printf("Could not delete orphaned file: %s\n", fullPath)
			} else {
				fmt.Printf("Deleted orphaned file: %s\n", fullPath)
			}
		}
	}
	return nil
}

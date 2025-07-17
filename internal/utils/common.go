package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

// ValidateOwnership checks if a record with the given id exists in a known table and belongs to userID.
func ValidateOwnership(db *sql.DB, table string, id int, userID int) error {
	// Whitelist allowed table names
	switch table {
	case "contacts", "companies":
		// OK
	default:
		return errors.New("invalid table for ownership check")
	}

	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = ? AND user_id = ?)", table)

	var exists bool
	if err := db.QueryRow(query, id, userID).Scan(&exists); err != nil {
		return fmt.Errorf("error checking %s ownership: %w", table, err)
	}
	if !exists {
		return fmt.Errorf("%s not found or does not belong to the user", table)
	}
	return nil
}
func GeneratePassword(plain string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), err
}
func SplitName(full string) (string, string) {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return "", ""
	}
	first := parts[0]
	last := strings.Join(parts[1:], " ")
	return first, last
}

package tokenstore

import (
	"errors"
	"fmt"
	"github.com/tidwall/buntdb"
	"strconv"
	"time"
)

type TokenStore interface {
	SaveIDToken(userID int, token string, expiresAt time.Time) error
	GetIDToken(userID int) (string, error)
	DeleteIDToken(userID int) error
}
type BuntDBTokenStore struct {
	DB *buntdb.DB
}

// NewBuntDBTokenStore opens the buntDB database at the given path.
func NewBuntDBTokenStore(path string) (*BuntDBTokenStore, error) {
	db, err := buntdb.Open(path)
	if err != nil {
		return nil, err
	}
	return &BuntDBTokenStore{DB: db}, nil
}

func userKey(userID int) string {
	return "id_token:" + strconv.Itoa(userID)
}

// SaveIDToken stores the token with an expiry.
func (s *BuntDBTokenStore) SaveIDToken(userID int, token string, expiresAt time.Time) error {
	key := userKey(userID)
	ttl := time.Until(expiresAt).Round(time.Second)
	return s.DB.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, token, &buntdb.SetOptions{
			Expires: true,
			TTL:     ttl,
		})
		return err
	})
}

// GetIDToken retrieves the token by user ID.
func (s *BuntDBTokenStore) GetIDToken(userID int) (string, error) {
	key := userKey(userID)
	var token string
	err := s.DB.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		token = val
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("token not found: %w", err)
	}
	return token, nil
}

// DeleteIDToken removes a stored token.
func (s *BuntDBTokenStore) DeleteIDToken(userID int) error {
	key := userKey(userID)
	return s.DB.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		return nil
	})
}

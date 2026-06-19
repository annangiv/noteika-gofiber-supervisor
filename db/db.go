package db

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrNotFound = errors.New("record not found")
	ErrExpired  = errors.New("record has expired")
)

type User struct {
	ID             string `json:"id"`
	EmailHash      []byte `json:"email_hash"`
	EncryptedEmail []byte `json:"encrypted_email"`
	OAuthProvider  string `json:"oauth_provider"`
	OAuthID        string `json:"oauth_id"`
	FullName            string  `json:"full_name"`
	Tier                string  `json:"tier"`
	SearchMinSimilarity float32 `json:"search_min_similarity,omitempty"`
	CreatedAt           int64   `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type Capture struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Project   string    `json:"project"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	SourceURL string    `json:"source_url"`
	Type      string    `json:"type"` // "note", "link", "qa", "code"
	Tags      []string  `json:"tags,omitempty"`
	Embedding []float32 `json:"embedding,omitempty"`
	CreatedAt int64     `json:"created_at"`
	UpdatedAt int64     `json:"updated_at"`
	DeletedAt int64     `json:"deleted_at"`
}

type BadgerRepo struct {
	db *badger.DB
}

func NewBadgerRepo(dbPath string) (*BadgerRepo, error) {
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db at %s: %w", dbPath, err)
	}
	return &BadgerRepo{db: db}, nil
}

func (r *BadgerRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// ==========================================
// USER REPOSITORY OPERATIONS
// ==========================================

func (r *BadgerRepo) SaveUser(user User) error {
	return r.db.Update(func(txn *badger.Txn) error {
		userBytes, err := json.Marshal(user)
		if err != nil {
			return err
		}

		// Save user primary record
		userKey := []byte(fmt.Sprintf("user:id:%s", user.ID))
		if err := txn.Set(userKey, userBytes); err != nil {
			return err
		}

		// Save user secondary index (email_hash -> user_id)
		emailHashHex := hex.EncodeToString(user.EmailHash)
		indexKey := []byte(fmt.Sprintf("user:email_hash:%s", emailHashHex))
		return txn.Set(indexKey, []byte(user.ID))
	})
}

func (r *BadgerRepo) GetUser(id string) (User, error) {
	var user User
	err := r.db.View(func(txn *badger.Txn) error {
		userKey := []byte(fmt.Sprintf("user:id:%s", id))
		item, err := txn.Get(userKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &user)
		})
	})
	return user, err
}

func (r *BadgerRepo) GetUserByEmailHash(hash []byte) (User, error) {
	var user User
	err := r.db.View(func(txn *badger.Txn) error {
		emailHashHex := hex.EncodeToString(hash)
		indexKey := []byte(fmt.Sprintf("user:email_hash:%s", emailHashHex))

		item, err := txn.Get(indexKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		var userID string
		err = item.Value(func(val []byte) error {
			userID = string(val)
			return nil
		})
		if err != nil {
			return err
		}

		userKey := []byte(fmt.Sprintf("user:id:%s", userID))
		userItem, err := txn.Get(userKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		return userItem.Value(func(val []byte) error {
			return json.Unmarshal(val, &user)
		})
	})
	return user, err
}

func (r *BadgerRepo) DeleteUser(id string) error {
	return r.DeleteUserAndData(id)
}

// DeleteUserAndData removes the user record, email index, and all captures.
func (r *BadgerRepo) DeleteUserAndData(userID string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		capturePrefix := []byte(fmt.Sprintf("capture:%s:", userID))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(capturePrefix); it.ValidForPrefix(capturePrefix); it.Next() {
			if err := txn.Delete(it.Item().KeyCopy(nil)); err != nil {
				return err
			}
		}

		userKey := []byte(fmt.Sprintf("user:id:%s", userID))
		item, err := txn.Get(userKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		var user User
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &user)
		}); err != nil {
			return err
		}

		if err := txn.Delete(userKey); err != nil {
			return err
		}

		emailHashHex := hex.EncodeToString(user.EmailHash)
		indexKey := []byte(fmt.Sprintf("user:email_hash:%s", emailHashHex))
		return txn.Delete(indexKey)
	})
}

// ==========================================
// SESSION REPOSITORY OPERATIONS
// ==========================================

func (r *BadgerRepo) SaveSession(session Session) error {
	return r.db.Update(func(txn *badger.Txn) error {
		sessionBytes, err := json.Marshal(session)
		if err != nil {
			return err
		}

		sessionKey := []byte(fmt.Sprintf("session:%s", session.ID))
		// Use badger TTL if we want automatic cleanup
		ttl := time.Unix(session.ExpiresAt, 0).Sub(time.Now())
		if ttl <= 0 {
			return fmt.Errorf("session has already expired")
		}

		e := badger.NewEntry(sessionKey, sessionBytes).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (r *BadgerRepo) GetSession(id string) (Session, error) {
	var session Session
	err := r.db.View(func(txn *badger.Txn) error {
		sessionKey := []byte(fmt.Sprintf("session:%s", id))
		item, err := txn.Get(sessionKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err == nil && time.Now().Unix() > session.ExpiresAt {
		return session, ErrExpired
	}

	return session, err
}

func (r *BadgerRepo) DeleteSession(id string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		sessionKey := []byte(fmt.Sprintf("session:%s", id))
		err := txn.Delete(sessionKey)
		if err == badger.ErrKeyNotFound {
			return nil // Idempotent delete
		}
		return err
	})
}

// ==========================================
// OAUTH STATE REPOSITORY OPERATIONS
// ==========================================

func (r *BadgerRepo) SaveOauthState(state string, provider string, expiresAt int64) error {
	return r.db.Update(func(txn *badger.Txn) error {
		stateKey := []byte(fmt.Sprintf("oauth_state:%s", state))
		value := fmt.Sprintf("%s|%d", provider, expiresAt)

		ttl := time.Unix(expiresAt, 0).Sub(time.Now())
		if ttl <= 0 {
			return fmt.Errorf("oauth state expiry has already passed")
		}

		e := badger.NewEntry(stateKey, []byte(value)).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (r *BadgerRepo) TakeOauthState(state string) (string, error) {
	var provider string
	err := r.db.Update(func(txn *badger.Txn) error {
		stateKey := []byte(fmt.Sprintf("oauth_state:%s", state))
		item, err := txn.Get(stateKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		var valueStr string
		err = item.Value(func(val []byte) error {
			valueStr = string(val)
			return nil
		})
		if err != nil {
			return err
		}

		// Clean up/consume state token atomically
		if err := txn.Delete(stateKey); err != nil {
			return err
		}

		parts := strings.Split(valueStr, "|")
		if len(parts) != 2 {
			return fmt.Errorf("invalid oauth state data")
		}

		var expiresAt int64
		_, err = fmt.Sscanf(parts[1], "%d", &expiresAt)
		if err != nil {
			return err
		}

		if time.Now().Unix() > expiresAt {
			return ErrExpired
		}

		provider = parts[0]
		return nil
	})
	return provider, err
}

// ==========================================
// CAPTURES REPOSITORY OPERATIONS
// ==========================================

func (r *BadgerRepo) SaveCapture(capture Capture) error {
	return r.db.Update(func(txn *badger.Txn) error {
		captureBytes, err := json.Marshal(capture)
		if err != nil {
			return err
		}

		captureKey := []byte(fmt.Sprintf("capture:%s:%s", capture.UserID, capture.ID))
		return txn.Set(captureKey, captureBytes)
	})
}

func (r *BadgerRepo) GetCapture(userID string, id string) (Capture, error) {
	var capture Capture
	err := r.db.View(func(txn *badger.Txn) error {
		captureKey := []byte(fmt.Sprintf("capture:%s:%s", userID, id))
		item, err := txn.Get(captureKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &capture)
		})
	})
	return capture, err
}

func (r *BadgerRepo) ListCaptures(userID string, projectFilter string) ([]Capture, error) {
	var captures []Capture
	err := r.db.View(func(txn *badger.Txn) error {
		prefix := []byte(fmt.Sprintf("capture:%s:", userID))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var capture Capture
				if err := json.Unmarshal(val, &capture); err != nil {
					return err
				}
				
				// Soft-delete filtering:
				if projectFilter == "Trash" {
					if capture.DeletedAt == 0 {
						return nil
					}
				} else {
					if capture.DeletedAt > 0 {
						return nil
					}
					// Filter by project if parameter is specified
					if projectFilter != "" && !strings.EqualFold(capture.Project, projectFilter) {
						return nil
					}
				}

				captures = append(captures, capture)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return captures, err
}

func (r *BadgerRepo) DeleteCapture(userID string, id string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		captureKey := []byte(fmt.Sprintf("capture:%s:%s", userID, id))
		err := txn.Delete(captureKey)
		if err == badger.ErrKeyNotFound {
			return ErrNotFound
		}
		return err
	})
}

func (r *BadgerRepo) ListProjects(userID string) ([]string, error) {
	projectsMap := make(map[string]bool)
	projectsMap["Inbox"] = true // Always ensure Inbox exists

	err := r.db.View(func(txn *badger.Txn) error {
		prefix := []byte(fmt.Sprintf("capture:%s:", userID))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var capture Capture
				if err := json.Unmarshal(val, &capture); err != nil {
					return err
				}
				if capture.DeletedAt == 0 && capture.Project != "" {
					projectsMap[capture.Project] = true
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var projects []string
	for p := range projectsMap {
		projects = append(projects, p)
	}
	sort.Strings(projects)
	return projects, nil
}

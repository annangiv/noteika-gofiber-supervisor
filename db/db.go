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
	StripeCustomerID    string  `json:"stripe_customer_id,omitempty"`
	SearchMinSimilarity float32 `json:"search_min_similarity,omitempty"`
	EncryptionSalt      []byte  `json:"encryption_salt,omitempty"`
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
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
	Ciphertext []byte `json:"ciphertext,omitempty"`
	// Legacy plaintext fields (pre-E2E captures only).
	Title     string `json:"title,omitempty"`
	Body      string `json:"body,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Type      string `json:"type"` // "note", "link", "qa", "code"
	Fingerprint []byte  `json:"fingerprint,omitempty"`
	// EncryptedVector is the real embedding, AES-GCM-encrypted client-side with
	// the vault key (never a server-held key). Sent back only for search
	// candidates so the client can decrypt instead of re-running the embedding
	// model on every result.
	EncryptedVector []byte `json:"encrypted_vector,omitempty"`
	CreatedAt int64     `json:"created_at"`
	UpdatedAt int64     `json:"updated_at"`
	DeletedAt int64     `json:"deleted_at"`
}

func (c Capture) IsEncrypted() bool {
	return len(c.Ciphertext) > 0
}

// Project holds an encrypted project display name; ProjectID on Capture references it.
// The literal sentinel "inbox" is the default project and is never stored here.
type Project struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Ciphertext []byte `json:"ciphertext"`
	CreatedAt  int64  `json:"created_at"`
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
		if err := txn.Set(indexKey, []byte(user.ID)); err != nil {
			return err
		}

		if user.StripeCustomerID != "" {
			stripeKey := []byte(fmt.Sprintf("user:stripe_customer:%s", user.StripeCustomerID))
			if err := txn.Set(stripeKey, []byte(user.ID)); err != nil {
				return err
			}
		}
		return nil
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

func (r *BadgerRepo) GetUserByStripeCustomerID(customerID string) (User, error) {
	var user User
	err := r.db.View(func(txn *badger.Txn) error {
		indexKey := []byte(fmt.Sprintf("user:stripe_customer:%s", customerID))
		item, err := txn.Get(indexKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		var userID string
		if err := item.Value(func(val []byte) error {
			userID = string(val)
			return nil
		}); err != nil {
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

func (r *BadgerRepo) CountActiveCaptures(userID string) (int, error) {
	count := 0
	err := r.db.View(func(txn *badger.Txn) error {
		prefix := []byte(fmt.Sprintf("capture:%s:", userID))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var capture Capture
				if err := json.Unmarshal(val, &capture); err != nil {
					return err
				}
				if capture.DeletedAt == 0 {
					count++
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return count, err
}

func (r *BadgerRepo) DeleteUser(id string) error {
	return r.DeleteUserAndData(id)
}

// DeleteUserAndData removes the user record, email index, and all captures/projects.
func (r *BadgerRepo) DeleteUserAndData(userID string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		for _, prefix := range [][]byte{
			[]byte(fmt.Sprintf("capture:%s:", userID)),
			[]byte(fmt.Sprintf("project:%s:", userID)),
		} {
			it := txn.NewIterator(opts)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				if err := txn.Delete(it.Item().KeyCopy(nil)); err != nil {
					it.Close()
					return err
				}
			}
			it.Close()
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
		if err := txn.Delete(indexKey); err != nil {
			return err
		}
		if user.StripeCustomerID != "" {
			stripeKey := []byte(fmt.Sprintf("user:stripe_customer:%s", user.StripeCustomerID))
			if err := txn.Delete(stripeKey); err != nil && err != badger.ErrKeyNotFound {
				return err
			}
		}
		return nil
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
					if projectFilter != "" && !strings.EqualFold(capture.ProjectID, projectFilter) {
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

func (r *BadgerRepo) SaveProject(project Project) error {
	return r.db.Update(func(txn *badger.Txn) error {
		projectBytes, err := json.Marshal(project)
		if err != nil {
			return err
		}

		projectKey := []byte(fmt.Sprintf("project:%s:%s", project.UserID, project.ID))
		return txn.Set(projectKey, projectBytes)
	})
}

// ListProjects scans the small per-user projects collection (one row per project,
// not per capture). The "inbox" sentinel default is never stored here.
func (r *BadgerRepo) ListProjects(userID string) ([]Project, error) {
	var projects []Project
	err := r.db.View(func(txn *badger.Txn) error {
		prefix := []byte(fmt.Sprintf("project:%s:", userID))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var project Project
				if err := json.Unmarshal(val, &project); err != nil {
					return err
				}
				projects = append(projects, project)
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

	sort.Slice(projects, func(i, j int) bool { return projects[i].CreatedAt < projects[j].CreatedAt })
	return projects, nil
}

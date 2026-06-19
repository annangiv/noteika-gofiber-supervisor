package db

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	FullName       string `json:"full_name"`
	Tier           string `json:"tier"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type Note struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
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
	return r.db.Update(func(txn *badger.Txn) error {
		// Fetch user to delete secondary index as well
		userKey := []byte(fmt.Sprintf("user:id:%s", id))
		item, err := txn.Get(userKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		var user User
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &user)
		})
		if err != nil {
			return err
		}

		// Delete primary record
		if err := txn.Delete(userKey); err != nil {
			return err
		}

		// Delete secondary index
		emailHashHex := hex.EncodeToString(user.EmailHash)
		indexKey := []byte(fmt.Sprintf("user:email_hash:%s", emailHashHex))
		if err := txn.Delete(indexKey); err != nil {
			return err
		}

		// Optional: clean up sessions for user
		// In badger, we would prefix scan session keys and delete those that match user.ID.
		// For simplicity, we keep them or let them expire naturally.
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
// NOTES REPOSITORY OPERATIONS
// ==========================================

func (r *BadgerRepo) SaveNote(note Note) error {
	return r.db.Update(func(txn *badger.Txn) error {
		noteBytes, err := json.Marshal(note)
		if err != nil {
			return err
		}

		noteKey := []byte(fmt.Sprintf("note:%s:%s", note.UserID, note.ID))
		return txn.Set(noteKey, noteBytes)
	})
}

func (r *BadgerRepo) GetNote(userID string, id string) (Note, error) {
	var note Note
	err := r.db.View(func(txn *badger.Txn) error {
		noteKey := []byte(fmt.Sprintf("note:%s:%s", userID, id))
		item, err := txn.Get(noteKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &note)
		})
	})
	return note, err
}

func (r *BadgerRepo) ListNotes(userID string) ([]Note, error) {
	var notes []Note
	err := r.db.View(func(txn *badger.Txn) error {
		prefix := []byte(fmt.Sprintf("note:%s:", userID))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var note Note
				if err := json.Unmarshal(val, &note); err != nil {
					return err
				}
				notes = append(notes, note)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return notes, err
}

func (r *BadgerRepo) DeleteNote(userID string, noteID string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		noteKey := []byte(fmt.Sprintf("note:%s:%s", userID, noteID))
		err := txn.Delete(noteKey)
		if err == badger.ErrKeyNotFound {
			return ErrNotFound
		}
		return err
	})
}

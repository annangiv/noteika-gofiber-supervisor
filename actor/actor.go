package actor

import "my-app/db"

// Message represents the standard container passed between web routes and actors.
type Message struct {
	Type         string
	Payload      interface{}
	ResponseChan chan Response
}

// Response is the synchronous answer sent back by the actor.
type Response struct {
	Data interface{}
	Err  error
}

// Actor represents a supervised processing unit.
type Actor interface {
	// Run runs the message-processing loop of the actor.
	// It blocks until the context is cancelled or the actor stops/panics.
	Run(mailbox chan Message) error
}

// ==========================================
// VAULT ACTOR MESSAGES
// ==========================================

const (
	TypeUpsertUser          = "UpsertUser"
	TypeGetUser             = "GetUser"
	TypeGetUserByEmailHash  = "GetUserByEmailHash"
	TypeDeleteUser          = "DeleteUser"
	TypeSaveSession         = "SaveSession"
	TypeGetSession          = "GetSession"
	TypeDeleteSession       = "DeleteSession"
	TypeSaveOauthState      = "SaveOauthState"
	TypeTakeOauthState      = "TakeOauthState"
	TypeSaveNote            = "SaveNote"
	TypeGetNote             = "GetNote"
	TypeListNotes           = "ListNotes"
	TypeDeleteNote          = "DeleteNote"
	TypeDebugCrash          = "DebugCrash"
)

type UpsertUserPayload struct {
	User db.User
}

type GetUserPayload struct {
	ID string
}

type GetUserByEmailHashPayload struct {
	Hash []byte
}

type DeleteUserPayload struct {
	ID string
}

type SaveSessionPayload struct {
	Session db.Session
}

type GetSessionPayload struct {
	ID string
}

type DeleteSessionPayload struct {
	ID string
}

type SaveOauthStatePayload struct {
	State     string
	Provider  string
	ExpiresAt int64
}

type TakeOauthStatePayload struct {
	State string
}

type SaveNotePayload struct {
	Note db.Note
}

type GetNotePayload struct {
	UserID string
	ID     string
}

type ListNotesPayload struct {
	UserID string
}

type DeleteNotePayload struct {
	UserID string
	ID     string
}

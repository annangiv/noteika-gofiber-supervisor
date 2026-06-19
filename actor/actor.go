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
	TypeGetUserByStripeCustomer = "GetUserByStripeCustomer"
	TypeCountActiveCaptures = "CountActiveCaptures"
	TypeDeleteUser          = "DeleteUser"
	TypeSaveSession         = "SaveSession"
	TypeGetSession          = "GetSession"
	TypeDeleteSession       = "DeleteSession"
	TypeSaveOauthState      = "SaveOauthState"
	TypeTakeOauthState      = "TakeOauthState"
	TypeSaveCapture         = "SaveCapture"
	TypeGetCapture          = "GetCapture"
	TypeListCaptures        = "ListCaptures"
	TypeDeleteCapture       = "DeleteCapture"
	TypeListProjects        = "ListProjects"
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

type GetUserByStripeCustomerPayload struct {
	CustomerID string
}

type CountActiveCapturesPayload struct {
	UserID string
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

type SaveCapturePayload struct {
	Capture db.Capture
}

type GetCapturePayload struct {
	UserID string
	ID     string
}

type ListCapturesPayload struct {
	UserID        string
	ProjectFilter string
}

type DeleteCapturePayload struct {
	UserID string
	ID     string
}

type ListProjectsPayload struct {
	UserID string
}

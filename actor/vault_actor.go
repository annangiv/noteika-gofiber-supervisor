package actor

import (
	"fmt"
	"log"
	"my-app/db"
)

// VaultActor processes all database operations in an isolated event loop.
type VaultActor struct {
	repo *db.BadgerRepo
}

func NewVaultActor(repo *db.BadgerRepo) *VaultActor {
	return &VaultActor{repo: repo}
}

// Run starts the event loop, pulling messages from the mailbox.
func (a *VaultActor) Run(mailbox chan Message) error {
	log.Println("[VaultActor] Event loop started")
	for msg := range mailbox {
		a.handleMessage(msg)
	}
	log.Println("[VaultActor] Mailbox channel closed, event loop exiting")
	return nil
}

func (a *VaultActor) handleMessage(msg Message) {
	var resp Response

	// Panic handler to notify client and propagate the crash to the supervisor
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[VaultActor] CRITICAL: Panicked during message execution: %v", r)
			resp.Err = fmt.Errorf("database actor crashed: %v", r)
			if msg.ResponseChan != nil {
				select {
				case msg.ResponseChan <- resp:
				default:
				}
			}
			// Re-panic so that the supervisor's monitor catches it and restarts this actor
			panic(r)
		}
	}()

	switch msg.Type {
	case TypeDebugCrash:
		log.Println("[VaultActor] Simulated crash triggered by user request!")
		panic("database vault actor crash")

	case TypeUpsertUser:
		payload := msg.Payload.(UpsertUserPayload)
		err := a.repo.SaveUser(payload.User)
		resp.Err = err

	case TypeGetUser:
		payload := msg.Payload.(GetUserPayload)
		user, err := a.repo.GetUser(payload.ID)
		resp.Data = user
		resp.Err = err

	case TypeGetUserByEmailHash:
		payload := msg.Payload.(GetUserByEmailHashPayload)
		user, err := a.repo.GetUserByEmailHash(payload.Hash)
		resp.Data = user
		resp.Err = err

	case TypeDeleteUser:
		payload := msg.Payload.(DeleteUserPayload)
		err := a.repo.DeleteUser(payload.ID)
		resp.Err = err

	case TypeSaveSession:
		payload := msg.Payload.(SaveSessionPayload)
		err := a.repo.SaveSession(payload.Session)
		resp.Err = err

	case TypeGetSession:
		payload := msg.Payload.(GetSessionPayload)
		session, err := a.repo.GetSession(payload.ID)
		resp.Data = session
		resp.Err = err

	case TypeDeleteSession:
		payload := msg.Payload.(DeleteSessionPayload)
		err := a.repo.DeleteSession(payload.ID)
		resp.Err = err

	case TypeSaveOauthState:
		payload := msg.Payload.(SaveOauthStatePayload)
		err := a.repo.SaveOauthState(payload.State, payload.Provider, payload.ExpiresAt)
		resp.Err = err

	case TypeTakeOauthState:
		payload := msg.Payload.(TakeOauthStatePayload)
		provider, err := a.repo.TakeOauthState(payload.State)
		resp.Data = provider
		resp.Err = err

	case TypeSaveNote:
		payload := msg.Payload.(SaveNotePayload)
		err := a.repo.SaveNote(payload.Note)
		resp.Err = err

	case TypeGetNote:
		payload := msg.Payload.(GetNotePayload)
		note, err := a.repo.GetNote(payload.UserID, payload.ID)
		resp.Data = note
		resp.Err = err

	case TypeListNotes:
		payload := msg.Payload.(ListNotesPayload)
		notes, err := a.repo.ListNotes(payload.UserID)
		resp.Data = notes
		resp.Err = err

	case TypeDeleteNote:
		payload := msg.Payload.(DeleteNotePayload)
		err := a.repo.DeleteNote(payload.UserID, payload.ID)
		resp.Err = err

	default:
		resp.Err = fmt.Errorf("unknown message type: %s", msg.Type)
	}

	if msg.ResponseChan != nil {
		select {
		case msg.ResponseChan <- resp:
		default:
		}
	}
}

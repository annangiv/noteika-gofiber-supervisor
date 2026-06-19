package web

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v81"
	billingportal "github.com/stripe/stripe-go/v81/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/webhook"

	"my-app/actor"
	"my-app/db"
	"my-app/utils"
)

type BillingHandler struct {
	gateway *actor.ActorGateway
}

func NewBillingHandler(gateway *actor.ActorGateway) *BillingHandler {
	return &BillingHandler{gateway: gateway}
}

func (h *BillingHandler) loadUser(userID string) (db.User, error) {
	res, err := h.gateway.Send(actor.TypeGetUser, actor.GetUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		return db.User{}, err
	}
	user, ok := res.(db.User)
	if !ok {
		return db.User{}, db.ErrNotFound
	}
	return user, nil
}

func (h *BillingHandler) userEmail(user db.User) string {
	email, err := utils.DecryptEmail(user.EncryptedEmail)
	if err != nil {
		return ""
	}
	return email
}

func (h *BillingHandler) userHasProAccess(user db.User) bool {
	if db.IsProTier(user.Tier) {
		return true
	}
	return isOwnerEmail(h.userEmail(user))
}

func (h *BillingHandler) activeCaptureCount(userID string) (int, error) {
	res, err := h.gateway.Send(actor.TypeCountActiveCaptures, actor.CountActiveCapturesPayload{UserID: userID}, 5*time.Second)
	if err != nil {
		return 0, err
	}
	count, ok := res.(int)
	if !ok {
		return 0, fmt.Errorf("invalid capture count response")
	}
	return count, nil
}

func (h *BillingHandler) saveUser(user db.User) error {
	user.UpdatedAt = time.Now().Unix()
	_, err := h.gateway.Send(actor.TypeUpsertUser, actor.UpsertUserPayload{User: user}, 5*time.Second)
	return err
}

func (h *BillingHandler) ensureStripeCustomer(user db.User, email string) (db.User, error) {
	if user.StripeCustomerID != "" {
		return user, nil
	}
	if !StripeEnabled() {
		return user, fmt.Errorf("billing is not configured")
	}

	stripe.Key = stripeSecretKey()
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(user.FullName),
		Metadata: map[string]string{
			"noteika_user_id": user.ID,
		},
	}
	cust, err := customer.New(params)
	if err != nil {
		return user, err
	}

	user.StripeCustomerID = cust.ID
	if err := h.saveUser(user); err != nil {
		return user, err
	}
	return user, nil
}

// CreateCheckout starts a Stripe Checkout subscription session.
func (h *BillingHandler) CreateCheckout(c *fiber.Ctx) error {
	if !StripeEnabled() {
		return c.Status(503).JSON(fiber.Map{"error": "billing is not configured on this server"})
	}

	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	user, err := h.loadUser(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user"})
	}
	if h.userHasProAccess(user) {
		return c.Status(400).JSON(fiber.Map{"error": "you already have Pro access"})
	}

	email := h.userEmail(user)
	if email == "" {
		return c.Status(500).JSON(fiber.Map{"error": "could not resolve account email"})
	}

	user, err = h.ensureStripeCustomer(user, email)
	if err != nil {
		log.Printf("[Billing] ensureStripeCustomer: %v", err)
		return c.Status(502).JSON(fiber.Map{"error": "failed to create billing customer"})
	}

	baseURL := appPublicURL(c.BaseURL())
	stripe.Key = stripeSecretKey()

	params := &stripe.CheckoutSessionParams{
		Customer:          stripe.String(user.StripeCustomerID),
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ClientReferenceID: stripe.String(user.ID),
		SuccessURL:        stripe.String(baseURL + "/account?billing=success"),
		CancelURL:         stripe.String(baseURL + "/account?billing=cancel"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(stripePriceID()),
				Quantity: stripe.Int64(1),
			},
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		log.Printf("[Billing] checkout session: %v", err)
		return c.Status(502).JSON(fiber.Map{"error": "failed to start checkout"})
	}

	return c.JSON(fiber.Map{"url": sess.URL})
}

// CreatePortal opens the Stripe Customer Portal for subscription management.
func (h *BillingHandler) CreatePortal(c *fiber.Ctx) error {
	if !StripeEnabled() {
		return c.Status(503).JSON(fiber.Map{"error": "billing is not configured on this server"})
	}

	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	user, err := h.loadUser(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user"})
	}
	if user.StripeCustomerID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "no billing account yet — upgrade first"})
	}

	baseURL := appPublicURL(c.BaseURL())
	stripe.Key = stripeSecretKey()

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(user.StripeCustomerID),
		ReturnURL: stripe.String(baseURL + "/account"),
	}
	sess, err := billingportal.New(params)
	if err != nil {
		log.Printf("[Billing] portal session: %v", err)
		return c.Status(502).JSON(fiber.Map{"error": "failed to open billing portal"})
	}

	return c.JSON(fiber.Map{"url": sess.URL})
}

// Webhook handles Stripe subscription lifecycle events.
func (h *BillingHandler) Webhook(c *fiber.Ctx) error {
	secret := stripeWebhookSecret()
	if secret == "" {
		return c.Status(503).SendString("webhook not configured")
	}

	payload := c.Body()
	sig := c.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, secret)
	if err != nil {
		log.Printf("[Billing] webhook signature: %v", err)
		return c.Status(400).SendString("invalid signature")
	}

	switch event.Type {
	case "checkout.session.completed":
		err = h.onCheckoutCompleted(event.Data.Raw)
	case "customer.subscription.updated", "customer.subscription.deleted":
		err = h.onSubscriptionChange(event.Data.Raw)
	default:
		// ignore unhandled events
	}

	if err != nil {
		log.Printf("[Billing] webhook %s: %v", event.Type, err)
		return c.Status(500).SendString("webhook handler failed")
	}

	return c.SendString("ok")
}

func (h *BillingHandler) onCheckoutCompleted(raw json.RawMessage) error {
	var payload struct {
		ClientReferenceID string            `json:"client_reference_id"`
		Customer          string            `json:"customer"`
		Metadata          map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}

	userID := payload.ClientReferenceID
	if userID == "" && payload.Metadata != nil {
		userID = payload.Metadata["noteika_user_id"]
	}
	if userID == "" && payload.Customer != "" {
		res, err := h.gateway.Send(actor.TypeGetUserByStripeCustomer, actor.GetUserByStripeCustomerPayload{
			CustomerID: payload.Customer,
		}, 5*time.Second)
		if err == nil {
			if u, ok := res.(db.User); ok {
				userID = u.ID
			}
		}
	}
	if userID == "" {
		return fmt.Errorf("checkout session missing user reference")
	}

	user, err := h.loadUser(userID)
	if err != nil {
		return err
	}

	if payload.Customer != "" {
		user.StripeCustomerID = payload.Customer
	}
	user.Tier = db.TierPro
	return h.saveUser(user)
}

func (h *BillingHandler) onSubscriptionChange(raw json.RawMessage) error {
	var payload struct {
		Customer string `json:"customer"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.Customer == "" {
		return fmt.Errorf("subscription missing customer")
	}

	res, err := h.gateway.Send(actor.TypeGetUserByStripeCustomer, actor.GetUserByStripeCustomerPayload{
		CustomerID: payload.Customer,
	}, 5*time.Second)
	if err != nil {
		return err
	}
	user, ok := res.(db.User)
	if !ok {
		return db.ErrNotFound
	}

	switch payload.Status {
	case string(stripe.SubscriptionStatusActive), string(stripe.SubscriptionStatusTrialing):
		user.Tier = db.TierPro
	default:
		user.Tier = db.TierFree
	}
	return h.saveUser(user)
}
func (h *BillingHandler) BillingStatus(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	user, err := h.loadUser(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user"})
	}

	count, err := h.activeCaptureCount(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to count captures"})
	}

	pro := h.userHasProAccess(user)
	limit := db.FreeCaptureLimit()

	return c.JSON(fiber.Map{
		"tier":                 user.Tier,
		"pro_access":           pro,
		"capture_count":        count,
		"capture_limit":        limit,
		"stripe_enabled":       StripeEnabled(),
		"has_stripe_customer":  user.StripeCustomerID != "",
	})
}

// EnforceCaptureLimit returns an error response if a free user exceeded their quota.
func (h *BillingHandler) EnforceCaptureLimit(c *fiber.Ctx, user db.User) error {
	if h.userHasProAccess(user) {
		return nil
	}

	count, err := h.activeCaptureCount(user.ID)
	if err != nil {
		log.Printf("[Billing] count captures: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to check plan limits"})
	}

	limit := db.FreeCaptureLimit()
	if count < limit {
		return nil
	}

	return c.Status(402).JSON(fiber.Map{
		"error":         "free capture limit reached",
		"capture_count": count,
		"capture_limit": limit,
		"upgrade_url":   "/account",
	})
}

// BillingStatus returns plan usage for the account UI.

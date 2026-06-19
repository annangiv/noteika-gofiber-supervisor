package web

import (
	"os"
	"strings"
)

func stripeSecretKey() string {
	return strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
}

func stripeWebhookSecret() string {
	return strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
}

func stripePriceID() string {
	return strings.TrimSpace(os.Getenv("STRIPE_PRICE_ID"))
}

func appPublicURL(fallback string) string {
	if u := strings.TrimSpace(os.Getenv("APP_PUBLIC_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return strings.TrimRight(fallback, "/")
}

func ownerEmail() string {
	return strings.TrimSpace(os.Getenv("NOTEIKA_OWNER_EMAIL"))
}

func StripeEnabled() bool {
	return stripeSecretKey() != "" && stripePriceID() != ""
}

func isOwnerEmail(email string) bool {
	owner := ownerEmail()
	if owner == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(email), owner)
}

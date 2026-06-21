package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2/google"

	"my-app/actor"
	"my-app/db"
)

type VerifyIAPRequest struct {
	Platform      string `json:"platform"`       // "android" or "ios"
	PurchaseToken string `json:"purchase_token"` // Google Play purchase token or Apple App Store base64 receipt
	ProductID     string `json:"product_id"`     // e.g. "noteika_pro_monthly"
}

// VerifyIAP verifies a native in-app purchase and upgrades the user's tier.
func (h *BillingHandler) VerifyIAP(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req VerifyIAPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.PurchaseToken == "" || req.ProductID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "missing purchase_token or product_id"})
	}

	var active bool
	var err error

	switch req.Platform {
	case "android":
		active, err = h.verifyAndroidPurchase(req.ProductID, req.PurchaseToken)
	case "ios":
		active, err = h.verifyIOSPurchase(req.PurchaseToken)
	default:
		return c.Status(400).JSON(fiber.Map{"error": "unsupported platform: " + req.Platform})
	}

	if err != nil {
		log.Printf("[IAP] Verification failed for user %s: %v", userID, err)
		return c.Status(502).JSON(fiber.Map{"error": "iap verification failed: " + err.Error()})
	}

	if !active {
		return c.Status(400).JSON(fiber.Map{"error": "subscription is inactive or expired"})
	}

	// Upgrade the user in database
	res, err := h.gateway.Send(actor.TypeGetUser, actor.GetUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user"})
	}
	user, ok := res.(db.User)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	user.Tier = db.TierPro
	user.UpdatedAt = time.Now().Unix()

	if _, err := h.gateway.Send(actor.TypeUpsertUser, actor.UpsertUserPayload{User: user}, 5*time.Second); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update user plan"})
	}

	captureCount, _ := h.activeCaptureCount(userID)

	return c.JSON(fiber.Map{
		"success": true,
		"user": fiber.Map{
			"id":             user.ID,
			"email":          h.userEmail(user),
			"oauth_provider": user.OAuthProvider,
			"full_name":      user.FullName,
			"pro_access":     true,
			"capture_count":  captureCount,
			"capture_limit":  db.FreeCaptureLimit(),
		},
	})
}

func (h *BillingHandler) verifyAndroidPurchase(subscriptionID string, token string) (bool, error) {
	// Look for Service Account credentials in environment variables
	saJSON := os.Getenv("GOOGLE_PLAY_SERVICE_ACCOUNT_JSON")
	packageName := os.Getenv("ANDROID_PACKAGE_NAME")
	if packageName == "" {
		packageName = "com.example.noteika_mobile" // Default package name
	}

	if saJSON == "" {
		// Fallback for local sandbox testing
		log.Printf("[IAP] Warning: GOOGLE_PLAY_SERVICE_ACCOUNT_JSON is not set. Simulating Android IAP verification as SUCCESS.")
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := google.JWTConfigFromJSON([]byte(saJSON), "https://www.googleapis.com/auth/androidpublisher")
	if err != nil {
		return false, fmt.Errorf("failed to parse Google Service Account JSON: %w", err)
	}

	client := config.Client(ctx)
	url := fmt.Sprintf("https://androidpublisher.googleapis.com/androidpublisher/v3/applications/%s/purchases/subscriptions/%s/tokens/%s", packageName, subscriptionID, token)

	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to fetch purchase info from Google Play: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("google play verification returned status %d: %s", resp.StatusCode, string(body))
	}

	var subscriptionPurchase struct {
		StartTimeMillis    string `json:"startTimeMillis"`
		ExpiryTimeMillis   string `json:"expiryTimeMillis"`
		AutoResumeTimeMillis string `json:"autoResumeTimeMillis"`
		PaymentState       int    `json:"paymentState"`
		CancelReason       int    `json:"cancelReason"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&subscriptionPurchase); err != nil {
		return false, fmt.Errorf("failed to decode google subscription response: %w", err)
	}

	// Double check expiry
	var expiryMs int64
	_, _ = fmt.Sscan(subscriptionPurchase.ExpiryTimeMillis, &expiryMs)
	if expiryMs > 0 {
		expiryTime := time.Unix(expiryMs/1000, 0)
		if time.Now().After(expiryTime) {
			log.Printf("[IAP] Android subscription expired at: %v", expiryTime)
			return false, nil
		}
	}

	return true, nil
}

func (h *BillingHandler) verifyIOSPurchase(receiptData string) (bool, error) {
	sharedSecret := os.Getenv("APPLE_IAP_SHARED_SECRET")

	if sharedSecret == "" {
		// Fallback for local sandbox testing
		log.Printf("[IAP] Warning: APPLE_IAP_SHARED_SECRET is not set. Simulating iOS IAP verification as SUCCESS.")
		return true, nil
	}

	requestPayload := map[string]interface{}{
		"receipt-data":             receiptData,
		"password":                 sharedSecret,
		"exclude-old-transactions": true,
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return false, err
	}

	// Verify receipt using Apple verifyReceipt sandbox/production check
	active, err := h.queryAppleVerificationServer("https://sandbox.itunes.apple.com/verifyReceipt", payloadBytes)
	if err == nil && active {
		return true, nil
	}

	return h.queryAppleVerificationServer("https://buy.itunes.apple.com/verifyReceipt", payloadBytes)
}

func (h *BillingHandler) queryAppleVerificationServer(url string, payload []byte) (bool, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("apple verification endpoint returned status %d", resp.StatusCode)
	}

	var appleResponse struct {
		Status   int `json:"status"`
		Receipt  struct {
			InApp []struct {
				ExpiresDateMs string `json:"expires_date_ms"`
				ProductID     string `json:"product_id"`
			} `json:"in_app"`
		} `json:"receipt"`
		LatestReceiptInfo []struct {
			ExpiresDateMs string `json:"expires_date_ms"`
			ProductID     string `json:"product_id"`
		} `json:"latest_receipt_info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&appleResponse); err != nil {
		return false, err
	}

	if appleResponse.Status != 0 {
		return false, fmt.Errorf("apple receipt rejected with status code %d", appleResponse.Status)
	}

	transactions := appleResponse.LatestReceiptInfo
	if len(transactions) == 0 {
		transactions = appleResponse.Receipt.InApp
	}

	for _, tx := range transactions {
		var expiryMs int64
		_, _ = fmt.Sscan(tx.ExpiresDateMs, &expiryMs)
		if expiryMs > 0 {
			expiryTime := time.Unix(expiryMs/1000, 0)
			if time.Now().Before(expiryTime) {
				return true, nil
			}
		}
	}

	return false, nil
}

package web

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"my-app/actor"
	"my-app/db"
	"my-app/supervisor"
)

type Server struct {
	app        *fiber.App
	gateway    *actor.ActorGateway
	supervisor *supervisor.Supervisor
	registry   *actor.ActorRegistry
}

func NewServer(gateway *actor.ActorGateway, sup *supervisor.Supervisor, reg *actor.ActorRegistry) *Server {
	app := fiber.New(fiber.Config{
		AppName: "Go Fiber OTP Supervisor App",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("[Fiber] Error handled: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
		},
	})

	return &Server{
		app:        app,
		gateway:    gateway,
		supervisor: sup,
		registry:   reg,
	}
}

func (s *Server) Start(port string) error {
	// Add middleware
	s.app.Use(recover.New())
	s.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))

	// Setup handlers
	authHandler := NewAuthHandler(s.gateway)
	billingHandler := NewBillingHandler(s.gateway)
	capturesHandler := NewCapturesHandler(s.gateway, billingHandler)
	vaultHandler := NewVaultHandler(s.gateway)
	accountHandler := NewAccountHandler(s.gateway)
	debugHandler := NewDebugHandler(s.supervisor, s.registry, s.gateway)

	// Static assets route
	s.app.Static("/static", "./static")

	// Favicon (browsers request /favicon.ico by default)
	s.app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("./static/favicon.svg")
	})

	// ==========================================
	// PAGES / SPA SHELL ROUTER
	// ==========================================
	// Serve index.html for main pages
	pageHandler := func(c *fiber.Ctx) error {
		return c.SendFile("./static/index.html")
	}
	s.app.Get("/", pageHandler)
	s.app.Get("/login", pageHandler)
	s.app.Get("/pricing", pageHandler)
	s.app.Get("/about", pageHandler)
	s.app.Get("/notes", pageHandler)
	s.app.Get("/account", pageHandler)
	s.app.Get("/import", pageHandler)
	s.app.Get("/dev/import", func(c *fiber.Ctx) error {
		return c.Redirect("/import", fiber.StatusMovedPermanently)
	})
	s.app.Get("/dashboard", pageHandler) // legacy redirect target

	// Auth trigger and callback
	s.app.Get("/auth/login/:provider", authHandler.Login)
	s.app.Get("/auth/:provider/callback", authHandler.Callback)
	s.app.Get("/auth/logout", authHandler.Logout)

	// Stripe webhook (no session auth — verified by signature)
	s.app.Post("/webhooks/stripe", billingHandler.Webhook)

	// ==========================================
	// API SECURE ROUTES (REQUIRE MIDDLEWARE)
	// ==========================================
	api := s.app.Group("/api", s.authMiddleware)

	// Current user endpoint
	api.Get("/auth/me", authHandler.Me)

	// Captures CRUD & Search API
	api.Get("/captures", capturesHandler.List)
	api.Post("/captures", capturesHandler.Create)
	api.Post("/captures/import", capturesHandler.Import)
	api.Get("/captures/:id", capturesHandler.Get)
	api.Patch("/captures/:id", capturesHandler.Update)
	api.Delete("/captures/:id", capturesHandler.Delete)
	api.Post("/captures/empty-trash", capturesHandler.EmptyTrash)
	api.Post("/captures/restore/:id", capturesHandler.Restore)
	api.Get("/projects", capturesHandler.ListProjects)
	api.Post("/projects", capturesHandler.CreateProject)
	api.Get("/tags", capturesHandler.ListTags)
	api.Get("/vault/salt", vaultHandler.GetSalt)
	api.Post("/captures/search", capturesHandler.Search)

	// Account
	api.Get("/account/export", accountHandler.Export)
	api.Patch("/account/settings", accountHandler.UpdateSettings)
	api.Delete("/account", accountHandler.DeleteAccount)
	api.Get("/billing/status", billingHandler.BillingStatus)
	api.Post("/billing/checkout", billingHandler.CreateCheckout)
	api.Post("/billing/portal", billingHandler.CreatePortal)

	// Debug & Telemetry
	api.Post("/debug/crash", debugHandler.Crash)
	api.Get("/debug/stats", debugHandler.Stats)

	// SPA fallback for client-side routes (after /api, /static, /auth, /oauth)
	s.app.Get("/*", pageHandler)

	// Run Web Server
	log.Printf("[Fiber] Server starting on port %s", port)
	return s.app.Listen(":" + port)
}

// authMiddleware extracts the session cookie and verifies it via the supervised VaultActor
func (s *Server) authMiddleware(c *fiber.Ctx) error {
	sessionID := c.Cookies(SessionCookieName)
	if sessionID == "" {
		c.Locals("userID", "")
		return c.Next()
	}

	// Verify session in Badger via actor
	res, err := s.gateway.Send(actor.TypeGetSession, actor.GetSessionPayload{ID: sessionID}, 5*time.Second)
	if err != nil {
		if err == actor.ErrActorUnavailable {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "database actor is currently restarting, please retry in a moment",
			})
		}
		log.Printf("[Server] Session lookup failed: %v", err)
		c.Locals("userID", "")
		return c.Next()
	}

	session, ok := res.(db.Session)
	if !ok {
		c.Locals("userID", "")
		return c.Next()
	}

	if time.Now().Unix() > session.ExpiresAt {
		// Session expired, delete it
		_, _ = s.gateway.Send(actor.TypeDeleteSession, actor.DeleteSessionPayload{ID: sessionID}, 5*time.Second)
		c.Locals("userID", "")
		return c.Next()
	}

	c.Locals("userID", session.UserID)
	return c.Next()
}

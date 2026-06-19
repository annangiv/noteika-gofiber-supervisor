package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	notesHandler := NewNotesHandler(s.gateway)
	debugHandler := NewDebugHandler(s.supervisor, s.registry, s.gateway)

	// Static assets route
	s.app.Static("/static", "./static")

	// ==========================================
	// PAGES / SPA SHELL ROUTER
	// ==========================================
	// Serve index.html for main pages
	pageHandler := func(c *fiber.Ctx) error {
		return c.SendFile("./static/index.html")
	}
	s.app.Get("/", pageHandler)
	s.app.Get("/login", pageHandler)
	s.app.Get("/dashboard", pageHandler)

	// ==========================================
	// NATIVE FIBER MOCK OAUTH ENDPOINTS
	// ==========================================
	s.app.Get("/oauth/mock/authorize", func(c *fiber.Ctx) error {
		provider := c.Query("provider")
		state := c.Query("state")

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Developer Mock Authorization</title>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
					background-color: #0d1117;
					color: #c9d1d9;
					display: flex;
					justify-content: center;
					align-items: center;
					height: 100vh;
					margin: 0;
				}
				.card {
					background-color: #161b22;
					border: 1px solid #30363d;
					border-radius: 12px;
					padding: 32px;
					width: 360px;
					box-shadow: 0 8px 24px rgba(0,0,0,0.5);
				}
				h2 { margin-top: 0; color: #58a6ff; font-weight: 500; }
				label { display: block; margin: 16px 0 6px; font-size: 14px; color: #8b949e; }
				input {
					width: 100%%;
					padding: 10px;
					background-color: #0d1117;
					border: 1px solid #30363d;
					border-radius: 6px;
					color: #fff;
					font-size: 14px;
					box-sizing: border-box;
				}
				input:focus { border-color: #58a6ff; outline: none; }
				button {
					width: 100%%;
					padding: 12px;
					background-color: #238636;
					color: white;
					border: none;
					border-radius: 6px;
					font-size: 16px;
					font-weight: 600;
					margin-top: 24px;
					cursor: pointer;
					transition: background 0.2s;
				}
				button:hover { background-color: #2ea44f; }
				.info { font-size: 12px; color: #8b949e; margin-top: 16px; text-align: center; line-height: 1.4; }
			</style>
		</head>
		<body>
			<div class="card">
				<h2>Mock OAuth Login</h2>
				<div style="font-size: 14px; color: #8b949e; margin-bottom: 20px;">
					Simulating login for <strong>%s</strong>
				</div>
				<form action="/oauth/mock/submit" method="POST">
					<input type="hidden" name="provider" value="%s">
					<input type="hidden" name="state" value="%s">

					<label for="name">Full Name</label>
					<input type="text" id="name" name="name" value="Developer User" required>

					<label for="email">Email Address</label>
					<input type="email" id="email" name="email" value="dev-user@example.com" required>

					<button type="submit">Authorize Developer App</button>
				</form>
				<div class="info">
					This bypasses external APIs and generates a secure local session. Use any test account name or email.
				</div>
			</div>
		</body>
		</html>
		`, provider, provider, state)

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	})

	s.app.Post("/oauth/mock/submit", func(c *fiber.Ctx) error {
		provider := c.FormValue("provider")
		state := c.FormValue("state")
		name := c.FormValue("name")
		email := c.FormValue("email")

		if name == "" || email == "" {
			return c.Status(400).SendString("Name and Email are required")
		}

		profile := struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}{
			Email: email,
			Name:  name,
		}

		jsonBytes, _ := json.Marshal(profile)
		encodedPayload := base64.URLEncoding.EncodeToString(jsonBytes)
		mockCode := fmt.Sprintf("mock_%s", encodedPayload)

		callbackURL := fmt.Sprintf("/auth/%s/callback?code=%s&state=%s", provider, mockCode, url.QueryEscape(state))
		return c.Redirect(callbackURL)
	})

	// Auth trigger and callback
	s.app.Get("/auth/login/:provider", authHandler.Login)
	s.app.Get("/auth/:provider/callback", authHandler.Callback)
	s.app.Get("/auth/logout", authHandler.Logout)

	// ==========================================
	// API SECURE ROUTES (REQUIRE MIDDLEWARE)
	// ==========================================
	api := s.app.Group("/api", s.authMiddleware)

	// Current user endpoint
	api.Get("/auth/me", authHandler.Me)

	// Notes CRUD
	api.Get("/notes", notesHandler.List)
	api.Post("/notes", notesHandler.Create)
	api.Delete("/notes/:id", notesHandler.Delete)

	// Debug & Telemetry
	api.Post("/debug/crash", debugHandler.Crash)
	api.Get("/debug/stats", debugHandler.Stats)

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

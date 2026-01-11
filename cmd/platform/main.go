package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/serbia-gov/platform/internal/agency"
	"github.com/serbia-gov/platform/internal/ai"
	"github.com/serbia-gov/platform/internal/audit"
	caseapi "github.com/serbia-gov/platform/internal/case/api"
	caseinfra "github.com/serbia-gov/platform/internal/case/infrastructure"
	"github.com/serbia-gov/platform/internal/document"
	"github.com/serbia-gov/platform/internal/federation/trust"
	"github.com/serbia-gov/platform/internal/privacy"
	"github.com/serbia-gov/platform/internal/simulation"
	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/config"
	"github.com/serbia-gov/platform/internal/shared/database"
	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/metrics"
	secmiddleware "github.com/serbia-gov/platform/internal/shared/middleware"
)

// App holds all application dependencies
type App struct {
	Config       *config.Config
	DB           *database.DB
	Bus          *events.Bus
	PrivacyGuard *privacy.PrivacyGuard
}

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	app := &App{Config: cfg}

	// Initialize database (optional - skip if not available)
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		fmt.Printf("Warning: Database not available: %v\n", err)
		fmt.Println("Running in limited mode without database...")
	} else {
		app.DB = db
		defer db.Close()

		// Run migrations
		if err := database.Migrate(ctx, db.Pool); err != nil {
			fmt.Printf("Warning: Migration failed: %v\n", err)
		}
	}

	// Initialize event bus with KurrentDB (optional - skip if not available)
	bus, err := events.NewBus(ctx, cfg.KurrentDB)
	if err != nil {
		fmt.Printf("Warning: KurrentDB not available: %v\n", err)
		fmt.Println("Running without event streaming...")
	} else {
		app.Bus = bus
		defer bus.Close()
		fmt.Println("KurrentDB Event Bus initialized")
	}

	// Initialize Privacy Guard (optional - skip in local-only mode)
	if cfg.Privacy.EnablePrivacyGuard && cfg.Privacy.FacilityType == "central" {
		// Create a simple violation logger using audit
		var violationHandler privacy.ViolationHandler
		if app.Bus != nil {
			// Use KurrentDB-based audit repository
			auditRepo := audit.NewKurrentDBRepository(app.Bus.Client())
			// Wrap audit repo as violation handler
			violationHandler = &auditViolationHandler{auditRepo: auditRepo}
		}

		guardConfig := privacy.PrivacyGuardConfig{
			ExemptPaths:      cfg.Privacy.ExemptPaths,
			ExemptPrefixes:   cfg.Privacy.ExemptPrefixes,
			BlockOnViolation: cfg.Privacy.BlockOnViolation,
			LogViolations:    cfg.Privacy.LogViolations,
		}
		app.PrivacyGuard = privacy.NewPrivacyGuard(violationHandler, guardConfig)
		fmt.Printf("Privacy Guard enabled (facility type: %s)\n", cfg.Privacy.FacilityType)
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(secmiddleware.SecurityHeaders)
	r.Use(metrics.Middleware)
	r.Use(corsMiddleware)

	// Privacy Guard middleware (if enabled for central system)
	if app.PrivacyGuard != nil {
		r.Use(app.PrivacyGuard.Middleware)
	}

	// Health checks (unauthenticated)
	r.Get("/health", healthHandler(app))
	r.Get("/ready", readyHandler(app))
	r.Handle("/metrics", metrics.Handler())

	// API info
	r.Get("/", infoHandler)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (no auth required for now in dev mode)
		if cfg.Server.Env == "production" {
			r.Use(auth.Middleware(cfg.Auth))
		}

		// Agency module
		if app.DB != nil {
			agencyRepo := agency.NewRepository(app.DB.Pool)
			agencyHandler := agency.NewHandler(agencyRepo, app.Bus)
			r.Mount("/", agencyHandler.Routes())

			// Case module
			caseRepo := caseinfra.NewPostgresRepository(app.DB.Pool)
			caseHandler := caseapi.NewHandler(caseRepo, app.Bus)
			r.Mount("/cases", caseHandler.Routes())

			// Document module
			documentRepo := document.NewRepository(app.DB.Pool)
			documentHandler := document.NewHandler(documentRepo, app.Bus)
			r.Mount("/documents", documentHandler.Routes())

			// Audit module - uses KurrentDB (append-only event store)
			if app.Bus != nil {
				auditRepo := audit.NewKurrentDBRepository(app.Bus.Client())
				if err := auditRepo.Initialize(ctx); err != nil {
					fmt.Printf("Warning: Audit initialization failed: %v\n", err)
				}
				auditHandler := audit.NewHandler(auditRepo)
				r.Mount("/audit", auditHandler.Routes())

				// Start audit subscriber
				auditSubscriber := audit.NewSubscriber(auditRepo, app.Bus)
				if err := auditSubscriber.Start(ctx); err != nil {
					fmt.Printf("Warning: Audit subscriber failed to start: %v\n", err)
				} else {
					fmt.Println("Audit subscriber started (KurrentDB)")
				}
			}

			// Federation - Trust Authority
			trustAuthority, err := trust.NewAuthority(nil) // nil repo for in-memory MVP
			if err != nil {
				fmt.Printf("Warning: Trust Authority initialization failed: %v\n", err)
			} else {
				// Seed Kikinda pilot agencies
				seedKikindaPilot(trustAuthority)

				trustHandler := trust.NewHandler(trustAuthority)
				r.Mount("/federation/trust", trustHandler.Routes())
				fmt.Println("Federation Trust Authority initialized")
			}
		}

		// AI Module - always available (connects to AI mock service)
		if cfg.AI.Enabled {
			aiClient := ai.NewClient(ai.ClientConfig{
				BaseURL: cfg.AI.URL,
			})
			aiHandler := ai.NewHandler(aiClient)
			r.Mount("/ai", aiHandler.Routes())
			fmt.Printf("AI Module enabled (service: %s)\n", cfg.AI.URL)
		}

		// Simulation Module - for demo/training purposes
		if app.Bus != nil {
			simHandler := simulation.NewHandler(app.Bus)
			r.Mount("/simulation", simHandler.Routes())
			fmt.Println("Simulation Module enabled")
		}
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			fmt.Printf("Server shutdown error: %v\n", err)
		}
		close(done)
	}()

	fmt.Println("============================================")
	fmt.Println("Serbia Government Interoperability Platform")
	fmt.Println("============================================")
	fmt.Printf("Environment:    %s\n", cfg.Server.Env)
	fmt.Printf("Server:         http://localhost:%d\n", cfg.Server.Port)
	fmt.Printf("API:            http://localhost:%d/api/v1\n", cfg.Server.Port)
	fmt.Printf("Health:         http://localhost:%d/health\n", cfg.Server.Port)
	fmt.Printf("Facility Type:  %s\n", cfg.Privacy.FacilityType)
	fmt.Printf("Facility Code:  %s\n", cfg.Privacy.FacilityCode)
	fmt.Printf("Privacy Guard:  %v\n", cfg.Privacy.EnablePrivacyGuard)
	fmt.Printf("KurrentDB:      %s:%d\n", cfg.KurrentDB.Host, cfg.KurrentDB.Port)
	fmt.Println("============================================")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}

	<-done
	fmt.Println("Server stopped")
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":    "Serbia Government Interoperability Platform",
		"version": "0.1.0",
		"status":  "MVP Development",
		"docs":    "/api/v1",
	})
}

func healthHandler(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}
}

func readyHandler(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{
			"server": "ready",
		}

		// Check database
		if app.DB != nil {
			if err := app.DB.Health(r.Context()); err != nil {
				checks["database"] = "not ready: " + err.Error()
			} else {
				checks["database"] = "ready"
			}
		} else {
			checks["database"] = "not configured"
		}

		// Check KurrentDB
		if app.Bus != nil {
			if err := app.Bus.Health(); err != nil {
				checks["kurrentdb"] = "not ready: " + err.Error()
			} else {
				checks["kurrentdb"] = "ready"
			}
		} else {
			checks["kurrentdb"] = "not configured"
		}

		allReady := true
		for _, status := range checks {
			if status != "ready" && status != "not configured" {
				allReady = false
				break
			}
		}

		status := http.StatusOK
		if !allReady {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{
			"status": map[bool]string{true: "ready", false: "not ready"}[allReady],
			"checks": checks,
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// auditViolationHandler wraps audit repository to implement ViolationHandler.
type auditViolationHandler struct {
	auditRepo audit.AuditRepository
}

func (h *auditViolationHandler) HandleViolation(ctx context.Context, violation *privacy.PIIViolation) error {
	if h.auditRepo == nil {
		return nil
	}

	action := privacy.AuditActionPIIViolationDetected
	if violation.Blocked {
		action = privacy.AuditActionPIIViolationBlocked
	}

	entry := audit.NewAuditEntry(
		audit.ActorTypeSystem,
		"privacy-guard",              // ActorID
		nil,                          // ActorAgencyID
		action,                       // Action
		"pii_violation",              // ResourceType
		&violation.ID,                // ResourceID
		map[string]any{
			"field":          violation.Field,
			"location":       violation.Location,
			"blocked":        violation.Blocked,
			"masked_value":   violation.MaskedValue,
			"request_path":   violation.RequestPath,
			"request_method": violation.RequestMethod,
		},
		"", // prevHash - will be set by repository
	)

	return h.auditRepo.Append(ctx, entry)
}

// seedKikindaPilot registers Kikinda pilot agencies in the Trust Authority
func seedKikindaPilot(authority *trust.Authority) {
	ctx := context.Background()

	// Kikinda pilot - full hierarchy from local to national level
	agencies := []struct {
		name       string
		code       string
		gatewayURL string
	}{
		// === NACIONALNI NIVO ===
		{"Vlada Republike Srbije", "VLADA-RS", "https://vlada.gov.rs/api"},
		{"Republički zavod za statistiku", "RZS", "https://stat.gov.rs/api"},

		// === MINISTARSTVA ===
		{"Ministarstvo za rad, zapošljavanje, boračka i socijalna pitanja", "MINRZS", "https://minrzs.gov.rs/api"},
		{"Ministarstvo zdravlja", "MINZDRAVLJA", "https://zdravlje.gov.rs/api"},
		{"Ministarstvo unutrašnjih poslova", "MUP", "https://mup.gov.rs/api"},
		{"Ministarstvo prosvete", "MINPROSVETE", "https://prosveta.gov.rs/api"},
		{"Ministarstvo pravde", "MINPRAVDE", "https://mpravde.gov.rs/api"},
		{"Ministarstvo za brigu o porodici i demografiju", "MINDEM", "https://minbpd.gov.rs/api"},

		// === KIKINDA - LOKALNI NIVO ===
		{"Opština Kikinda", "OU-KI", "https://opstina.kikinda.gov.rs/api"},
		{"Osnovni sud u Kikindi", "SUD-KI", "https://ki.os.sud.rs/api"},

		// === KIKINDA - SOCIJALNA ZAŠTITA ===
		{"Centar za socijalni rad Kikinda", "CSR-KI", "https://csr.kikinda.gov.rs/api"},
		{"Gerontološki centar Kikinda", "GC-KI", "https://gc.kikinda.gov.rs/api"},
		{"NSZ Filijala Kikinda", "NSZ-KI", "https://nsz.kikinda.gov.rs/api"},

		// === KIKINDA - ZDRAVSTVO ===
		{"Dom zdravlja Kikinda", "DZ-KI", "https://dz.kikinda.gov.rs/api"},
		{"Opšta bolnica Kikinda", "OB-KI", "https://bolnica.kikinda.gov.rs/api"},
		{"Apoteka Kikinda", "APO-KI", "https://apoteka.kikinda.gov.rs/api"},

		// === KIKINDA - BEZBEDNOST ===
		{"Policijska uprava Kikinda", "PU-KI", "https://pu.kikinda.gov.rs/api"},

		// === KIKINDA - OBRAZOVANJE ===
		{"Predškolska ustanova \"Dragoljub Udicki\"", "PU-DU-KI", "https://vrtic.kikinda.gov.rs/api"},
		{"Osnovna škola \"Vuk Karadžić\" Kikinda", "OS-VK-KI", "https://osvuk.kikinda.edu.rs/api"},
		{"Gimnazija \"Dušan Vasiljev\" Kikinda", "GIM-KI", "https://gimnazija.kikinda.edu.rs/api"},
	}

	// Register all pilot agencies
	for _, a := range agencies {
		agency, err := authority.RegisterAgency(ctx, a.name, a.code, a.gatewayURL)
		if err != nil {
			fmt.Printf("Warning: Failed to register %s: %v\n", a.code, err)
			continue
		}

		// Register standard services for each agency
		services := []struct {
			serviceType string
			path        string
			version     string
		}{
			{"case.share", "/api/v1/cases/share", "1.0"},
			{"case.transfer", "/api/v1/cases/transfer", "1.0"},
			{"document.exchange", "/api/v1/documents/exchange", "1.0"},
			{"document.verify", "/api/v1/documents/verify", "1.0"},
			{"notification.send", "/api/v1/notifications", "1.0"},
		}

		for _, s := range services {
			_, err := authority.RegisterService(ctx, agency.ID, s.serviceType, s.path, s.version)
			if err != nil {
				fmt.Printf("Warning: Failed to register service %s for %s: %v\n", s.serviceType, a.code, err)
			}
		}
	}

	fmt.Println("Kikinda pilot agencies registered (20 agencies - local to national)")
}

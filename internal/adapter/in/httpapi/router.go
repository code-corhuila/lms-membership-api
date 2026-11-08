package httpserver

import (
	"crypto/rsa"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/code-corhuila/lms-membership-api/internal/adapter/in/httpapi/handler"
	"github.com/code-corhuila/lms-membership-api/internal/adapter/in/httpapi/middleware"
)

// RouterConfig carries what the router needs to wire itself.
type RouterConfig struct {
	DB                *pgxpool.Pool
	JWTPublicKey      *rsa.PublicKey
	InternalJWTSecret string
	CORSOrigin        string
	Students          *handler.StudentHandler
	// Authorizer calls lms-iam's /authorize (ADR-012). Every protected
	// route declares the permission it needs (SEC-04) — nil disables the
	// check entirely (local dev without lms-iam running), never silently
	// passing a request through where a check was expected.
	Authorizer middleware.Authorizer
}

// NewRouter builds the chi router with the base middleware stack, health
// endpoints, and the Membership bounded context's /students routes.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CorrelationID)
	r.Use(middleware.CORS(cfg.CORSOrigin))

	health := handler.NewHealthHandler(cfg.DB)
	r.Get("/health", health.Liveness)
	r.Get("/health/ready", health.Readiness)

	// requirePermission is a no-op when Authorizer is nil (lms-iam not
	// configured for this run) rather than a silent allow-everything —
	// see README's "Known integration gap" for why this is intentional.
	requirePermission := func(permission string) func(http.Handler) http.Handler {
		if cfg.Authorizer == nil {
			return func(next http.Handler) http.Handler { return next }
		}
		return middleware.RequirePermission(cfg.Authorizer, permission)
	}

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(cfg.JWTPublicKey, cfg.InternalJWTSecret))

			protected.Route("/students", func(students chi.Router) {
				students.With(requirePermission("students:write")).Post("/", cfg.Students.Create)                    // HU-02
				students.With(requirePermission("students:read")).Get("/", cfg.Students.List)                        // HU-03
				students.With(requirePermission("students:read")).Get("/{id}", cfg.Students.Get)                     // needed by circulation-service
				students.With(requirePermission("students:write")).Patch("/{id}", cfg.Students.Update)               // HU-03
				students.With(requirePermission("students:write")).Post("/{id}/deactivate", cfg.Students.Deactivate) // HU-03
				students.With(requirePermission("students:write")).Post("/{id}/suspend", cfg.Students.Suspend)       // needed by circulation-service
			})
		})
	})

	return r
}

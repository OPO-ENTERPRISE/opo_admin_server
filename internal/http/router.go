package http

import (
	"net/http"

	"opo_admin_server/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(cfg config.Config) http.Handler {
	r := chi.NewRouter()

	// Middleware CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Middleware de logging
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log de requests
			next.ServeHTTP(w, r)
		})
	})

	// Rutas públicas
	r.Route(cfg.APIBasePath, func(r chi.Router) {
		// Health check
		r.Get("/healthz", Healthz)

		// Test CORS endpoint
		r.Get("/test-cors", TestCORS)

		// Test login endpoint (sin base de datos)
		r.Post("/test-login", TestLogin)

		// Autenticación (solo login)
		r.Post("/auth/login", AuthLogin(cfg))

		// Topics públicos (filtrados por área)
		r.Get("/topics/area/{areaId}", TopicListByArea(cfg))
	})

	// Rutas protegidas (requieren JWT)
	r.Route(cfg.APIBasePath+"/admin", func(r chi.Router) {
		r.Use(AuthJWT(cfg))

		// Gestión del usuario administrador
		r.Get("/user", AdminUserGet(cfg))
		r.Put("/user", AdminUserUpdate(cfg))
		r.Post("/user/reset-password", AdminUserResetPassword(cfg))

		// Administración de topics
		r.Get("/topics", AdminTopicsList(cfg))
		r.Get("/topics/{id}", AdminTopicsGetByID(cfg))
		r.Get("/topics/{id}/subtopics", AdminTopicsGetSubtopics(cfg))
		r.Post("/topics", AdminTopicsCreate(cfg))
		r.Put("/topics/{id}", AdminTopicsUpdate(cfg))
		r.Patch("/topics/{id}/enabled", AdminTopicsToggleEnabled(cfg))
		r.Patch("/topics/{id}/premium", AdminTopicsTogglePremium(cfg))
		r.Delete("/topics/{id}", AdminTopicsDelete(cfg))

		// Administración de áreas
		r.Get("/areas", AdminAreasList(cfg))
		r.Get("/areas/{id}", AdminAreasGetByID(cfg))
		r.Post("/areas", AdminAreasCreate(cfg))
		r.Put("/areas/{id}", AdminAreasUpdate(cfg))
		r.Patch("/areas/{id}/enabled", AdminAreasToggleEnabled(cfg))
		r.Delete("/areas/{id}", AdminAreasDelete(cfg))

		// Administración de usuarios
		r.Get("/users", AdminUsersList(cfg))
		r.Patch("/users/{id}/enabled", AdminUsersToggleEnabled(cfg))

		// Estadísticas
		r.Get("/stats/user", AdminStatsUser(cfg))
		r.Get("/stats/topics", AdminStatsTopics(cfg))
		r.Get("/stats/area/{areaId}", AdminStatsArea(cfg))
		r.Get("/stats/areas", AdminStatsAllAreas(cfg))
	})

	return r
}

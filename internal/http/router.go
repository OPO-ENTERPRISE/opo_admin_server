package http

import (
	"net/http"

	"opo_admin_server/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	println("CORSAllowedOrigins: ", cfg.CORSAllowedOrigins)

	// Middleware CORS mejorado
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With", "Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"},
		ExposedHeaders:   []string{"Link", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 horas
	}))

	// Middleware de logging mejorado
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			println("Request:", r.Method, r.URL.Path, "Origin:", r.Header.Get("Origin"))
			next.ServeHTTP(w, r)
		})
	})

	// Handler global para OPTIONS (captura todas las rutas)
	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
		// El middleware CORS ya configuró los headers
		w.WriteHeader(http.StatusNoContent)
	})

	// Rutas públicas
	r.Route(cfg.APIBasePath, func(r chi.Router) {
		// Handler para OPTIONS dentro del grupo API
		r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

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

		// Handler para OPTIONS en rutas protegidas
		r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		// Gestión del usuario administrador
		r.Get("/user", AdminUserGet(cfg))
		r.Put("/user", AdminUserUpdate(cfg))
		r.Post("/user/reset-password", AdminUserResetPassword(cfg))

		// Administración de topics
		r.Get("/topics", AdminTopicsList(cfg))
		r.Get("/topics/{id}", AdminTopicsGetByID(cfg))
		r.Get("/topics/{id}/subtopics", AdminTopicsGetSubtopics(cfg))
		r.Post("/topics/{id}/subtopics", AdminTopicsCreateSubtopic(cfg))
		r.Post("/topics", AdminTopicsCreate(cfg))
		r.Put("/topics/{id}", AdminTopicsUpdate(cfg))
		r.Patch("/topics/{id}/enabled", AdminTopicsToggleEnabled(cfg))
		r.Patch("/topics/{id}/premium", AdminTopicsTogglePremium(cfg))
		r.Delete("/topics/{id}", AdminTopicsDelete(cfg))
		
		// Gestión de preguntas de topics
		r.Get("/topics/{id}/available-sources", AdminGetAvailableSourceTopics(cfg))
		r.Post("/topics/{id}/copy-questions", AdminCopyQuestionsFromTopics(cfg))
		r.Post("/topics/{id}/upload-questions", AdminUploadQuestionsToTopic(cfg))

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

		// Administración de proveedores de publicidad
		r.Get("/providers", AdminProvidersList(cfg))
		r.Get("/providers/{id}", AdminProvidersGetByID(cfg))
		r.Post("/providers", AdminProvidersCreate(cfg))
		r.Put("/providers/{id}", AdminProvidersUpdate(cfg))
		r.Patch("/providers/{id}/enabled", AdminProvidersToggleEnabled(cfg))
		r.Delete("/providers/{id}", AdminProvidersDelete(cfg))

		// Estadísticas
		r.Get("/stats/user", AdminStatsUser(cfg))
		r.Get("/stats/topics", AdminStatsTopics(cfg))
		r.Get("/stats/area/{areaId}", AdminStatsArea(cfg))
		r.Get("/stats/areas", AdminStatsAllAreas(cfg))

		// Gestión de base de datos
		r.Get("/database/stats", AdminDatabaseStats(cfg))
		r.Get("/database/download", AdminDatabaseDownload(cfg))
	})

	return r
}

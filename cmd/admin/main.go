package main

import (
	"log"
	"net/http"

	"opo_admin_server/internal/config"
	httpapi "opo_admin_server/internal/http"
)

func main() {
	// Cargar configuración
	cfg := config.Load()

	// Crear router
	router := httpapi.NewRouter(cfg)

	// Iniciar servidor
	log.Printf("🚀 Iniciando servidor de administración en puerto %s", cfg.Port)
	log.Printf("📡 API Base Path: %s", cfg.APIBasePath)
	log.Printf("🌐 CORS Origins: %v", cfg.CORSAllowedOrigins)

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("❌ Error al iniciar servidor: %v", err)
	}
}

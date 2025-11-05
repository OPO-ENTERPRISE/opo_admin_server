package main

import (
	"log"
	"net/http"
	"time"

	"opo_admin_server/internal/config"
	httpapi "opo_admin_server/internal/http"
)

func main() {
	// Cargar configuración
	cfg := config.Load()

	// Crear router
	router := httpapi.NewRouter(cfg)

	// Configurar servidor HTTP con límites aumentados para archivos grandes
	port := ":" + cfg.Port
	server := &http.Server{
		Addr:         port,
		Handler:      router,
		ReadTimeout:  30 * time.Minute,  // Tiempo para leer el request completo (archivos grandes)
		WriteTimeout: 30 * time.Minute,  // Tiempo para escribir la respuesta
		IdleTimeout:  120 * time.Second, // Tiempo de conexión idle
		MaxHeaderBytes: 1 << 20,         // 1MB para headers (suficiente para multipart)
	}

	log.Printf("🚀 Iniciando servidor de administración en puerto %s", cfg.Port)
	log.Printf("📡 API Base Path: %s", cfg.APIBasePath)
	log.Printf("🌐 CORS Origins: %v", cfg.CORSAllowedOrigins)
	log.Printf("📦 Límite de request: 100MB (configurado en handlers)")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Error al iniciar servidor: %v", err)
	}
}

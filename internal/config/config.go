package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	APIBasePath        string
	JWTSecret          string
	CORSAllowedOrigins []string
	DBURL              string
	DBName             string
	PineconeAPIKey     string
}

func Load() Config {
	// Cargar archivo .env si existe
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró archivo .env, usando variables de entorno del sistema")
	}

	// Logs de debug para variables de entorno
	log.Println("=== CONFIGURACIÓN ADMIN SERVER ===")
	log.Printf("PORT: %s", os.Getenv("PORT"))
	log.Printf("API_BASE_PATH: %s", os.Getenv("API_BASE_PATH"))
	log.Printf("JWT_SECRET: %s", os.Getenv("JWT_SECRET"))
	log.Printf("CORS_ALLOWED_ORIGINS: %s", os.Getenv("CORS_ALLOWED_ORIGINS"))
	log.Printf("DB_NAME: %s", os.Getenv("DB_NAME"))
	log.Printf("DB_URL: %s", os.Getenv("DB_URL"))
	log.Printf("MONGO_URL: %s", os.Getenv("MONGO_URL"))
	log.Println("=== FIN CONFIGURACIÓN ===")

	// Log específico para MongoDB
	log.Println("=== MONGO CONFIGURATION DEBUG ===")
	log.Printf("🔍 DB_URL from env: %s", os.Getenv("DB_URL"))
	log.Printf("🔍 MONGO_URL from env: %s", os.Getenv("MONGO_URL"))
	log.Printf("🔍 DB_NAME from env: %s", os.Getenv("DB_NAME"))
	log.Println("=== END MONGO DEBUG ===")

	dbURL := getenv("DB_URL", "")
	if dbURL == "" {
		dbURL = getenv("MONGO_URL", "")
	}
	if dbURL == "" {
		log.Fatal("No se encontró DB_URL ni MONGO_URL en las variables de entorno")
	}

	// Log final de la URL de MongoDB que se va a usar
	log.Printf("🔗 MongoDB URL final que se usará: %s", dbURL)

	// Parsear CORS origins
	corsOrigins := getenv("CORS_ALLOWED_ORIGINS", "http://localhost:8100,https://localhost:8100,capacitor://localhost,ionic://localhost")

	// Limpiar cualquier = al inicio (problema de Cloud Run)
	corsOrigins = strings.TrimPrefix(corsOrigins, "=")

	var allowedOrigins []string
	for _, origin := range strings.Split(corsOrigins, ",") {
		origin = strings.TrimSpace(origin)
		// Limpiar cualquier = al inicio de cada origen
		origin = strings.TrimPrefix(origin, "=")
		if origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}

	// Log de debug para ver los orígenes parseados
	log.Printf("🌐 CORS Origins parseados: %v", allowedOrigins)

	config := Config{
		Port:               getenv("PORT", "8080"), // Puerto diferente para admin
		APIBasePath:        getenv("API_BASE_PATH", "/api/v1"),
		JWTSecret:          getenv("JWT_SECRET", "admin-secret-key"),
		CORSAllowedOrigins: allowedOrigins,
		DBURL:              dbURL,
		DBName:             getenv("DB_NAME", "opo"),
		PineconeAPIKey:     getenv("PINECONE_API_KEY", ""),
	}

	// Log de la configuración final
	log.Printf("=== CONFIGURACIÓN FINAL ADMIN ===")
	log.Printf("Port: %s", config.Port)
	log.Printf("APIBasePath: %s", config.APIBasePath)
	log.Printf("JWTSecret: %s", config.JWTSecret)
	log.Printf("CORSAllowedOrigins: %v", config.CORSAllowedOrigins)
	log.Printf("DBURL: %s", config.DBURL)
	log.Printf("DBName: %s", config.DBName)
	if config.PineconeAPIKey != "" {
		log.Printf("PineconeAPIKey: %s***", config.PineconeAPIKey[:min(10, len(config.PineconeAPIKey))])
	} else {
		log.Printf("PineconeAPIKey: (no configurado)")
	}
	log.Println("=== FIN CONFIGURACIÓN FINAL ===")

	return config
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

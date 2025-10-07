package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opo_admin_server/internal/config"
	"opo_admin_server/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Healthz - Endpoint de salud
func Healthz(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "ok",
		"ts":     time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, response)
}

// TestCORS - Test CORS endpoint
func TestCORS(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "CORS test successful",
		"origin":  r.Header.Get("Origin"),
		"method":  r.Method,
		"ts":      time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, response)
}

// TestLogin - Login de prueba sin base de datos
func TestLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
		return
	}

	// Credenciales de prueba
	if req.Email == "admin@example.com" && req.Password == "admin123" {
		// Simular usuario
		user := map[string]interface{}{
			"id":        "test-admin-id",
			"name":      "Administrador Test",
			"email":     "admin@example.com",
			"appId":     "1",
			"createdAt": time.Now().Format(time.RFC3339),
			"updatedAt": time.Now().Format(time.RFC3339),
		}

		// Generar token JWT simple
		token := generateJWT("test-admin-id", "admin@example.com", config.Config{JWTSecret: "test-secret"})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user":  user,
			"token": token,
		})
	} else {
		writeError(w, http.StatusBadRequest, "invalid_credentials", "usuario o contraseña incorrectos")
	}
}

// AuthLogin - Autenticación del usuario administrador
func AuthLogin(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "email y password requeridos")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			log.Printf("❌ Error conectando a MongoDB: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		log.Printf("✅ Conexión a MongoDB exitosa")
		log.Printf("🗄️ Base de datos: %s", cfg.DBName)

		users := client.Database(cfg.DBName).Collection("user")
		log.Printf("👤 Buscando usuario con email: %s", req.Email)

		var user domain.User
		if err := users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				log.Printf("❌ Usuario no encontrado: %s", req.Email)
				writeError(w, http.StatusBadRequest, "invalid_credentials", "usuario o contraseña incorrectos")
				return
			}
			log.Printf("❌ Error buscando usuario: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ Usuario encontrado: %s (ID: %s)", user.Name, user.ID)

		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
			log.Printf("❌ Contraseña incorrecta para usuario: %s", req.Email)
			writeError(w, http.StatusBadRequest, "invalid_credentials", "usuario o contraseña incorrectos")
			return
		}

		log.Printf("✅ Contraseña correcta para usuario: %s", req.Email)

		// Actualizar último login
		users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
			"$set": bson.M{"lastLogin": time.Now()},
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user": map[string]interface{}{
				"id":        user.ID,
				"name":      user.Name,
				"email":     user.Email,
				"appId":     user.AppID,
				"createdAt": user.CreatedAt.Format(time.RFC3339),
				"updatedAt": user.UpdatedAt.Format(time.RFC3339),
			},
			"token": generateJWT(user.ID, user.Email, cfg),
		})
	}
}

// TopicListByArea - Listar topics por área (endpoint público)
func TopicListByArea(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaId := chi.URLParam(r, "areaId")

		areaInt, err := strconv.Atoi(areaId)
		if err != nil || (areaInt != 1 && areaInt != 2) {
			writeError(w, http.StatusBadRequest, "invalid_area", "area debe ser 1 o 2")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Consultar topics_uuid_map filtrado por área y habilitados
		col := client.Database(cfg.DBName).Collection("topics_uuid_map")
		filter := bson.M{
			"area":    areaInt, // Ahora es int directamente
			"enabled": true,
		}

		cur, err := col.Find(ctx, filter, options.Find())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var topics []domain.Topic
		if err := cur.All(ctx, &topics); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Separar topics principales de subtopics
		var mainTopics []domain.Topic
		subtopics := make(map[int][]domain.Topic) // Cambiado a map[int]

		for _, topic := range topics {
			if topic.IsMainTopic() {
				mainTopics = append(mainTopics, topic)
			} else {
				subtopics[topic.RootID] = append(subtopics[topic.RootID], topic)
			}
		}

		// Construir respuesta con jerarquía
		var result []domain.TopicResponse
		for _, mainTopic := range mainTopics {
			response := domain.TopicResponse{
				Title:    mainTopic.Title,
				UUID:     mainTopic.UUID,
				RootUUID: mainTopic.RootUUID,
				ID:       mainTopic.TopicID,
			}

			// Agregar subtopics si existen
			if children, exists := subtopics[mainTopic.TopicID]; exists {
				response.Children = make([]domain.TopicResponse, len(children))
				for i, child := range children {
					response.Children[i] = domain.TopicResponse{
						Title:    child.Title,
						UUID:     child.UUID,
						RootUUID: child.RootUUID,
						ID:       child.TopicID,
					}
				}
			}

			result = append(result, response)
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// Utilidades

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, domain.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func getMongoClient(ctx context.Context, cfg config.Config) (*mongo.Client, error) {
	return mongo.Connect(ctx, options.Client().ApplyURI(cfg.DBURL))
}

func generateJWT(userID, email string, cfg config.Config) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(cfg.JWTSecret))
	return s
}

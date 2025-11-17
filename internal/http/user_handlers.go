package http

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"opo_admin_server/internal/config"
	"opo_admin_server/internal/domain"
	"opo_admin_server/internal/services"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserDeactivateRequest - Solicitud de baja (endpoint público)
func UserDeactivateRequest(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		if req.Email == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "email requerido")
			return
		}

		// Validar formato de email básico
		if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "email inválido")
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

		users := client.Database(cfg.DBName).Collection("user")

		// Verificar si el usuario existe
		var user domain.User
		if err := users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				// Por seguridad, no revelamos si el email existe o no
				log.Printf("⚠️ Solicitud de baja para email no registrado: %s", req.Email)
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"message": "Si el email existe, recibirás un correo con las instrucciones para confirmar la baja.",
				})
				return
			}
			log.Printf("❌ Error buscando usuario: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Verificar si el usuario ya está desactivado
		if !user.Enabled {
			log.Printf("⚠️ Usuario ya desactivado: %s", req.Email)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "Si el email existe, recibirás un correo con las instrucciones para confirmar la baja.",
			})
			return
		}

		// Generar token JWT para confirmación (válido por 24 horas)
		token := generateDeactivationToken(user.ID, user.Email, cfg)

		// Enviar email de confirmación
		emailService := services.NewEmailService(cfg)
		if err := emailService.SendDeactivationEmail(user.Email, token); err != nil {
			log.Printf("❌ Error enviando email de desactivación: %v", err)
			// No revelamos el error al usuario por seguridad
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "Si el email existe, recibirás un correo con las instrucciones para confirmar la baja.",
			})
			return
		}

		log.Printf("✅ Email de desactivación enviado a: %s", req.Email)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Si el email existe, recibirás un correo con las instrucciones para confirmar la baja.",
		})
	}
}

// UserDeactivateConfirm - Confirmación de baja (endpoint público)
func UserDeactivateConfirm(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			renderDeactivationPage(w, false, "Token de confirmación no proporcionado")
			return
		}

		// Validar y decodificar token
		claims, err := validateDeactivationToken(token, cfg)
		if err != nil {
			log.Printf("❌ Token inválido: %v", err)
			renderDeactivationPage(w, false, "Token inválido o expirado. Por favor, solicita una nueva baja.")
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			renderDeactivationPage(w, false, "Token inválido")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			log.Printf("❌ Error conectando a MongoDB: %v", err)
			renderDeactivationPage(w, false, "Error interno del servidor. Por favor, intenta más tarde.")
			return
		}
		defer client.Disconnect(context.Background())

		users := client.Database(cfg.DBName).Collection("user")

		// Buscar usuario
		var user domain.User
		if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				renderDeactivationPage(w, false, "Usuario no encontrado")
				return
			}
			log.Printf("❌ Error buscando usuario: %v", err)
			renderDeactivationPage(w, false, "Error interno del servidor. Por favor, intenta más tarde.")
			return
		}

		// Desactivar usuario
		update := bson.M{
			"$set": bson.M{
				"enabled":   false,
				"updatedAt": time.Now(),
			},
		}

		result, err := users.UpdateOne(ctx, bson.M{"_id": userID}, update)
		if err != nil {
			log.Printf("❌ Error desactivando usuario: %v", err)
			renderDeactivationPage(w, false, "Error al procesar la solicitud. Por favor, intenta más tarde.")
			return
		}

		if result.MatchedCount == 0 {
			renderDeactivationPage(w, false, "Usuario no encontrado")
			return
		}

		log.Printf("✅ Usuario desactivado: %s (ID: %s)", user.Email, userID)

		renderDeactivationPage(w, true, "Tu cuenta ha sido desactivada correctamente.")
	}
}

// generateDeactivationToken genera un token JWT para confirmación de baja
func generateDeactivationToken(userID, email string, cfg config.Config) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"type":  "deactivation",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(), // 24 horas de validez
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(cfg.JWTSecret))
	return s
}

// validateDeactivationToken valida y decodifica un token de desactivación
func validateDeactivationToken(tokenString string, cfg config.Config) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	// Verificar que el token es de tipo desactivación
	if claims["type"] != "deactivation" {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// renderDeactivationPage renderiza la página HTML de confirmación
func renderDeactivationPage(w http.ResponseWriter, success bool, message string) {
	tmpl := `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{if .Success}}Baja confirmada{{else}}Error en la baja{{end}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background-color: #ffffff;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            max-width: 500px;
            width: 100%;
            padding: 40px;
            text-align: center;
        }
        .icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 30px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 40px;
        }
        .icon.success {
            background-color: #d4edda;
            color: #155724;
        }
        .icon.error {
            background-color: #f8d7da;
            color: #721c24;
        }
        h1 {
            color: #2c3e50;
            margin-bottom: 20px;
            font-size: 28px;
        }
        .message {
            color: #555;
            font-size: 16px;
            line-height: 1.6;
            margin-bottom: 30px;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            font-size: 14px;
            color: #777;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon {{if .Success}}success{{else}}error{{end}}">
            {{if .Success}}✓{{else}}✗{{end}}
        </div>
        <h1>{{if .Success}}Baja confirmada{{else}}Error en la baja{{end}}</h1>
        <p class="message">{{.Message}}</p>
        <div class="footer">
            <p>Este es un mensaje automático del sistema.</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("deactivation").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	data := map[string]interface{}{
		"Success": success,
		"Message": message,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("❌ Error ejecutando template: %v", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
	}
}



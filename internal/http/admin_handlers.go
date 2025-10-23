package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opo_admin_server/internal/config"
	"opo_admin_server/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// AdminUserGet - Obtener información del usuario administrador
func AdminUserGet(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		users := client.Database(cfg.DBName).Collection("user")

		// Obtener ID del usuario del contexto
		userID := r.Context().Value("user_id")
		if userID == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
			return
		}

		var user domain.User
		if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

// AdminUserUpdate - Actualizar información del usuario administrador
func AdminUserUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			AppID string `json:"appId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.Name = strings.TrimSpace(req.Name)

		if req.Name == "" || req.Email == "" || (req.AppID != "1" && req.AppID != "2") {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name, email y appId (1 o 2) requeridos")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		users := client.Database(cfg.DBName).Collection("user")

		update := bson.M{
			"$set": bson.M{
				"name":      req.Name,
				"email":     req.Email,
				"appId":     req.AppID,
				"updatedAt": time.Now(),
			},
		}

		var user domain.User
		if err := users.FindOneAndUpdate(ctx, bson.M{}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

// AdminUserResetPassword - Cambiar contraseña del usuario administrador
func AdminUserResetPassword(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		if req.CurrentPassword == "" || req.NewPassword == "" || len(req.NewPassword) < 6 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "currentPassword y newPassword (>=6) requeridos")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		users := client.Database(cfg.DBName).Collection("user")

		var user domain.User
		if err := users.FindOne(ctx, bson.M{}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)) != nil {
			writeError(w, http.StatusBadRequest, "invalid_password", "contraseña actual incorrecta")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "no se pudo procesar la nueva contraseña")
			return
		}

		users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
			"$set": bson.M{
				"password":  string(hash),
				"updatedAt": time.Now(),
			},
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Contraseña actualizada exitosamente",
		})
	}
}

// AdminTopicsList - Listar topics (solo temas principales filtrados por área del usuario)
func AdminTopicsList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsear parámetros de consulta
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}
		areaParam := r.URL.Query().Get("area")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		// Determinar el área a filtrar
		var filterArea int
		if areaParam != "" {
			// Si viene el parámetro area, usarlo
			filterArea, err = strconv.Atoi(areaParam)
			if err != nil {
				log.Printf("❌ AdminTopicsList - Error convirtiendo área: %v", err)
				writeError(w, http.StatusBadRequest, "invalid_area", "área debe ser un número válido")
				return
			}
			log.Printf("🔍 AdminTopicsList - Parámetro area recibido (string): '%s'", areaParam)
			log.Printf("🔍 AdminTopicsList - Usando área del parámetro (int): %d", filterArea)
		} else {
			// Si no viene area, usar el área del usuario (appId)
			userID := r.Context().Value("user_id")
			if userID == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
				return
			}

			users := client.Database(cfg.DBName).Collection("user")
			var user domain.User
			if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo usuario: "+err.Error())
				return
			}

			if user.AppID == "1" {
				filterArea = 1
			} else if user.AppID == "2" {
				filterArea = 2
			} else {
				writeError(w, http.StatusForbidden, "forbidden", "usuario sin área asignada válida")
				return
			}
			log.Printf("🔍 AdminTopicsList - Usando área del usuario (appId): %s -> %d", user.AppID, filterArea)
		}

		// Construir filtro: solo temas principales donde id === rootId Y del área especificada
		// Usando $expr para comparar campos dentro del mismo documento
		filter := bson.M{
			"area": filterArea,
			"$expr": bson.M{
				"$eq": []interface{}{"$id", "$rootId"},
			},
		}

		// Agregar filtro de enabled si viene en los parámetros
		enabledParam := r.URL.Query().Get("enabled")
		if enabledParam != "" {
			enabled := enabledParam == "true"
			filter["enabled"] = enabled
			log.Printf("🔍 AdminTopicsList - Aplicando filtro enabled: %v", enabled)
		}

		// Agregar filtro de premium si viene en los parámetros
		premiumParam := r.URL.Query().Get("premium")
		if premiumParam != "" {
			premium := premiumParam == "true"
			filter["premium"] = premium
			log.Printf("🔍 AdminTopicsList - Aplicando filtro premium: %v", premium)
		}

		// Agregar filtro de type si viene en los parámetros
		typeParam := r.URL.Query().Get("type")
		if typeParam != "" {
			// Validar que el type sea válido
			if typeParam == "topic" || typeParam == "exam" || typeParam == "misc" {
				filter["type"] = typeParam
				log.Printf("🔍 AdminTopicsList - Aplicando filtro type: %s", typeParam)
			} else {
				log.Printf("⚠️ AdminTopicsList - Type inválido ignorado: %s", typeParam)
			}
		}

		// Agregar filtro de búsqueda si viene en los parámetros
		searchParam := r.URL.Query().Get("search")
		if searchParam != "" {
			filter["title"] = bson.M{"$regex": searchParam, "$options": "i"}
			log.Printf("🔍 AdminTopicsList - Aplicando filtro search: %s", searchParam)
		}

		log.Printf("🔍 AdminTopicsList - Filtro MongoDB final: %+v", filter)

		// Contar total de temas principales
		total, err := col.CountDocuments(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminTopicsList - Total de topics encontrados con filtro: %d", total)

		// Opciones de paginación con ordenamiento por order
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "order", Value: 1}})

		// Obtener temas principales
		cur, err := col.Find(ctx, filter, opts)
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

		log.Printf("🔍 AdminTopicsList - Topics recuperados: %d", len(topics))
		if len(topics) > 0 {
			log.Printf("🔍 AdminTopicsList - Primer topic - ID: %d, Área: %d, Premium: %v, Title: %s", topics[0].TopicID, topics[0].Area, topics[0].Premium, topics[0].Title)
		}

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: topics,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminTopicsGetByID - Obtener topic específico
func AdminTopicsGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		var topic domain.Topic
		if err := col.FindOne(ctx, bson.M{"id": id}).Decode(&topic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, topic)
	}
}

// AdminTopicsGetSubtopics - Obtener subtemas de un tema principal
func AdminTopicsGetSubtopics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
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

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		// Obtener el tema principal
		var parentTopic domain.Topic
		if err := col.FindOne(ctx, bson.M{"id": id}).Decode(&parentTopic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "tema principal no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Obtener subtemas
		filter := bson.M{
			"rootId": id,
			"id":     bson.M{"$ne": id}, // id !== rootId
		}

		cur, err := col.Find(ctx, filter, options.Find())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var subtopics []domain.Topic
		if err := cur.All(ctx, &subtopics); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		response := map[string]interface{}{
			"subtopics": subtopics,
			"parentTopic": map[string]interface{}{
				"_id":   parentTopic.ID,
				"id":    parentTopic.TopicID,
				"uuid":  parentTopic.UUID,
				"title": parentTopic.Title,
			},
			"total": len(subtopics),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminTopicsCreate - Crear nuevo topic
func AdminTopicsCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.Topic

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones básicas
		if req.TopicID == 0 || req.UUID == "" || req.Title == "" || req.Area == 0 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "id, uuid, title y area requeridos")
			return
		}

		if req.Area != 1 && req.Area != 2 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 1 o 2")
			return
		}

		// Validar tipo si se proporciona, sino establecer valor por defecto
		if req.Type == "" {
			req.Type = "topic" // Valor por defecto
		} else {
			if req.Type != "topic" && req.Type != "exam" && req.Type != "misc" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "type debe ser 'topic', 'exam' o 'misc'")
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		// Verificar que no existe
		var existing domain.Topic
		if err := col.FindOne(ctx, bson.M{"id": req.TopicID}).Decode(&existing); err == nil {
			writeError(w, http.StatusConflict, "topic_exists", "topic con este ID ya existe")
			return
		}

		// Si es un tema principal, rootId = id
		if req.RootID == 0 {
			req.RootID = req.TopicID
		}
		if req.RootUUID == "" {
			req.RootUUID = req.UUID
		}

		now := time.Now()
		req.ID = uuid.NewString()
		req.CreatedAt = now
		req.UpdatedAt = now

		if _, err := col.InsertOne(ctx, req); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminTopicsCreate - Topic %d creado con type: %s", req.TopicID, req.Type)
		writeJSON(w, http.StatusCreated, req)
	}
}

// AdminTopicsUpdate - Actualizar topic
func AdminTopicsUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		var req domain.Topic

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validar área si se proporciona
		if req.Area != 0 {
			if req.Area != 1 && req.Area != 2 {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 1 o 2")
				return
			}
		}

		// Validar tipo si se proporciona
		if req.Type != "" {
			if req.Type != "topic" && req.Type != "exam" && req.Type != "misc" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "type debe ser 'topic', 'exam' o 'misc'")
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		update := bson.M{
			"$set": bson.M{
				"title":       req.Title,
				"description": req.Description,
				"imageUrl":    req.ImageURL,
				"enabled":     req.Enabled,
				"order":       req.Order,
				"updatedAt":   time.Now(),
			},
		}

		// Agregar área al update si se proporciona
		if req.Area != 0 {
			update["$set"].(bson.M)["area"] = req.Area
			log.Printf("🔄 AdminTopicsUpdate - Actualizando área del topic %d a %d", id, req.Area)
		}

		// Agregar tipo al update si se proporciona
		if req.Type != "" {
			update["$set"].(bson.M)["type"] = req.Type
			log.Printf("🔄 AdminTopicsUpdate - Actualizando type del topic %d a %s", id, req.Type)
		}

		var topic domain.Topic
		if err := col.FindOneAndUpdate(ctx, bson.M{"id": id}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&topic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Si se cambió el área o el tipo y es un tema principal, actualizar todos los subtopics
		if (req.Area != 0 || req.Type != "") && topic.IsMainTopic() {
			log.Printf("🔍 AdminTopicsUpdate - Es un tema principal, buscando subtopics con rootId=%d", id)

			// Buscar todos los subtopics (donde rootId == id del tema principal y id != rootId)
			subtopicsFilter := bson.M{
				"rootId": id,
				"id":     bson.M{"$ne": id}, // id !== rootId (son subtopics)
			}

			// Contar cuántos subtopics hay
			subtopicsCount, err := col.CountDocuments(ctx, subtopicsFilter)
			if err != nil {
				log.Printf("⚠️ AdminTopicsUpdate - Error contando subtopics: %v", err)
			} else {
				log.Printf("📊 AdminTopicsUpdate - Encontrados %d subtopics para actualizar", subtopicsCount)

				if subtopicsCount > 0 {
					// Preparar actualización de subtopics
					subtopicsUpdateFields := bson.M{
						"updatedAt": time.Now(),
					}

					// Agregar área si se cambió
					if req.Area != 0 {
						subtopicsUpdateFields["area"] = req.Area
					}

					// Agregar tipo si se cambió
					if req.Type != "" {
						subtopicsUpdateFields["type"] = req.Type
					}

					subtopicsUpdate := bson.M{
						"$set": subtopicsUpdateFields,
					}

					updateResult, err := col.UpdateMany(ctx, subtopicsFilter, subtopicsUpdate)
					if err != nil {
						log.Printf("❌ AdminTopicsUpdate - Error actualizando subtopics: %v", err)
						// No devolvemos error porque el topic principal sí se actualizó
					} else {
						if req.Area != 0 && req.Type != "" {
							log.Printf("✅ AdminTopicsUpdate - %d subtopics actualizados (área: %d, type: %s)", updateResult.ModifiedCount, req.Area, req.Type)
						} else if req.Area != 0 {
							log.Printf("✅ AdminTopicsUpdate - %d subtopics actualizados al área %d", updateResult.ModifiedCount, req.Area)
						} else if req.Type != "" {
							log.Printf("✅ AdminTopicsUpdate - %d subtopics actualizados al type %s", updateResult.ModifiedCount, req.Type)
						}
					}
				}
			}
		}

		log.Printf("✅ AdminTopicsUpdate - Topic %d actualizado exitosamente", id)
		writeJSON(w, http.StatusOK, topic)
	}
}

// AdminTopicsToggleEnabled - Cambiar estado enabled/disabled
func AdminTopicsToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"_id":     idStr,
			"enabled": req.Enabled,
			"message": "Estado del topic actualizado exitosamente",
		})
	}
}

// AdminTopicsTogglePremium - Cambiar estado premium/no-premium
func AdminTopicsTogglePremium(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		log.Printf("🔍 Toggle Premium - ID string recibido: %s", idStr)

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Printf("❌ Toggle Premium - Error al convertir ID: %v", err)
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}
		log.Printf("🔍 Toggle Premium - ID convertido a int: %d", id)

		var req struct {
			Premium bool `json:"premium"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("❌ Toggle Premium - Error al decodificar JSON: %v", err)
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}
		log.Printf("🔍 Toggle Premium - Valor premium recibido: %v", req.Premium)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		update := bson.M{
			"$set": bson.M{
				"premium":   req.Premium,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"_id":     idStr,
			"premium": req.Premium,
			"message": "Estado premium del topic actualizado exitosamente",
		})
	}
}

// AdminTopicsDelete - Eliminar topic
func AdminTopicsDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		result, err := col.DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "Topic eliminado exitosamente",
			"deletedId": idStr,
		})
	}
}

// AdminStatsUser - Estadísticas del usuario administrador
func AdminStatsUser(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Obtener usuario
		users := client.Database(cfg.DBName).Collection("user")
		var user domain.User
		if err := users.FindOne(ctx, bson.M{}).Decode(&user); err != nil {
			writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
			return
		}

		// Obtener estadísticas de topics
		topics := client.Database(cfg.DBName).Collection("topics_uuid_map")

		totalTopics, _ := topics.CountDocuments(ctx, bson.M{})
		enabledTopics, _ := topics.CountDocuments(ctx, bson.M{"enabled": true})
		disabledTopics := totalTopics - enabledTopics

		response := domain.UserStats{
			User: user,
			SystemInfo: domain.SystemInfo{
				TotalTopics:    int(totalTopics),
				EnabledTopics:  int(enabledTopics),
				DisabledTopics: int(disabledTopics),
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminStatsArea - Estadísticas de un área específica
func AdminStatsArea(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaIdStr := chi.URLParam(r, "areaId")

		areaId, err := strconv.Atoi(areaIdStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_area", "área debe ser un número válido")
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

		// Obtener información del área
		areasCol := client.Database(cfg.DBName).Collection("apps")
		var area domain.App
		if err := areasCol.FindOne(ctx, bson.M{"id": areaIdStr}).Decode(&area); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Estadísticas de topics
		topicsCol := client.Database(cfg.DBName).Collection("topics_uuid_map")

		totalTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId})

		// Contar main topics (id === rootId)
		mainTopicsFilter := bson.M{
			"area": areaId,
			"$expr": bson.M{
				"$eq": []interface{}{"$id", "$rootId"},
			},
		}
		mainTopics, _ := topicsCol.CountDocuments(ctx, mainTopicsFilter)

		// Contar subtopics (id !== rootId)
		subtopicsFilter := bson.M{
			"area": areaId,
			"$expr": bson.M{
				"$ne": []interface{}{"$id", "$rootId"},
			},
		}
		subtopics, _ := topicsCol.CountDocuments(ctx, subtopicsFilter)

		// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
		enabledTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
		disabledTopics := totalTopics - enabledTopics

		// Estadísticas de usuarios
		usersCol := client.Database(cfg.DBName).Collection("users")

		totalUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId})
		// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
		enabledUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
		disabledUsers := totalUsers - enabledUsers

		response := map[string]interface{}{
			"areaId":   areaId,
			"areaName": area.Name,
			"topics": map[string]interface{}{
				"total":      int(totalTopics),
				"mainTopics": int(mainTopics),
				"subtopics":  int(subtopics),
				"enabled":    int(enabledTopics),
				"disabled":   int(disabledTopics),
			},
			"users": map[string]interface{}{
				"total":    int(totalUsers),
				"enabled":  int(enabledUsers),
				"disabled": int(disabledUsers),
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminStatsAllAreas - Estadísticas de todas las áreas
func AdminStatsAllAreas(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Obtener todas las áreas
		areasCol := client.Database(cfg.DBName).Collection("apps")
		cur, err := areasCol.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var areas []domain.App
		if err := cur.All(ctx, &areas); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Obtener estadísticas para cada área
		var allStats []map[string]interface{}

		topicsCol := client.Database(cfg.DBName).Collection("topics_uuid_map")
		usersCol := client.Database(cfg.DBName).Collection("users")

		for _, area := range areas {
			areaId, _ := strconv.Atoi(area.ID)

			// Estadísticas de topics
			totalTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId})

			mainTopicsFilter := bson.M{
				"area": areaId,
				"$expr": bson.M{
					"$eq": []interface{}{"$id", "$rootId"},
				},
			}
			mainTopics, _ := topicsCol.CountDocuments(ctx, mainTopicsFilter)

			subtopicsFilter := bson.M{
				"area": areaId,
				"$expr": bson.M{
					"$ne": []interface{}{"$id", "$rootId"},
				},
			}
			subtopics, _ := topicsCol.CountDocuments(ctx, subtopicsFilter)

			// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
			enabledTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
			disabledTopics := totalTopics - enabledTopics

			// Estadísticas de usuarios
			totalUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId})
			// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
			enabledUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
			disabledUsers := totalUsers - enabledUsers

			areaStats := map[string]interface{}{
				"areaId":   areaId,
				"areaName": area.Name,
				"topics": map[string]interface{}{
					"total":      int(totalTopics),
					"mainTopics": int(mainTopics),
					"subtopics":  int(subtopics),
					"enabled":    int(enabledTopics),
					"disabled":   int(disabledTopics),
				},
				"users": map[string]interface{}{
					"total":    int(totalUsers),
					"enabled":  int(enabledUsers),
					"disabled": int(disabledUsers),
				},
			}

			allStats = append(allStats, areaStats)
		}

		writeJSON(w, http.StatusOK, allStats)
	}
}

// AdminStatsTopics - Estadísticas de topics
func AdminStatsTopics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		topics := client.Database(cfg.DBName).Collection("topics_uuid_map")

		totalTopics, _ := topics.CountDocuments(ctx, bson.M{})
		enabledTopics, _ := topics.CountDocuments(ctx, bson.M{"enabled": true})
		disabledTopics := totalTopics - enabledTopics

		// Topics por área
		topicsByArea := make(map[string]int)
		pnTopics, _ := topics.CountDocuments(ctx, bson.M{"area": 1})
		psTopics, _ := topics.CountDocuments(ctx, bson.M{"area": 2})
		topicsByArea["PN"] = int(pnTopics)
		topicsByArea["PS"] = int(psTopics)

		response := domain.TopicStats{
			TotalTopics:    int(totalTopics),
			TopicsByArea:   topicsByArea,
			EnabledTopics:  int(enabledTopics),
			DisabledTopics: int(disabledTopics),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// ========== Handlers de Áreas (Apps) ==========

// AdminAreasList - Listar áreas con paginación
func AdminAreasList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsear parámetros de consulta
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		// Contar total de áreas
		total, err := col.CountDocuments(ctx, bson.M{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Opciones de paginación con ordenamiento por order
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "order", Value: 1}})

		// Obtener áreas
		cur, err := col.Find(ctx, bson.M{}, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var areas []domain.App
		if err := cur.All(ctx, &areas); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: areas,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminAreasGetByID - Obtener área por ID
func AdminAreasGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		var area domain.App
		// Buscar por el campo id en lugar de _id
		if err := col.FindOne(ctx, bson.M{"id": id}).Decode(&area); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, area)
	}
}

// AdminAreasCreate - Crear nueva área
func AdminAreasCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.App

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones básicas
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" || len(req.Name) < 3 || len(req.Name) > 100 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		// Generar IDs únicos
		now := time.Now()
		req.MongoID = uuid.NewString()

		// Generar ID numérico secuencial
		count, _ := col.CountDocuments(ctx, bson.M{})
		req.ID = strconv.FormatInt(count+1, 10)

		req.CreatedAt = now
		req.UpdatedAt = now

		if _, err := col.InsertOne(ctx, req); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, req)
	}
}

// AdminAreasUpdate - Actualizar área
func AdminAreasUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req domain.App

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones básicas
		req.Name = strings.TrimSpace(req.Name)
		if req.Name != "" && (len(req.Name) < 3 || len(req.Name) > 100) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		update := bson.M{
			"$set": bson.M{
				"name":        req.Name,
				"description": req.Description,
				"order":       req.Order,
				"updatedAt":   time.Now(),
			},
		}

		var area domain.App
		// Buscar por el campo id en lugar de _id
		if err := col.FindOneAndUpdate(ctx, bson.M{"id": id}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&area); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, area)
	}
}

// AdminAreasToggleEnabled - Cambiar estado enabled/disabled
func AdminAreasToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		// Buscar por el campo id en lugar de _id
		result, err := col.UpdateOne(ctx, bson.M{"id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"enabled": req.Enabled,
			"message": "Estado del área actualizado exitosamente",
		})
	}
}

// AdminAreasDelete - Eliminar área
func AdminAreasDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		// Buscar por el campo id en lugar de _id
		result, err := col.DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "Área eliminada exitosamente",
			"deletedId": id,
		})
	}
}

// ========== Handlers de Usuarios ==========

// AdminUsersList - Listar usuarios filtrados por área
func AdminUsersList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsear parámetros de consulta
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}
		areaParam := r.URL.Query().Get("area")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("users")

		// Determinar el área a filtrar
		var filterArea int
		if areaParam != "" {
			// Si viene el parámetro area, usarlo
			filterArea, err = strconv.Atoi(areaParam)
			if err != nil {
				log.Printf("❌ AdminUsersList - Error convirtiendo área: %v", err)
				writeError(w, http.StatusBadRequest, "invalid_area", "área debe ser un número válido")
				return
			}
			log.Printf("🔍 AdminUsersList - Usando área del parámetro: %d", filterArea)
		} else {
			// Si no viene area, usar el área del admin logueado
			userID := r.Context().Value("user_id")
			if userID == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
				return
			}

			users := client.Database(cfg.DBName).Collection("user")
			var adminUser domain.User
			if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&adminUser); err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo admin: "+err.Error())
				return
			}

			if adminUser.AppID == "1" {
				filterArea = 1
			} else if adminUser.AppID == "2" {
				filterArea = 2
			} else {
				writeError(w, http.StatusForbidden, "forbidden", "admin sin área asignada válida")
				return
			}
			log.Printf("🔍 AdminUsersList - Usando área del admin: %d", filterArea)
		}

		// Construir filtro por área
		filter := bson.M{
			"area": filterArea,
		}

		log.Printf("🔍 AdminUsersList - Filtro MongoDB: %+v", filter)

		// Contar total de usuarios
		total, err := col.CountDocuments(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminUsersList - Total usuarios encontrados: %d", total)

		// Opciones de paginación con ordenamiento por createdAt descendente
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "createdAt", Value: -1}})

		// Obtener usuarios
		cur, err := col.Find(ctx, filter, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var users []domain.User
		if err := cur.All(ctx, &users); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminUsersList - Usuarios recuperados: %d", len(users))

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: users,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminUsersToggleEnabled - Cambiar estado enabled/disabled de un usuario
func AdminUsersToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("users")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		// Buscar por el campo _id
		result, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "usuario no encontrado")
			return
		}

		log.Printf("✅ AdminUsersToggleEnabled - Usuario %s actualizado a enabled: %v", id, req.Enabled)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"enabled": req.Enabled,
			"message": "Estado del usuario actualizado exitosamente",
		})
	}
}

// ========== Handlers de Proveedores de Publicidad ==========

// AdminProvidersList - Listar proveedores con paginación
func AdminProvidersList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		filter := bson.M{}

		// Filtro por enabled si viene en parámetros
		enabledParam := r.URL.Query().Get("enabled")
		if enabledParam != "" {
			enabled := enabledParam == "true"
			filter["enabled"] = enabled
		}

		total, err := col.CountDocuments(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "order", Value: 1}})

		cur, err := col.Find(ctx, filter, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var providers []domain.AdProvider
		if err := cur.All(ctx, &providers); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: providers,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminProvidersGetByID - Obtener proveedor por ID
func AdminProvidersGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		var provider domain.AdProvider
		if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&provider); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, provider)
	}
}

// AdminProvidersCreate - Crear nuevo proveedor
func AdminProvidersCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.AdProvider

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.ProviderID = strings.TrimSpace(strings.ToLower(req.ProviderID))

		if req.Name == "" || req.ProviderID == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name y providerId requeridos")
			return
		}

		if len(req.Name) < 3 || len(req.Name) > 100 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		// Verificar que no existe un proveedor con el mismo providerId
		var existing domain.AdProvider
		if err := col.FindOne(ctx, bson.M{"providerId": req.ProviderID}).Decode(&existing); err == nil {
			writeError(w, http.StatusConflict, "provider_exists", "ya existe un proveedor con este ID")
			return
		}

		now := time.Now()
		req.ID = uuid.NewString()
		req.CreatedAt = now
		req.UpdatedAt = now

		if _, err := col.InsertOne(ctx, req); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminProvidersCreate - Proveedor %s creado", req.ProviderID)
		writeJSON(w, http.StatusCreated, req)
	}
}

// AdminProvidersUpdate - Actualizar proveedor
func AdminProvidersUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req domain.AdProvider

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Name = strings.TrimSpace(req.Name)

		if req.Name != "" && (len(req.Name) < 3 || len(req.Name) > 100) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		update := bson.M{
			"$set": bson.M{
				"name":      req.Name,
				"icon":      req.Icon,
				"color":     req.Color,
				"order":     req.Order,
				"updatedAt": time.Now(),
			},
		}

		var provider domain.AdProvider
		if err := col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&provider); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminProvidersUpdate - Proveedor %s actualizado", id)
		writeJSON(w, http.StatusOK, provider)
	}
}

// AdminProvidersToggleEnabled - Cambiar estado enabled/disabled
func AdminProvidersToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
			return
		}

		log.Printf("✅ AdminProvidersToggleEnabled - Proveedor %s actualizado a enabled: %v", id, req.Enabled)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"enabled": req.Enabled,
			"message": "Estado del proveedor actualizado exitosamente",
		})
	}
}

// AdminProvidersDelete - Eliminar proveedor
func AdminProvidersDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		result, err := col.DeleteOne(ctx, bson.M{"_id": id})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
			return
		}

		log.Printf("✅ AdminProvidersDelete - Proveedor %s eliminado", id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "Proveedor eliminado exitosamente",
			"deletedId": id,
		})
	}
}

// ========== Handlers de Base de Datos ==========

// AdminDatabaseStats - Obtener estadísticas de la base de datos
func AdminDatabaseStats(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		database := client.Database(cfg.DBName)

		// Obtener lista de colecciones
		collections, err := database.ListCollectionNames(ctx, bson.M{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo colecciones: "+err.Error())
			return
		}

		var collectionStats []domain.CollectionStats
		var totalDocuments int64
		var totalSize int64

		// Obtener estadísticas de cada colección
		for _, collectionName := range collections {
			collection := database.Collection(collectionName)

			// Contar documentos
			docCount, err := collection.CountDocuments(ctx, bson.M{})
			if err != nil {
				log.Printf("⚠️ Error contando documentos en colección %s: %v", collectionName, err)
				docCount = 0
			}

			// Obtener estadísticas de la colección
			var stats bson.M
			err = database.RunCommand(ctx, bson.M{"collStats": collectionName}).Decode(&stats)
			if err != nil {
				log.Printf("⚠️ Error obteniendo estadísticas de colección %s: %v", collectionName, err)
				stats = bson.M{"size": 0}
			}

			size := int64(0)
			if sizeVal, ok := stats["size"].(int32); ok {
				size = int64(sizeVal)
			} else if sizeVal, ok := stats["size"].(int64); ok {
				size = sizeVal
			}

			collectionStats = append(collectionStats, domain.CollectionStats{
				Name:          collectionName,
				DocumentCount: docCount,
				Size:          size,
			})

			totalDocuments += docCount
			totalSize += size
		}

		response := domain.DatabaseStats{
			DatabaseName:   cfg.DBName,
			TotalSize:      totalSize,
			Collections:    collectionStats,
			TotalDocuments: totalDocuments,
		}

		log.Printf("✅ AdminDatabaseStats - Estadísticas obtenidas: %d colecciones, %d documentos, %d bytes", len(collectionStats), totalDocuments, totalSize)
		writeJSON(w, http.StatusOK, response)
	}
}

// AdminDatabaseDownload - Descargar backup de la base de datos
func AdminDatabaseDownload(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		database := client.Database(cfg.DBName)

		// Obtener lista de colecciones
		collections, err := database.ListCollectionNames(ctx, bson.M{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo colecciones: "+err.Error())
			return
		}

		// Configurar headers para descarga
		backupFileName := fmt.Sprintf("mongodb_backup_%s_%s.json", cfg.DBName, time.Now().Format("2006-01-02_15-04-05"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", backupFileName))
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Escribir inicio del JSON con streaming
		w.Write([]byte("{\n"))
		w.Write([]byte(fmt.Sprintf("  \"database\": %q,\n", cfg.DBName)))
		w.Write([]byte(fmt.Sprintf("  \"exportedAt\": %q,\n", time.Now().Format(time.RFC3339))))
		w.Write([]byte("  \"collections\": {\n"))

		// Exportar cada colección con streaming
		for i, collectionName := range collections {
			collection := database.Collection(collectionName)

			// Escribir nombre de la colección
			w.Write([]byte(fmt.Sprintf("    %q: [\n", collectionName)))

			// Obtener documentos con cursor para evitar cargar todo en memoria
			cursor, err := collection.Find(ctx, bson.M{})
			if err != nil {
				log.Printf("⚠️ Error obteniendo documentos de colección %s: %v", collectionName, err)
				w.Write([]byte("    ]"))
				if i < len(collections)-1 {
					w.Write([]byte(","))
				}
				w.Write([]byte("\n"))
				continue
			}
			defer cursor.Close(ctx)

			// Procesar documentos uno por uno
			docCount := 0
			for cursor.Next(ctx) {
				var document bson.M
				if err := cursor.Decode(&document); err != nil {
					log.Printf("⚠️ Error decodificando documento en colección %s: %v", collectionName, err)
					continue
				}

				// Escribir documento
				if docCount > 0 {
					w.Write([]byte(","))
				}
				w.Write([]byte("\n      "))

				// Convertir documento a JSON
				docJSON, err := json.Marshal(document)
				if err != nil {
					log.Printf("⚠️ Error serializando documento en colección %s: %v", collectionName, err)
					continue
				}
				w.Write(docJSON)
				docCount++

				// Flush para enviar datos inmediatamente
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}

			// Cerrar array de documentos
			if docCount > 0 {
				w.Write([]byte("\n    "))
			}
			w.Write([]byte("]"))
			if i < len(collections)-1 {
				w.Write([]byte(","))
			}
			w.Write([]byte("\n"))

			log.Printf("✅ Colección %s exportada: %d documentos", collectionName, docCount)
		}

		// Cerrar JSON
		w.Write([]byte("  }\n"))
		w.Write([]byte("}\n"))

		log.Printf("✅ AdminDatabaseDownload - Backup JSON enviado exitosamente: %s", backupFileName)
	}
}

// AdminGetAvailableSourceTopics - Obtener temas disponibles de otras áreas como fuente de preguntas
func AdminGetAvailableSourceTopics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topicIdStr := chi.URLParam(r, "id")
		topicId, err := strconv.Atoi(topicIdStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_topic_id", "topic ID debe ser un número")
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

		db := client.Database(cfg.DBName)
		topicsCol := db.Collection("topics_uuid_map")
		questionsUnitsCol := db.Collection("questions_units_uuid")

		// 1. Obtener el tema destino para conocer su área
		var destTopic domain.Topic
		if err := topicsCol.FindOne(ctx, bson.M{"id": topicId}).Decode(&destTopic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "topic_not_found", "tema destino no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// 2. Buscar temas principales de OTRAS áreas (excluyendo la del tema destino)
		filter := bson.M{
			"area":    bson.M{"$ne": destTopic.Area},                                                  // Excluir el área del tema destino
			"id":      bson.M{"$eq": bson.M{"$expr": bson.M{"$eq": []interface{}{"$id", "$rootId"}}}}, // Solo temas principales
			"enabled": true,                                                                           // Solo temas habilitados
		}

		cur, err := topicsCol.Find(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var mainTopics []domain.Topic
		if err := cur.All(ctx, &mainTopics); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// 3. Para cada tema principal, contar subtemas y preguntas
		var sourceTopics []domain.SourceTopicInfo
		for _, topic := range mainTopics {
			// Contar subtemas
			subtopicFilter := bson.M{
				"rootUuid": topic.UUID,
				"enabled":  true,
			}
			subtopicCount, err := topicsCol.CountDocuments(ctx, subtopicFilter)
			if err != nil {
				log.Printf("Error contando subtemas para topic %s: %v", topic.UUID, err)
				subtopicCount = 0
			}

			// Obtener todos los UUIDs del tema (principal + subtemas)
			var allUuids []string
			allUuids = append(allUuids, topic.UUID) // Añadir tema principal

			// Añadir subtemas
			subtopicCur, err := topicsCol.Find(ctx, subtopicFilter)
			if err == nil {
				var subtopics []domain.Topic
				if err := subtopicCur.All(ctx, &subtopics); err == nil {
					for _, subtopic := range subtopics {
						allUuids = append(allUuids, subtopic.UUID)
					}
				}
				subtopicCur.Close(ctx)
			}

			// Contar preguntas totales (principal + subtemas)
			questionCount := int64(0)
			if len(allUuids) > 0 {
				questionFilter := bson.M{"topicUuid": bson.M{"$in": allUuids}}
				questionCount, err = questionsUnitsCol.CountDocuments(ctx, questionFilter)
				if err != nil {
					log.Printf("Error contando preguntas para topic %s: %v", topic.UUID, err)
					questionCount = 0
				}
			}

			sourceTopic := domain.SourceTopicInfo{
				TopicID:       topic.TopicID,
				UUID:          topic.UUID,
				Title:         topic.Title,
				Area:          topic.Area,
				IsMain:        true,
				SubtopicCount: int(subtopicCount),
				QuestionCount: int(questionCount),
			}

			sourceTopics = append(sourceTopics, sourceTopic)
		}

		writeJSON(w, http.StatusOK, sourceTopics)
	}
}

// AdminCopyQuestionsFromTopics - Copiar preguntas desde temas origen al tema destino
func AdminCopyQuestionsFromTopics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topicIdStr := chi.URLParam(r, "id")
		topicId, err := strconv.Atoi(topicIdStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_topic_id", "topic ID debe ser un número")
			return
		}

		var req domain.CopyQuestionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "JSON inválido")
			return
		}

		if len(req.SourceTopicUuids) == 0 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "debe seleccionar al menos un tema origen")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		db := client.Database(cfg.DBName)
		topicsCol := db.Collection("topics_uuid_map")
		questionsUnitsCol := db.Collection("questions_units_uuid")
		questionsCol := db.Collection("questions")

		// 1. Validar que el tema destino existe y obtener su información
		var destTopic domain.Topic
		if err := topicsCol.FindOne(ctx, bson.M{"id": topicId}).Decode(&destTopic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "topic_not_found", "tema destino no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// 2. Validar que los temas origen existen y son principales
		var sourceTopics []domain.Topic
		for _, uuid := range req.SourceTopicUuids {
			var topic domain.Topic
			if err := topicsCol.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&topic); err != nil {
				if err == mongo.ErrNoDocuments {
					writeError(w, http.StatusUnprocessableEntity, "source_topic_not_found", fmt.Sprintf("tema origen con UUID %s no encontrado", uuid))
					return
				}
				writeError(w, http.StatusInternalServerError, "server_error", err.Error())
				return
			}

			// Verificar que es tema principal
			if !topic.IsMainTopic() {
				writeError(w, http.StatusUnprocessableEntity, "not_main_topic", fmt.Sprintf("el tema %s no es un tema principal", topic.Title))
				return
			}

			// Verificar que no es del mismo área
			if topic.Area == destTopic.Area {
				writeError(w, http.StatusUnprocessableEntity, "same_area", fmt.Sprintf("no se puede copiar desde el mismo área (%d)", topic.Area))
				return
			}

			sourceTopics = append(sourceTopics, topic)
		}

		// 3. Recopilar todos los UUIDs de temas origen (principal + subtemas)
		var allSourceUuids []string
		for _, topic := range sourceTopics {
			allSourceUuids = append(allSourceUuids, topic.UUID)

			// Añadir subtemas
			subtopicFilter := bson.M{
				"rootUuid": topic.UUID,
				"enabled":  true,
			}
			subtopicCur, err := topicsCol.Find(ctx, subtopicFilter)
			if err == nil {
				var subtopics []domain.Topic
				if err := subtopicCur.All(ctx, &subtopics); err == nil {
					for _, subtopic := range subtopics {
						allSourceUuids = append(allSourceUuids, subtopic.UUID)
					}
				}
				subtopicCur.Close(ctx)
			}
		}

		// 4. Obtener todas las preguntas de los temas origen
		questionUnitsFilter := bson.M{"topicUuid": bson.M{"$in": allSourceUuids}}
		questionUnitsCur, err := questionsUnitsCol.Find(ctx, questionUnitsFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer questionUnitsCur.Close(ctx)

		type QuestionUnit struct {
			TopicID    int    `bson:"topicId"`
			TopicUuid  string `bson:"topicUuid"`
			QuestionId int    `bson:"questionId"`
			Area       int    `bson:"area"`
		}

		var questionUnits []QuestionUnit
		if err := questionUnitsCur.All(ctx, &questionUnits); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if len(questionUnits) == 0 {
			writeError(w, http.StatusUnprocessableEntity, "no_questions", "no hay preguntas disponibles en los temas seleccionados")
			return
		}

		// 5. Verificar que las preguntas existen en la colección questions
		var questionIds []int
		for _, unit := range questionUnits {
			questionIds = append(questionIds, unit.QuestionId)
		}

		// Eliminar duplicados
		questionIdMap := make(map[int]bool)
		var uniqueQuestionIds []int
		for _, id := range questionIds {
			if !questionIdMap[id] {
				questionIdMap[id] = true
				uniqueQuestionIds = append(uniqueQuestionIds, id)
			}
		}

		// Verificar existencia en questions
		questionsFilter := bson.M{"questionId": bson.M{"$in": uniqueQuestionIds}}
		questionsCount, err := questionsCol.CountDocuments(ctx, questionsFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if questionsCount == 0 {
			writeError(w, http.StatusUnprocessableEntity, "questions_not_found", "las preguntas no existen en la base de datos")
			return
		}

		// 6. Verificar qué preguntas ya existen en el tema destino
		existingFilter := bson.M{
			"topicUuid":  destTopic.UUID,
			"questionId": bson.M{"$in": uniqueQuestionIds},
		}
		existingCur, err := questionsUnitsCol.Find(ctx, existingFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer existingCur.Close(ctx)

		var existingUnits []QuestionUnit
		if err := existingCur.All(ctx, &existingUnits); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Crear mapa de questionIds existentes
		existingQuestionIds := make(map[int]bool)
		for _, unit := range existingUnits {
			existingQuestionIds[unit.QuestionId] = true
		}

		// 7. Preparar documentos para insertar (solo los que no existen)
		var documentsToInsert []interface{}
		questionsCopied := 0

		for _, questionId := range uniqueQuestionIds {
			if !existingQuestionIds[questionId] {
				doc := bson.M{
					"topicId":    destTopic.TopicID,
					"topicUuid":  destTopic.UUID,
					"questionId": questionId,
					"area":       destTopic.Area,
				}
				documentsToInsert = append(documentsToInsert, doc)
				questionsCopied++
			}
		}

		// 8. Insertar documentos usando bulkWrite
		if len(documentsToInsert) > 0 {
			var operations []mongo.WriteModel
			for _, doc := range documentsToInsert {
				operation := mongo.NewInsertOneModel().SetDocument(doc)
				operations = append(operations, operation)
			}

			_, err = questionsUnitsCol.BulkWrite(ctx, operations)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", err.Error())
				return
			}
		}

		// 9. Preparar respuesta
		response := domain.CopyQuestionsResponse{
			Message:         fmt.Sprintf("Se copiaron %d preguntas desde %d temas", questionsCopied, len(sourceTopics)),
			QuestionsCopied: questionsCopied,
			TopicsProcessed: len(sourceTopics),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

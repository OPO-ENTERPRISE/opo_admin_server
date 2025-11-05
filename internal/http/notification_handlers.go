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
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ========== Handlers de Notificaciones ==========

// AdminNotificationsList - Listar notificaciones con paginación
func AdminNotificationsList(cfg config.Config) http.HandlerFunc {
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

		col := client.Database(cfg.DBName).Collection("notifications")

		// Contar total de notificaciones
		total, err := col.CountDocuments(ctx, bson.M{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Opciones de paginación con ordenamiento por fecha de creación descendente
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "createdAt", Value: -1}})

		// Obtener notificaciones
		cur, err := col.Find(ctx, bson.M{}, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var notifications []domain.Notification
		if err := cur.All(ctx, &notifications); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: notifications,
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

// AdminNotificationsGetByID - Obtener notificación por ID
func AdminNotificationsGetByID(cfg config.Config) http.HandlerFunc {
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

		col := client.Database(cfg.DBName).Collection("notifications")

		var notification domain.Notification
		if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&notification); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "notificación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, notification)
	}
}

// AdminNotificationsCreate - Crear nueva notificación
func AdminNotificationsCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.CreateNotificationRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones
		req.Title = strings.TrimSpace(req.Title)
		req.Message = strings.TrimSpace(req.Message)

		if req.Title == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "title es requerido")
			return
		}
		if req.Message == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "message es requerido")
			return
		}
		if req.Type != "fixed" && req.Type != "simple" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "type debe ser 'fixed' o 'simple'")
			return
		}
		if req.Area != 0 && req.Area != 1 && req.Area != 2 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 0 (todas), 1 (PN) o 2 (PS)")
			return
		}

		// Si es tipo "fixed", validar que tenga actionType
		if req.Type == "fixed" && req.ActionType == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "actionType es requerido para notificaciones tipo 'fixed'")
			return
		}

		// Validar actionType válidos
		if req.ActionType != "" && req.ActionType != "update_app" && req.ActionType != "link" && req.ActionType != "acknowledge" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "actionType debe ser 'update_app', 'link' o 'acknowledge'")
			return
		}

		// Si actionType es "link", validar que actionData sea una URL válida
		if req.ActionType == "link" && req.ActionData == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "actionData (URL) es requerido cuando actionType es 'link'")
			return
		}

		// Si actionType es "update_app", validar que actionData tenga versión
		if req.ActionType == "update_app" && req.ActionData == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "actionData (versión mínima) es requerido cuando actionType es 'update_app'")
			return
		}

		// Obtener ID del usuario del contexto
		userID := r.Context().Value("user_id")
		if userID == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
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

		col := client.Database(cfg.DBName).Collection("notifications")

		// Crear notificación
		now := time.Now()
		notification := domain.Notification{
			ID:          uuid.NewString(),
			Title:       req.Title,
			Message:     req.Message,
			Type:        req.Type,
			Area:        req.Area,
			ActionType:  req.ActionType,
			ActionData:  req.ActionData,
			StartDate:   req.StartDate,
			EndDate:     req.EndDate,
			Enabled:     req.Enabled,
			CreatedBy:   userID.(string),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if _, err := col.InsertOne(ctx, notification); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminNotificationsCreate - Notificación creada: ID=%s, Title=%s, Area=%d", notification.ID, notification.Title, notification.Area)

		writeJSON(w, http.StatusCreated, notification)
	}
}

// AdminNotificationsUpdate - Actualizar notificación
func AdminNotificationsUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req domain.UpdateNotificationRequest
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

		col := client.Database(cfg.DBName).Collection("notifications")

		// Verificar que la notificación existe
		var existing domain.Notification
		if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&existing); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "notificación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Construir update
		update := bson.M{
			"$set": bson.M{
				"updatedAt": time.Now(),
			},
		}

		if req.Title != nil {
			title := strings.TrimSpace(*req.Title)
			if title == "" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "title no puede estar vacío")
				return
			}
			update["$set"].(bson.M)["title"] = title
		}
		if req.Message != nil {
			message := strings.TrimSpace(*req.Message)
			if message == "" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "message no puede estar vacío")
				return
			}
			update["$set"].(bson.M)["message"] = message
		}
		if req.Type != nil {
			if *req.Type != "fixed" && *req.Type != "simple" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "type debe ser 'fixed' o 'simple'")
				return
			}
			update["$set"].(bson.M)["type"] = *req.Type
		}
		if req.Area != nil {
			if *req.Area != 0 && *req.Area != 1 && *req.Area != 2 {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 0 (todas), 1 (PN) o 2 (PS)")
				return
			}
			update["$set"].(bson.M)["area"] = *req.Area
		}
		if req.ActionType != nil {
			if *req.ActionType != "" && *req.ActionType != "update_app" && *req.ActionType != "link" && *req.ActionType != "acknowledge" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "actionType debe ser 'update_app', 'link' o 'acknowledge'")
				return
			}
			update["$set"].(bson.M)["actionType"] = *req.ActionType
		}
		if req.ActionData != nil {
			update["$set"].(bson.M)["actionData"] = *req.ActionData
		}
		if req.StartDate != nil {
			update["$set"].(bson.M)["startDate"] = *req.StartDate
		}
		if req.EndDate != nil {
			update["$set"].(bson.M)["endDate"] = *req.EndDate
		}
		if req.Enabled != nil {
			update["$set"].(bson.M)["enabled"] = *req.Enabled
		}

		// Actualizar
		var updated domain.Notification
		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		if err := col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&updated); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminNotificationsUpdate - Notificación actualizada: ID=%s", id)

		writeJSON(w, http.StatusOK, updated)
	}
}

// AdminNotificationsDelete - Eliminar notificación
func AdminNotificationsDelete(cfg config.Config) http.HandlerFunc {
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

		col := client.Database(cfg.DBName).Collection("notifications")

		// Verificar que existe
		var existing domain.Notification
		if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&existing); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "notificación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Eliminar
		if _, err := col.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// También eliminar todos los registros de lectura asociados
		readsCol := client.Database(cfg.DBName).Collection("notification_reads")
		if _, err := readsCol.DeleteMany(ctx, bson.M{"notificationId": id}); err != nil {
			log.Printf("⚠️ AdminNotificationsDelete - Error eliminando registros de lectura: %v", err)
		}

		log.Printf("✅ AdminNotificationsDelete - Notificación eliminada: ID=%s", id)

		w.WriteHeader(http.StatusNoContent)
	}
}

// AdminNotificationsToggleEnabled - Activar/desactivar notificación
func AdminNotificationsToggleEnabled(cfg config.Config) http.HandlerFunc {
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

		col := client.Database(cfg.DBName).Collection("notifications")

		// Obtener notificación actual
		var notification domain.Notification
		if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&notification); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "notificación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Cambiar estado
		update := bson.M{
			"$set": bson.M{
				"enabled":   !notification.Enabled,
				"updatedAt": time.Now(),
			},
		}

		var updated domain.Notification
		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		if err := col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&updated); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminNotificationsToggleEnabled - Notificación %s: enabled=%v", id, updated.Enabled)

		writeJSON(w, http.StatusOK, updated)
	}
}

// AdminNotificationsStats - Obtener estadísticas de una notificación
func AdminNotificationsStats(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Verificar que la notificación existe
		notificationsCol := client.Database(cfg.DBName).Collection("notifications")
		var notification domain.Notification
		if err := notificationsCol.FindOne(ctx, bson.M{"_id": id}).Decode(&notification); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "notificación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		readsCol := client.Database(cfg.DBName).Collection("notification_reads")

		// Contar total de lecturas
		totalReads, _ := readsCol.CountDocuments(ctx, bson.M{"notificationId": id})

		// Contar acciones completadas (solo para tipo fixed)
		var totalActions int64
		if notification.Type == "fixed" {
			totalActions, _ = readsCol.CountDocuments(ctx, bson.M{"notificationId": id, "actionTaken": true})
		}

		// Contar usuarios únicos afectados
		pipeline := []bson.M{
			{"$match": bson.M{"notificationId": id}},
			{"$group": bson.M{"_id": "$userId"}},
			{"$count": "count"},
		}
		cursor, err := readsCol.Aggregate(ctx, pipeline)
		var affectedUsers int64
		if err == nil {
			var result []bson.M
			if err := cursor.All(ctx, &result); err == nil && len(result) > 0 {
				if count, ok := result[0]["count"].(int32); ok {
					affectedUsers = int64(count)
				}
			}
			cursor.Close(ctx)
		}

		stats := domain.NotificationStats{
			NotificationID: id,
			TotalReads:     totalReads,
			TotalActions:   totalActions,
			AffectedUsers:  affectedUsers,
		}

		writeJSON(w, http.StatusOK, stats)
	}
}


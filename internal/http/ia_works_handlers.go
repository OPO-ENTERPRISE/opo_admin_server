package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"opo_admin_server/internal/config"
	"opo_admin_server/internal/domain"
	"opo_admin_server/internal/services"

	"github.com/google/uuid"
)

// AdminIAWorksUploadFile - Subir archivo y convertirlo a texto plano
func AdminIAWorksUploadFile(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📤 [IA-WORKS-UPLOAD] Iniciando procesamiento de upload")
		
		// Validar método
		if r.Method != http.MethodPost {
			log.Printf("❌ [IA-WORKS-UPLOAD] Método no permitido: %s", r.Method)
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "solo se permite POST")
			return
		}

		log.Printf("📤 [IA-WORKS-UPLOAD] Content-Type: %s", r.Header.Get("Content-Type"))
		log.Printf("📤 [IA-WORKS-UPLOAD] Content-Length: %s", r.Header.Get("Content-Length"))

		// Parsear multipart form (límite de 100MB)
		log.Printf("📤 [IA-WORKS-UPLOAD] Parseando multipart form (límite: 100MB)...")
		err := r.ParseMultipartForm(100 << 20) // 100MB
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al parsear multipart form: %v", err)
			writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("error al parsear formulario: %v", err))
			return
		}
		log.Printf("✅ [IA-WORKS-UPLOAD] Multipart form parseado correctamente")

		// Obtener archivo
		log.Printf("📤 [IA-WORKS-UPLOAD] Obteniendo archivo del formulario...")
		file, handler, err := r.FormFile("file")
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al obtener archivo: %v", err)
			writeError(w, http.StatusBadRequest, "invalid_request", "archivo no encontrado en la solicitud")
			return
		}
		defer file.Close()
		
		log.Printf("✅ [IA-WORKS-UPLOAD] Archivo obtenido: %s (tamaño: %d bytes)", handler.Filename, handler.Size)

		// Validar tipo de archivo
		contentType := handler.Header.Get("Content-Type")
		log.Printf("📤 [IA-WORKS-UPLOAD] Content-Type del archivo: %s", contentType)
		
		if err := services.ValidateFileType(handler.Filename, contentType); err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error de validación de tipo: %v", err)
			writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
			return
		}
		log.Printf("✅ [IA-WORKS-UPLOAD] Tipo de archivo válido")

		// Crear directorio temporal si no existe
		tempDir := "/tmp/uploads"
		log.Printf("📤 [IA-WORKS-UPLOAD] Creando directorio temporal: %s", tempDir)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al crear directorio temporal: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", "error al crear directorio temporal")
			return
		}
		log.Printf("✅ [IA-WORKS-UPLOAD] Directorio temporal creado")

		// Generar ID único para el documento
		documentID := uuid.New().String()
		fileExt := filepath.Ext(handler.Filename)
		tempFilePath := filepath.Join(tempDir, documentID+fileExt)
		log.Printf("📤 [IA-WORKS-UPLOAD] Ruta temporal: %s", tempFilePath)

		// Guardar archivo temporalmente
		log.Printf("📤 [IA-WORKS-UPLOAD] Creando archivo temporal...")
		dst, err := os.Create(tempFilePath)
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al crear archivo temporal: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", "error al guardar archivo")
			return
		}
		defer dst.Close()

		// Copiar contenido del archivo
		log.Printf("📤 [IA-WORKS-UPLOAD] Copiando contenido del archivo...")
		bytesWritten, err := io.Copy(dst, file)
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al copiar archivo: %v", err)
			os.Remove(tempFilePath)
			writeError(w, http.StatusInternalServerError, "server_error", "error al guardar archivo")
			return
		}
		log.Printf("✅ [IA-WORKS-UPLOAD] Archivo guardado: %d bytes escritos", bytesWritten)

		// Determinar tipo de archivo
		fileType := strings.ToLower(fileExt)
		contentTypeForConversion := ""
		if fileType == ".docx" {
			contentTypeForConversion = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		} else if fileType == ".doc" {
			contentTypeForConversion = "application/msword"
		} else if fileType == ".pdf" {
			contentTypeForConversion = "application/pdf"
		} else if fileType == ".txt" {
			contentTypeForConversion = "text/plain"
		}

		// Convertir archivo a texto
		log.Printf("📤 [IA-WORKS-UPLOAD] Iniciando conversión del archivo (tipo: %s)...", contentTypeForConversion)
		text, err := services.ConvertFileToText(tempFilePath, contentTypeForConversion)
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al convertir archivo: %v", err)
			os.Remove(tempFilePath)
			writeError(w, http.StatusInternalServerError, "conversion_error", fmt.Sprintf("error al convertir archivo: %v", err))
			return
		}
		log.Printf("✅ [IA-WORKS-UPLOAD] Archivo convertido exitosamente. Texto extraído: %d caracteres", len(text))

		// Usar contentType original para almacenar
		if contentTypeForConversion != "" {
			fileType = contentTypeForConversion
		}

		// Limpiar archivo temporal
		log.Printf("📤 [IA-WORKS-UPLOAD] Limpiando archivo temporal...")
		if err := os.Remove(tempFilePath); err != nil {
			log.Printf("⚠️ [IA-WORKS-UPLOAD] Advertencia: no se pudo eliminar archivo temporal: %v", err)
		}

		// Guardar documento en MongoDB
		log.Printf("📤 [IA-WORKS-UPLOAD] Conectando a MongoDB...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error conectando a MongoDB: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", "error al conectar con base de datos")
			return
		}
		defer client.Disconnect(context.Background())
		log.Printf("✅ [IA-WORKS-UPLOAD] Conectado a MongoDB (DB: %s)", cfg.DBName)

		documents := client.Database(cfg.DBName).Collection("documents")
		document := domain.Document{
			ID:        documentID,
			FileName:  handler.Filename,
			FileType:  fileType,
			Text:      text,
			Status:    "uploaded",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		log.Printf("📤 [IA-WORKS-UPLOAD] Guardando documento en MongoDB (ID: %s)...", documentID)
		_, err = documents.InsertOne(ctx, document)
		if err != nil {
			log.Printf("❌ [IA-WORKS-UPLOAD] Error al guardar documento en MongoDB: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", fmt.Sprintf("error al guardar documento: %v", err))
			return
		}
		log.Printf("✅ [IA-WORKS-UPLOAD] Documento guardado en MongoDB exitosamente")

		// Preparar respuesta
		response := domain.UploadFileResponse{
			DocumentID: documentID,
			FileName:   handler.Filename,
			FileType:   fileType,
			Text:       text,
			Status:     "uploaded",
		}

		log.Printf("✅ [IA-WORKS-UPLOAD] Upload completado exitosamente. DocumentID: %s", documentID)
		writeJSON(w, http.StatusOK, response)
	}
}

// AdminIAWorksProcessVector - Procesar documento a vectores y guardar en Pinecone
func AdminIAWorksProcessVector(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.ProcessVectorRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validar request
		if req.DocumentID == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "documentId requerido")
			return
		}

		// Validar configuración de embedding
		if req.EmbeddingConfig.ChunkSize < 100 || req.EmbeddingConfig.ChunkSize > 2000 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "chunkSize debe estar entre 100 y 2000")
			return
		}

		if req.EmbeddingConfig.Overlap < 0 || req.EmbeddingConfig.Overlap > 500 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "overlap debe estar entre 0 y 500")
			return
		}

		// Obtener documento de MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			log.Printf("Error conectando a MongoDB: %v", err)
			writeError(w, http.StatusInternalServerError, "server_error", "error al conectar con base de datos")
			return
		}
		defer client.Disconnect(context.Background())

		documents := client.Database(cfg.DBName).Collection("documents")
		var document domain.Document
		if err := documents.FindOne(ctx, map[string]interface{}{"_id": req.DocumentID}).Decode(&document); err != nil {
			log.Printf("Error al buscar documento: %v", err)
			writeError(w, http.StatusNotFound, "not_found", "documento no encontrado")
			return
		}

		// Validar que el documento tenga texto
		if document.Text == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "el documento no tiene texto para procesar")
			return
		}

		// Dividir texto en chunks
		chunks, err := services.ChunkText(document.Text, req.EmbeddingConfig)
		if err != nil {
			log.Printf("Error al dividir texto: %v", err)
			writeError(w, http.StatusInternalServerError, "processing_error", fmt.Sprintf("error al dividir texto: %v", err))
			return
		}

		log.Printf("Texto dividido en %d chunks", len(chunks))

		// Generar embeddings
		embeddings, err := services.GenerateEmbeddings(chunks, req.EmbeddingConfig)
		if err != nil {
			log.Printf("Error al generar embeddings: %v", err)
			writeError(w, http.StatusInternalServerError, "embedding_error", fmt.Sprintf("error al generar embeddings: %v", err))
			return
		}

		log.Printf("Generados %d embeddings", len(embeddings))

		// Preparar vectores para Pinecone
		vectors := make([]domain.Vector, len(chunks))
		for i, chunk := range chunks {
			vectorID := fmt.Sprintf("%s-chunk-%d", req.DocumentID, i)
			
			// Preparar metadata
			metadata := make(map[string]interface{})
			metadata["documentId"] = req.DocumentID
			metadata["fileName"] = document.FileName
			metadata["chunkIndex"] = i
			metadata["text"] = chunk
			metadata["createdAt"] = time.Now().Format(time.RFC3339)
			
			// Agregar metadata personalizada si existe
			if req.EmbeddingConfig.Metadata != nil {
				for k, v := range req.EmbeddingConfig.Metadata {
					metadata[k] = v
				}
			}

			vectors[i] = domain.Vector{
				ID:         vectorID,
				Values:     embeddings[i],
				Metadata:   metadata,
				DocumentID: req.DocumentID,
				ChunkIndex: i,
				Text:       chunk,
			}
		}

		// Guardar en Pinecone
		namespace := fmt.Sprintf("document-%s", req.DocumentID)
		indexName := "admin-docs" // Por defecto, puede ser configurable
		
		if cfg.PineconeAPIKey == "" {
			writeError(w, http.StatusInternalServerError, "configuration_error", "PINECONE_API_KEY no configurada")
			return
		}

		pineconeClient := services.NewPineconeClient(cfg.PineconeAPIKey, indexName)
		if err := pineconeClient.StoreVectors(vectors, namespace); err != nil {
			log.Printf("Error al guardar en Pinecone: %v", err)
			writeError(w, http.StatusInternalServerError, "pinecone_error", fmt.Sprintf("error al guardar en Pinecone: %v", err))
			return
		}

		log.Printf("Guardados %d vectores en Pinecone (namespace: %s)", len(vectors), namespace)

		// Actualizar estado del documento en MongoDB
		documents.UpdateOne(ctx, map[string]interface{}{"_id": req.DocumentID}, map[string]interface{}{
			"$set": map[string]interface{}{
				"status":    "processed",
				"updatedAt": time.Now(),
			},
		})

		// Preparar respuesta
		response := domain.ProcessVectorResponse{
			VectorID:   fmt.Sprintf("vector-%s", req.DocumentID),
			Status:     "processed",
			ChunksCount: len(chunks),
		}

		writeJSON(w, http.StatusOK, response)
	}
}


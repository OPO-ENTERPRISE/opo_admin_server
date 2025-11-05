package domain

import "time"

// EmbeddingConfig representa la configuración para generar embeddings
type EmbeddingConfig struct {
	ChunkSize            int                    `json:"chunkSize" bson:"chunkSize"`
	Overlap              int                    `json:"overlap" bson:"overlap"`
	EmbeddingModel       string                 `json:"embeddingModel" bson:"embeddingModel"` // "openai", "huggingface", etc.
	ChunkingStrategy     string                 `json:"chunkingStrategy" bson:"chunkingStrategy"` // "characters", "paragraphs", "sections"
	Metadata             map[string]interface{} `json:"metadata" bson:"metadata"`
	OpenAIAPIKey         string                 `json:"openaiApiKey,omitempty" bson:"openaiApiKey,omitempty"`
	HuggingFaceAPIKey    string                 `json:"huggingFaceApiKey,omitempty" bson:"huggingFaceApiKey,omitempty"`
}

// Document representa un documento procesado
type Document struct {
	ID        string    `json:"id" bson:"_id"`
	FileName  string    `json:"fileName" bson:"fileName"`
	FileType  string    `json:"fileType" bson:"fileType"`
	Text      string    `json:"text" bson:"text"`
	Status    string    `json:"status" bson:"status"` // "uploaded", "processed", "error"
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// Vector representa un vector almacenado en Pinecone
type Vector struct {
	ID        string                 `json:"id" bson:"id"`
	Values    []float32              `json:"values" bson:"values"`
	Metadata  map[string]interface{} `json:"metadata" bson:"metadata"`
	DocumentID string                `json:"documentId" bson:"documentId"`
	ChunkIndex int                   `json:"chunkIndex" bson:"chunkIndex"`
	Text      string                 `json:"text" bson:"text"`
}

// ProcessVectorRequest representa la solicitud para procesar un documento a vectores
type ProcessVectorRequest struct {
	DocumentID      string          `json:"documentId"`
	EmbeddingConfig EmbeddingConfig `json:"embeddingConfig"`
}

// UploadFileResponse representa la respuesta al subir un archivo
type UploadFileResponse struct {
	DocumentID string `json:"documentId"`
	FileName   string `json:"fileName"`
	FileType   string `json:"fileType"`
	Text       string `json:"text"`
	Status     string `json:"status"`
}

// ProcessVectorResponse representa la respuesta al procesar un documento
type ProcessVectorResponse struct {
	VectorID   string `json:"vectorId"`
	Status     string `json:"status"`
	ChunksCount int   `json:"chunksCount"`
}


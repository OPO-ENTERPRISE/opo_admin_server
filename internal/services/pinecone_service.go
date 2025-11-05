package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"opo_admin_server/internal/domain"
)

const (
	PineconeBaseURL = "https://api.pinecone.io"
)

// PineconeClient maneja la comunicación con Pinecone
type PineconeClient struct {
	APIKey      string
	BaseURL     string
	Environment string
	IndexName   string
	HTTPClient  *http.Client
}

// NewPineconeClient crea un nuevo cliente de Pinecone
func NewPineconeClient(apiKey, indexName string) *PineconeClient {
	return &PineconeClient{
		APIKey:     apiKey,
		BaseURL:    PineconeBaseURL,
		IndexName:  indexName,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// UpsertRequest representa una solicitud de upsert a Pinecone
type UpsertRequest struct {
	Vectors   []PineconeVector `json:"vectors"`
	Namespace string           `json:"namespace,omitempty"`
}

// PineconeVector representa un vector en formato Pinecone
type PineconeVector struct {
	ID       string                 `json:"id"`
	Values   []float32              `json:"values"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpsertResponse representa la respuesta de Pinecone
type UpsertResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}

// StoreVectors almacena vectores en Pinecone
func (pc *PineconeClient) StoreVectors(vectors []domain.Vector, namespace string) error {
	if len(vectors) == 0 {
		return fmt.Errorf("no hay vectores para almacenar")
	}

	// Convertir domain.Vector a PineconeVector
	pineconeVectors := make([]PineconeVector, len(vectors))
	for i, v := range vectors {
		pineconeVectors[i] = PineconeVector{
			ID:       v.ID,
			Values:   v.Values,
			Metadata: v.Metadata,
		}
	}

	reqBody := UpsertRequest{
		Vectors:   pineconeVectors,
		Namespace: namespace,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error al serializar datos: %w", err)
	}

	url := fmt.Sprintf("%s/vectors/upsert", pc.getIndexURL())
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, jsonData)
	if err != nil {
		return fmt.Errorf("error al crear request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", pc.APIKey)

	resp, err := pc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error al realizar request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error de Pinecone (status %d): %s", resp.StatusCode, string(body))
	}

	var upsertResp UpsertResponse
	if err := json.NewDecoder(resp.Body).Decode(&upsertResp); err != nil {
		return fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	return nil
}

// QueryRequest representa una solicitud de consulta a Pinecone
type QueryRequest struct {
	Vector          []float32              `json:"vector"`
	TopK            int                    `json:"topK"`
	IncludeMetadata bool                   `json:"includeMetadata"`
	Namespace       string                 `json:"namespace,omitempty"`
	Filter          map[string]interface{} `json:"filter,omitempty"`
}

// QueryResponse representa la respuesta de una consulta
type QueryResponse struct {
	Matches []struct {
		ID       string                 `json:"id"`
		Score    float64                `json:"score"`
		Values   []float32              `json:"values,omitempty"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	} `json:"matches"`
}

// QueryVectors consulta vectores similares en Pinecone
func (pc *PineconeClient) QueryVectors(queryVector []float32, topK int, namespace string) ([]domain.Vector, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("vector de consulta vacío")
	}

	reqBody := QueryRequest{
		Vector:          queryVector,
		TopK:            topK,
		IncludeMetadata: true,
		Namespace:       namespace,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error al serializar datos: %w", err)
	}

	url := fmt.Sprintf("%s/query", pc.getIndexURL())
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, jsonData)
	if err != nil {
		return nil, fmt.Errorf("error al crear request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", pc.APIKey)

	resp, err := pc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al realizar request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error de Pinecone (status %d): %s", resp.StatusCode, string(body))
	}

	var queryResp QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	// Convertir respuesta a domain.Vector
	vectors := make([]domain.Vector, len(queryResp.Matches))
	for i, match := range queryResp.Matches {
		vectors[i] = domain.Vector{
			ID:       match.ID,
			Values:   match.Values,
			Metadata: match.Metadata,
		}
	}

	return vectors, nil
}

// getIndexURL obtiene la URL del índice de Pinecone
// Nota: Pinecone requiere que el índice se configure con la URL completa
// Por defecto, usamos: https://api.pinecone.io/indexes/{index-name}
// Si necesitas usar un índice específico, deberás configurar la URL completa
func (pc *PineconeClient) getIndexURL() string {
	// Para Pinecone Serverless, la URL es: https://{index-name}-{project-id}.svc.{environment}.pinecone.io
	// Para Pinecone Pods, la URL es: https://{index-name}-{project-id}.svc.{environment}.pinecone.io
	// Por ahora, usamos la API estándar que requiere configuración del índice
	return fmt.Sprintf("%s/indexes/%s", pc.BaseURL, pc.IndexName)
}

// StoreVectors almacena vectores en Pinecone (función de conveniencia)
func StoreVectors(vectors []domain.Vector, namespace, apiKey, indexName string) error {
	client := NewPineconeClient(apiKey, indexName)
	return client.StoreVectors(vectors, namespace)
}

// QueryVectors consulta vectores en Pinecone (función de conveniencia)
func QueryVectors(queryVector []float32, topK int, namespace, apiKey, indexName string) ([]domain.Vector, error) {
	client := NewPineconeClient(apiKey, indexName)
	return client.QueryVectors(queryVector, topK, namespace)
}


package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/nguyenthenguyen/docx"
)

const (
	MaxFileSize = 100 * 1024 * 1024 // 100MB
)

// ConvertPDFToText convierte un archivo PDF a texto plano
func ConvertPDFToText(filePath string) (string, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return "", fmt.Errorf("error al abrir PDF: %w", err)
	}
	defer doc.Close()

	var textBuilder strings.Builder
	totalPages := doc.NumPage()

	for i := 0; i < totalPages; i++ {
		text, err := doc.Text(i)
		if err != nil {
			return "", fmt.Errorf("error al leer página %d: %w", i, err)
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	return strings.TrimSpace(textBuilder.String()), nil
}

// ConvertWordToText convierte un archivo Word (DOCX) a texto plano
func ConvertWordToText(filePath string) (string, error) {
	// Verificar extensión
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".docx" && ext != ".doc" {
		return "", fmt.Errorf("formato no soportado: %s (solo se soporta .docx)", ext)
	}

	// Leer archivo DOCX
	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo Word: %w", err)
	}
	defer doc.Close()

	// Extraer texto
	text := doc.Editable().GetText()

	return strings.TrimSpace(text), nil
}

// ReadTextFile lee un archivo de texto plano
func ReadTextFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error al abrir archivo: %w", err)
	}
	defer file.Close()

	// Verificar tamaño
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("error al obtener información del archivo: %w", err)
	}

	if fileInfo.Size() > MaxFileSize {
		return "", fmt.Errorf("archivo demasiado grande: %d bytes (máximo: %d bytes)", fileInfo.Size(), MaxFileSize)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// ConvertFileToText convierte un archivo a texto plano según su extensión
func ConvertFileToText(filePath string, fileType string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Normalizar tipo de archivo
	if fileType == "" {
		fileType = ext
	}

	switch {
	case ext == ".pdf" || fileType == "application/pdf":
		return ConvertPDFToText(filePath)
	case ext == ".docx" || ext == ".doc" || fileType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" || fileType == "application/msword":
		return ConvertWordToText(filePath)
	case ext == ".txt" || fileType == "text/plain":
		return ReadTextFile(filePath)
	default:
		return "", fmt.Errorf("tipo de archivo no soportado: %s", ext)
	}
}

// ValidateFileType valida que el tipo de archivo sea soportado
func ValidateFileType(fileName string, contentType string) error {
	ext := strings.ToLower(filepath.Ext(fileName))
	
	// Validar por extensión
	allowedExts := []string{".pdf", ".docx", ".doc", ".txt"}
	isValidExt := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			isValidExt = true
			break
		}
	}

	if !isValidExt {
		return fmt.Errorf("extensión no permitida: %s. Extensiones permitidas: .pdf, .docx, .doc, .txt", ext)
	}

	// Validar por content type si está disponible
	if contentType != "" {
		allowedTypes := []string{
			"application/pdf",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/msword",
			"text/plain",
		}
		isValidType := false
		for _, allowedType := range allowedTypes {
			if contentType == allowedType {
				isValidType = true
				break
			}
		}
		if !isValidType {
			return fmt.Errorf("tipo de contenido no permitido: %s", contentType)
		}
	}

	return nil
}


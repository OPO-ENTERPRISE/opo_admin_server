package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gen2brain/go-fitz"
	"github.com/nguyenthenguyen/docx"
)

const (
	MaxFileSize = 100 * 1024 * 1024 // 100MB
)

type DocumentType string

const (
	DocumentTypePDF  DocumentType = "pdf"
	DocumentTypeDOCX DocumentType = "docx"
	DocumentTypeText DocumentType = "text"
)

var (
	allowedExtensions   = []string{".pdf", ".docx", ".txt"}
	supportedExtensions = map[string]DocumentType{
		".pdf":  DocumentTypePDF,
		".docx": DocumentTypeDOCX,
		".txt":  DocumentTypeText,
	}
	supportedMimeTypes = map[DocumentType][]string{
		DocumentTypePDF: {
			"application/pdf",
		},
		DocumentTypeDOCX: {
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		DocumentTypeText: {
			"text/plain",
		},
	}
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
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".docx" {
		return "", fmt.Errorf("formato no soportado: %s (solo se soporta .docx)", ext)
	}

	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo Word: %w", err)
	}
	defer doc.Close()

	content := doc.Editable().GetContent()
	text, err := extractTextFromDocxXML(content)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(text), nil
}

// extractTextFromDocxXML recorre el XML buscando nodos de texto y saltos relevantes
func extractTextFromDocxXML(xmlContent string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var builder strings.Builder

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error al parsear XML DOCX: %w", err)
		}

		startElement, ok := tok.(xml.StartElement)
		if !ok || startElement.Name.Space != "w" {
			continue
		}

		switch startElement.Name.Local {
		case "p":
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
		case "br":
			builder.WriteString("\n")
		case "tab":
			builder.WriteString("\t")
		case "t":
			var text string
			if err := decoder.DecodeElement(&text, &startElement); err != nil {
				return "", fmt.Errorf("error al decodificar texto DOCX: %w", err)
			}
			builder.WriteString(text)
		}
	}

	return strings.TrimSpace(builder.String()), nil
}

// ReadTextFile lee un archivo de texto plano
func ReadTextFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error al abrir archivo: %w", err)
	}
	defer file.Close()

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
	if err := ensureFileSizeWithinLimit(filePath); err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	docType, ok := supportedExtensions[ext]
	if !ok {
		return "", fmt.Errorf("extensión no permitida: %s. Extensiones permitidas: %s", ext, strings.Join(allowedExtensions, ", "))
	}

	detectedMime, err := detectMimeType(filePath)
	if err != nil {
		return "", err
	}

	if !mimeAllowedForType(docType, detectedMime) {
		return "", fmt.Errorf("el contenido detectado (%s) no coincide con la extensión %s", detectedMime, ext)
	}

	if fileType != "" && !mimeAllowedForType(docType, strings.ToLower(fileType)) {
		return "", fmt.Errorf("el tipo de contenido declarado (%s) no coincide con la extensión %s", fileType, ext)
	}

	switch docType {
	case DocumentTypePDF:
		return ConvertPDFToText(filePath)
	case DocumentTypeDOCX:
		return ConvertWordToText(filePath)
	case DocumentTypeText:
		return ReadTextFile(filePath)
	default:
		return "", fmt.Errorf("tipo de archivo no soportado: %s", ext)
	}
}

// ValidateFileType valida que el tipo de archivo sea soportado
func ValidateFileType(fileName string, contentType string) error {
	ext := strings.ToLower(filepath.Ext(fileName))
	docType, ok := supportedExtensions[ext]
	if !ok {
		return fmt.Errorf("extensión no permitida: %s. Extensiones permitidas: %s", ext, strings.Join(allowedExtensions, ", "))
	}

	if contentType != "" && !mimeAllowedForType(docType, strings.ToLower(contentType)) {
		return fmt.Errorf("tipo de contenido no permitido: %s para la extensión %s", contentType, ext)
	}

	return nil
}

func ensureFileSizeWithinLimit(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("error al obtener información del archivo: %w", err)
	}

	if info.Size() > MaxFileSize {
		return fmt.Errorf("archivo demasiado grande: %d bytes (máximo: %d bytes)", info.Size(), MaxFileSize)
	}

	return nil
}

func detectMimeType(filePath string) (string, error) {
	mime, err := mimetype.DetectFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error al detectar tipo de archivo: %w", err)
	}
	return strings.ToLower(mime.String()), nil
}

func mimeAllowedForType(docType DocumentType, candidate string) bool {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if candidate == "" {
		return false
	}

	for _, allowed := range supportedMimeTypes[docType] {
		if candidate == allowed || strings.HasPrefix(candidate, allowed+";") {
			return true
		}
	}

	return false
}

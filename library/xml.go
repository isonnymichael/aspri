package library

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Initiate XML Function
func InitiateXMLFunction(flags Flag) {
	// Check if mode is XML and extract operation is requested
	if *flags.XML && *flags.Extract {
		// Create a new XMLHandler instance
		xmlHandler := NewXMLHandler()

		// Extract URLs from the XML file
		urls, err := xmlHandler.ExtractURLs(*flags.Path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Print the extracted URLs
		for _, url := range urls {
			fmt.Println(url)
		}
	}
}

// XMLHandler handles XML-related operations
type XMLHandler struct {
	FilePath string
}

// NewXMLHandler creates a new XMLHandler instance
func NewXMLHandler() *XMLHandler {
	return &XMLHandler{}
}

// ExtractURLs extracts all URLs from an XML file
func (x *XMLHandler) ExtractURLs(filePath string) ([]string, error) {
	// Open and read the XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Create a decoder for XML content
	decoder := xml.NewDecoder(strings.NewReader(string(content)))

	// Store found URLs
	urls := make([]string, 0)

	// Track current element path
	var currentPath []string

	// Common file extensions to ignore
	fileExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".zip", ".rar", ".tar", ".gz",
		".mp3", ".mp4", ".avi", ".mov",
		".css", ".js", ".ico",
	}

	// Helper function to check if URL ends with ignored file extension
	isFileURL := func(url string) bool {
		urlLower := strings.ToLower(url)
		for _, ext := range fileExtensions {
			if strings.HasSuffix(urlLower, ext) {
				return true
			}
		}
		return false
	}

	// Iterate through XML tokens
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error decoding XML: %v", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Add element name to current path
			currentPath = append(currentPath, t.Name.Local)

			// Check attributes for URLs
			for _, attr := range t.Attr {
				if isURLAttribute(attr.Name.Local) {
					if isValidURL(attr.Value) && !isFileURL(attr.Value) {
						urls = append(urls, attr.Value)
					}
				}
			}

		case xml.EndElement:
			// Remove element from current path when closing tag is encountered
			if len(currentPath) > 0 {
				currentPath = currentPath[:len(currentPath)-1]
			}

		case xml.CharData:
			// Check text content for URLs
			text := string(t)
			if isValidURL(text) && !isFileURL(text) {
				urls = append(urls, strings.TrimSpace(text))
			}
		}
	}

	return urls, nil
}

// isURLAttribute checks if the attribute name typically contains URLs
func isURLAttribute(attrName string) bool {
	urlAttributes := []string{
		"href",
		"src",
		"url",
		"link",
		"data",
		"action",
	}

	attrName = strings.ToLower(attrName)
	for _, urlAttr := range urlAttributes {
		if strings.Contains(attrName, urlAttr) {
			return true
		}
	}
	return false
}

// isValidURL performs basic URL validation
func isValidURL(text string) bool {
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")
}

// HandleXMLCommand processes XML-related commands
func HandleXMLCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("insufficient arguments for XML command")
	}

	switch args[0] {
	case "--extract":
		if len(args) < 3 || args[1] != "--path" {
			return fmt.Errorf("missing file path for XML extraction")
		}

		filePath := args[2]
		handler := NewXMLHandler()
		urls, err := handler.ExtractURLs(filePath)
		if err != nil {
			return err
		}

		// Print extracted URLs
		fmt.Println("Extracted URLs:")
		for _, url := range urls {
			fmt.Println(url)
		}

	default:
		return fmt.Errorf("unknown XML command: %s", args[0])
	}

	return nil
}
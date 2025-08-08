package library

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// InitiateFabFunction handles FAB template fixes and checks
func InitiateFabFunction(flags Flag) {
	if *flags.FabTemplatesFix {
		err := FixFabTemplates(*flags.Path)
		if err != nil {
			fmt.Printf("❌ Error fixing FAB templates: %v\n", err)
		} else {
			fmt.Println("✅ All files processed successfully.")
		}
	}
	
	if *flags.FabTemplatesCheck {
		err := CheckFabTemplates(*flags.Path)
		if err != nil {
			fmt.Printf("❌ Error checking FAB templates: %v\n", err)
			os.Exit(1)
		}
	}
}

// FixFabTemplates processes all JSON files in the specified directory
// and fixes the structure according to the PowerShell script logic
func FixFabTemplates(templateDir string) error {
	if templateDir == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		templateDir = currentDir
	}

	// Find all JSON files in the directory
	files, err := filepath.Glob(filepath.Join(templateDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No JSON files found in directory: %s\n", templateDir)
		return nil
	}

	// Process each JSON file
	for _, file := range files {
		fmt.Printf("Processing %s...\n", filepath.Base(file))
		
		err := processJSONFile(file)
		if err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", filepath.Base(file), err)
			continue
		}
	}

	return nil
}

// processJSONFile handles the logic for a single JSON file
func processJSONFile(filePath string) error {
	// Read the file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON into a generic map to allow dynamic manipulation
	var jsonData map[string]interface{}
	err = json.Unmarshal(content, &jsonData)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	modified := false

	// Extract expected ID from filename (remove .json extension)
	fileName := filepath.Base(filePath)
	expectedID := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Check if ID matches filename and fix if needed
	if currentID, hasID := jsonData["id"]; hasID {
		if currentIDStr, ok := currentID.(string); ok {
			if currentIDStr != expectedID {
				fmt.Printf("  - Found ID mismatch: '%s' should be '%s' - fixing...\n", currentIDStr, expectedID)
				jsonData["id"] = expectedID
				modified = true
			}
		}
	} else {
		// Add missing ID field
		fmt.Printf("  - Missing 'id' field - adding with value '%s'...\n", expectedID)
		jsonData["id"] = expectedID
		modified = true
	}

	// Check if 'locations' exists instead of 'location'
	if locations, hasLocations := jsonData["locations"]; hasLocations {
		fmt.Println("  - Found 'locations' (plural) - fixing...")
		
		// Remove 'locations' and add 'location'
		delete(jsonData, "locations")
		
		// Create default location structure
		defaultLocation := []map[string]interface{}{
			{
				"logic": nil,
				"rules": []interface{}{},
			},
		}

		// If locations had valid data, use it; otherwise use default
		if locationsArray, ok := locations.([]interface{}); ok && len(locationsArray) > 0 {
			jsonData["location"] = locations
		} else {
			jsonData["location"] = defaultLocation
		}
		
		modified = true
	} else if _, hasLocation := jsonData["location"]; !hasLocation {
		// Missing 'location' - add it
		fmt.Println("  - Missing 'location' - adding...")
		jsonData["location"] = []map[string]interface{}{
			{
				"logic": nil,
				"rules": []interface{}{},
			},
		}
		modified = true
	} else {
		fmt.Println("  - Has 'location' field - checking structure...")
	}

	// Fix rules structure if needed
	if location, hasLocation := jsonData["location"]; hasLocation {
		if locationArray, ok := location.([]interface{}); ok && len(locationArray) > 0 {
			for _, loc := range locationArray {
				if locMap, ok := loc.(map[string]interface{}); ok {
					if rules, hasRules := locMap["rules"]; hasRules {
						if rulesArray, ok := rules.([]interface{}); ok && len(rulesArray) > 0 {
							for _, rule := range rulesArray {
								if ruleMap, ok := rule.(map[string]interface{}); ok {
									// Check if rule has 'id' instead of 'type'
									if idValue, hasId := ruleMap["id"]; hasId {
										if _, hasType := ruleMap["type"]; !hasType {
											fmt.Println("    - Found rule with 'id' instead of 'type' - fixing...")
											delete(ruleMap, "id")
											ruleMap["type"] = idValue
											modified = true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Always prettify JSON, even if no structural changes were made
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Check if the prettified content is different from the original
	if !modified && string(jsonBytes) == strings.TrimSpace(string(content)) {
		fmt.Printf("  - No changes needed for %s\n", filepath.Base(filePath))
		return nil
	}

	err = ioutil.WriteFile(filePath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	if modified {
		fmt.Printf("  - Saved changes to %s\n", filepath.Base(filePath))
	} else {
		fmt.Printf("  - Prettified JSON formatting for %s\n", filepath.Base(filePath))
	}

	return nil
}

// ValidationResult holds the result of template validation
type ValidationResult struct {
	FileName string
	IsValid  bool
	Errors   []string
}

// CheckFabTemplates validates all JSON template files in the specified directory
func CheckFabTemplates(templateDir string) error {
	if templateDir == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		templateDir = currentDir
	}

	fmt.Println("🔍 Checking template validity...")
	fmt.Printf("📁 Templates directory: %s\n\n", templateDir)

	// Check if directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("templates directory does not exist: %s", templateDir)
	}

	// Find all JSON files
	files, err := filepath.Glob(filepath.Join(templateDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("⚠️  No JSON template files found in the templates directory.")
		return nil
	}

	fmt.Printf("📊 Found %d template files to check\n\n", len(files))

	var validCount, invalidCount int
	var invalidTemplates []ValidationResult

	// Validate each template file
	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		validation := validateTemplate(filePath, fileName)

		if validation.IsValid {
			validCount++
			fmt.Printf("✅ %s\n", fileName)
		} else {
			invalidCount++
			invalidTemplates = append(invalidTemplates, validation)
			fmt.Printf("❌ %s\n", fileName)
			for _, err := range validation.Errors {
				fmt.Printf("   └─ %s\n", err)
			}
		}
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📋 VALIDATION SUMMARY")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("✅ Valid templates: %d\n", validCount)
	fmt.Printf("❌ Invalid templates: %d\n", invalidCount)
	fmt.Printf("📊 Total templates: %d\n", len(files))

	if invalidCount > 0 {
		fmt.Println("\n🚨 INVALID TEMPLATES DETAILS:")
		fmt.Println(strings.Repeat("-", 30))

		for i, template := range invalidTemplates {
			fmt.Printf("%d. %s\n", i+1, template.FileName)
			for _, err := range template.Errors {
				fmt.Printf("   • %s\n", err)
			}
			fmt.Println()
		}

		fmt.Println("⚠️  Please fix the invalid templates before proceeding.")
		os.Exit(1)
	} else {
		fmt.Println("\n🎉 All templates are valid!")
	}

	return nil
}

// validateTemplate checks if a single template file is valid
func validateTemplate(filePath, fileName string) ValidationResult {
	result := ValidationResult{
		FileName: fileName,
		IsValid:  true,
		Errors:   []string{},
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, "File does not exist")
		return result
	}

	// Read and parse JSON
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading file: %v", err))
		return result
	}

	var template map[string]interface{}
	err = json.Unmarshal(content, &template)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON: %v", err))
		return result
	}

	// Check if template has an id field
	id, hasID := template["id"]
	if !hasID {
		result.IsValid = false
		result.Errors = append(result.Errors, "Missing \"id\" field in template")
		return result
	}

	// Check if filename matches id
	idStr, ok := id.(string)
	if !ok {
		result.IsValid = false
		result.Errors = append(result.Errors, "ID field must be a string")
		return result
	}

	expectedFileName := idStr + ".json"
	if fileName != expectedFileName {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Filename mismatch: expected \"%s\", got \"%s\"", expectedFileName, fileName))
	}

	return result
}

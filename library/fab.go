package library

import (
	"bytes"
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

	// PRIORITY 1: Normalize filenames (underscore to hyphen) and handle duplicates
	err := normalizeFilenames(templateDir)
	if err != nil {
		return fmt.Errorf("failed to normalize filenames: %w", err)
	}

	// Find all JSON files in the directory (after normalization)
	files, err := filepath.Glob(filepath.Join(templateDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No JSON files found in directory: %s\n", templateDir)
		return nil
	}

	// Process each JSON file for content fixes and prettification
	var filesToDelete []string
	for _, file := range files {
		fmt.Printf("Processing %s...\n", filepath.Base(file))
		
		err := processJSONFile(file)
		if err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", filepath.Base(file), err)
			
			// Check if it's a JSON parsing error
			if strings.Contains(err.Error(), "failed to parse JSON") {
				filesToDelete = append(filesToDelete, file)
				fmt.Printf("  📝 Marked for deletion due to JSON parsing error: %s\n", filepath.Base(file))
			}
			continue
		}
	}

	// Delete files with JSON parsing errors
	if len(filesToDelete) > 0 {
		fmt.Printf("\n🗑️  Deleting %d files with JSON parsing errors...\n", len(filesToDelete))
		for _, file := range filesToDelete {
			err := os.Remove(file)
			if err != nil {
				fmt.Printf("❌ Failed to delete %s: %v\n", filepath.Base(file), err)
			} else {
				fmt.Printf("✅ Deleted: %s\n", filepath.Base(file))
			}
		}
	}

	// Final step: Prettify all JSON files (including those that weren't modified)
	fmt.Println("\n📝 Final step: Prettifying all JSON files...")
	err = prettifyAllJSONFiles(templateDir)
	if err != nil {
		return fmt.Errorf("failed to prettify JSON files: %w", err)
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

	// Remove docs field if present
	if _, hasDocs := jsonData["docs"]; hasDocs {
		fmt.Println("  - Removing 'docs' field...")
		delete(jsonData, "docs")
		modified = true
	}

	// Remove extraOptions field if present
	if _, hasExtraOptions := jsonData["extraOptions"]; hasExtraOptions {
		fmt.Println("  - Removing 'extraOptions' field...")
		delete(jsonData, "extraOptions")
		modified = true
	}

	// Fix location structure and rules
	if location, hasLocation := jsonData["location"]; hasLocation {
		locationFixed := fixLocationStructure(location)
		if locationFixed != nil {
			jsonData["location"] = locationFixed
			modified = true
		}
		
		// Remove location if rules are empty
		if shouldRemoveLocation(jsonData["location"]) {
			fmt.Println("  - Removing location with empty rules...")
			delete(jsonData, "location")
			modified = true
		}
	}

	// Save changes only if modifications were made to preserve original formatting
	if modified {
		// Format JSON properly but accept that Go will reorder keys alphabetically
		jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		err = ioutil.WriteFile(filePath, jsonBytes, 0644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		
		fmt.Printf("  - Saved changes to %s (Note: keys may be reordered alphabetically)\n", filepath.Base(filePath))
	} else {
		fmt.Printf("  - No changes needed for %s\n", filepath.Base(filePath))
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

// PrettifyFabTemplates formats all JSON files in the specified directory
// while trying to preserve the original key order as much as possible
func PrettifyFabTemplates(templateDir string) error {
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
		fmt.Printf("Prettifying %s...\n", filepath.Base(file))
		
		err := prettifyJSONFile(file)
		if err != nil {
			fmt.Printf("❌ Error prettifying %s: %v\n", filepath.Base(file), err)
			continue
		}
	}

	return nil
}

// prettifyJSONFile formats a single JSON file using json.Indent to preserve structure
func prettifyJSONFile(filePath string) error {
	// Read the file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Use json.Indent which preserves the original key order better than Marshal/Unmarshal
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, content, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	// Check if the content actually changed
	if string(content) == prettyJSON.String() {
		fmt.Printf("  - Already properly formatted: %s\n", filepath.Base(filePath))
		return nil
	}

	// Write the prettified content back to the file
	err = ioutil.WriteFile(filePath, prettyJSON.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("  - Prettified: %s\n", filepath.Base(filePath))
	return nil
}

// normalizeFilenames renames files with underscores to hyphens and handles duplicates
func normalizeFilenames(templateDir string) error {
	fmt.Println("🔄 Normalizing filenames (underscore to hyphen)...")

	// Find all JSON files in the directory
	files, err := filepath.Glob(filepath.Join(templateDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	var renamedFiles []string
	duplicateCount := 0

	// Process each file for renaming
	for _, file := range files {
		fileName := filepath.Base(file)
		fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		
		// Check if filename contains underscores
		if strings.Contains(fileNameWithoutExt, "_") {
			// Convert underscores to hyphens
			newFileNameWithoutExt := strings.ReplaceAll(fileNameWithoutExt, "_", "-")
			newFileName := newFileNameWithoutExt + ".json"
			newFilePath := filepath.Join(templateDir, newFileName)
			
			fmt.Printf("  - Renaming: %s → %s\n", fileName, newFileName)
			
			// Check if target file already exists
			if _, err := os.Stat(newFilePath); err == nil {
				// Target file exists, we have a duplicate
				fmt.Printf("    ⚠️  Target file %s already exists - removing original %s\n", newFileName, fileName)
				
				// Remove the original file (the one with underscores)
				err = os.Remove(file)
				if err != nil {
					fmt.Printf("    ❌ Failed to remove duplicate %s: %v\n", fileName, err)
					continue
				}
				
				duplicateCount++
				fmt.Printf("    ✅ Removed duplicate: %s\n", fileName)
			} else {
				// Target doesn't exist, safe to rename
				err = os.Rename(file, newFilePath)
				if err != nil {
					fmt.Printf("    ❌ Failed to rename %s to %s: %v\n", fileName, newFileName, err)
					continue
				}
				
				renamedFiles = append(renamedFiles, fmt.Sprintf("%s → %s", fileName, newFileName))
				fmt.Printf("    ✅ Renamed: %s → %s\n", fileName, newFileName)
			}
		}
	}

	// Summary
	if len(renamedFiles) > 0 || duplicateCount > 0 {
		fmt.Printf("📋 Filename normalization summary:\n")
		fmt.Printf("  ✅ Files renamed: %d\n", len(renamedFiles))
		fmt.Printf("  🗑️  Duplicates removed: %d\n", duplicateCount)
		
		if len(renamedFiles) > 0 {
			fmt.Println("  📝 Renamed files:")
			for _, rename := range renamedFiles {
				fmt.Printf("    • %s\n", rename)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("  ✅ No files needed renaming\n")
	}

	return nil
}

// fixLocationStructure fixes the location array structure and rules
func fixLocationStructure(location interface{}) interface{} {
	locationArray, ok := location.([]interface{})
	if !ok {
		return nil
	}

	var fixed bool
	var newLocationArray []interface{}

	// Check if this is old format (direct rules in location array)
	isOldFormat := false
	for _, loc := range locationArray {
		if locMap, ok := loc.(map[string]interface{}); ok {
			// Old format has id/type, operator, value directly in location
			if _, hasId := locMap["id"]; hasId {
				isOldFormat = true
				break
			}
			if _, hasType := locMap["type"]; hasType {
				if _, hasLogic := locMap["logic"]; !hasLogic || locMap["logic"] == nil {
					if _, hasRules := locMap["rules"]; !hasRules {
						isOldFormat = true
						break
					}
				}
			}
		}
	}

	if isOldFormat {
		// Convert entire old format to new format
		fmt.Println("    - Converting old format location structure to new format...")
		newLocMap := make(map[string]interface{})
		newLocMap["logic"] = nil
		var newRules []interface{}

		for _, loc := range locationArray {
			locMap, ok := loc.(map[string]interface{})
			if !ok {
				continue
			}

			newRule := make(map[string]interface{})

			// Fix id -> type
			if idVal, hasId := locMap["id"]; hasId {
				newRule["type"] = idVal
			} else if typeVal, hasType := locMap["type"]; hasType {
				newRule["type"] = typeVal
			}

			// Handle operator and value
			if operator, hasOp := locMap["operator"]; hasOp {
				// Fix between operator
				if operator == "between" {
					fmt.Println("      - Converting 'between' operator to '=='...")
					newRule["operator"] = "=="
					fixed = true
				} else {
					newRule["operator"] = operator
				}
			}

			// Ensure value is string
			if value, hasVal := locMap["value"]; hasVal {
				newRule["value"] = convertValueToString(value)
				if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", newRule["value"]) {
					fmt.Println("      - Converting value to string...")
					fixed = true
				}
			}

			// Add logic field
			newRule["logic"] = "OR"

			// Remove value_type if present
			delete(newRule, "value_type")

			newRules = append(newRules, newRule)
		}

		newLocMap["rules"] = newRules
		newLocationArray = append(newLocationArray, newLocMap)
		fixed = true
	} else {
		// New format, just fix the rules
		for _, loc := range locationArray {
			locMap, ok := loc.(map[string]interface{})
			if !ok {
				continue
			}

			newLocMap := make(map[string]interface{})
			newLocMap["logic"] = locMap["logic"]

			if rules, hasRules := locMap["rules"]; hasRules {
				rulesArray, ok := rules.([]interface{})
				if ok {
					var newRules []interface{}
					for _, rule := range rulesArray {
						ruleMap, ok := rule.(map[string]interface{})
						if !ok {
							continue
						}

						newRule := make(map[string]interface{})

						// Fix id -> type
						if idVal, hasId := ruleMap["id"]; hasId {
							fmt.Println("      - Converting rule 'id' to 'type'...")
							newRule["type"] = idVal
							fixed = true
						} else if typeVal, hasType := ruleMap["type"]; hasType {
							newRule["type"] = typeVal
						}

						// Handle operator
						if operator, hasOp := ruleMap["operator"]; hasOp {
							// Fix between operator
							if operator == "between" {
								fmt.Println("      - Converting 'between' operator to '=='...")
								newRule["operator"] = "=="
								fixed = true
							} else {
								newRule["operator"] = operator
							}
						}

						// Ensure value is string
						if value, hasVal := ruleMap["value"]; hasVal {
							newRule["value"] = convertValueToString(value)
							if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", newRule["value"]) {
								fmt.Println("      - Converting value to string...")
								fixed = true
							}
						}

						// Ensure logic field exists and is not null
						if logic, hasLogic := ruleMap["logic"]; hasLogic && logic != nil {
							newRule["logic"] = logic
						} else {
							fmt.Println("      - Setting missing/null logic to 'OR'...")
							newRule["logic"] = "OR"
							fixed = true
						}

						// Remove value_type if present
						if _, hasValueType := ruleMap["value_type"]; hasValueType {
							fmt.Println("      - Removing 'value_type' field...")
							fixed = true
						}

						newRules = append(newRules, newRule)
					}
					newLocMap["rules"] = newRules
				} else {
					newLocMap["rules"] = []interface{}{}
				}
			} else {
				newLocMap["rules"] = []interface{}{}
			}

			newLocationArray = append(newLocationArray, newLocMap)
		}
	}

	if fixed {
		return newLocationArray
	}
	return nil
}

// convertValueToString ensures value is always a string
func convertValueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		// Convert array to first element as string
		if len(v) > 0 {
			return fmt.Sprintf("%v", v[0])
		}
		return ""
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// shouldRemoveLocation checks if location should be removed (empty rules)
func shouldRemoveLocation(location interface{}) bool {
	locationArray, ok := location.([]interface{})
	if !ok {
		return false
	}

	for _, loc := range locationArray {
		locMap, ok := loc.(map[string]interface{})
		if !ok {
			continue
		}

		if rules, hasRules := locMap["rules"]; hasRules {
			rulesArray, ok := rules.([]interface{})
			if ok && len(rulesArray) > 0 {
				return false // Has rules, don't remove
			}
		}
	}

	return true // All locations have empty rules
}

// prettifyAllJSONFiles prettifies all JSON files in a directory using custom ordering
func prettifyAllJSONFiles(templateDir string) error {
	// Find all JSON files in the directory
	files, err := filepath.Glob(filepath.Join(templateDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	prettifiedCount := 0

	// Process each JSON file for prettification
	for _, file := range files {
		fileName := filepath.Base(file)
		
		// Read and parse JSON
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Printf("  ❌ Error reading %s: %v\n", fileName, err)
			continue
		}

		var jsonData map[string]interface{}
		err = json.Unmarshal(content, &jsonData)
		if err != nil {
			fmt.Printf("  ❌ Error parsing %s: %v\n", fileName, err)
			continue
		}

		// Create ordered JSON string
		orderedJSON := createOrderedJSON(jsonData)
		
		// Check if the content actually changed
		if strings.TrimSpace(string(content)) == strings.TrimSpace(orderedJSON) {
			fmt.Printf("  ✅ Already formatted: %s\n", fileName)
			continue
		}

		// Write the prettified content back to the file
		err = ioutil.WriteFile(file, []byte(orderedJSON), 0644)
		if err != nil {
			fmt.Printf("  ❌ Error writing %s: %v\n", fileName, err)
			continue
		}

		fmt.Printf("  ✅ Prettified: %s\n", fileName)
		prettifiedCount++
	}

	if prettifiedCount > 0 {
		fmt.Printf("📋 Prettification summary: %d files formatted\n", prettifiedCount)
	} else {
		fmt.Println("📋 All files were already properly formatted")
	}

	return nil
}

// createOrderedJSON creates a JSON string with custom key ordering
func createOrderedJSON(data map[string]interface{}) string {
	var buf bytes.Buffer
	writeOrderedJSON(&buf, data, 0)
	return buf.String()
}

// writeOrderedJSON writes JSON with specific key ordering
func writeOrderedJSON(buf *bytes.Buffer, data interface{}, indent int) {
	indentStr := strings.Repeat("  ", indent)
	nextIndentStr := strings.Repeat("  ", indent+1)

	switch v := data.(type) {
	case map[string]interface{}:
		buf.WriteString("{\n")
		
		// Define the key order (location always at the bottom)
		keyOrder := []string{
			"id", "name", "description", "license", "requires",
			"settings", "design", "cookie",
		}
		
		// Location will be handled separately at the end
		
		// Collect all keys that will be written (except location)
		var keysToWrite []string
		writtenKeys := make(map[string]bool)
		
		// Add keys in specified order
		for _, key := range keyOrder {
			if _, exists := v[key]; exists {
				keysToWrite = append(keysToWrite, key)
				writtenKeys[key] = true
			}
		}
		
		// Add remaining keys alphabetically (except location)
		var remainingKeys []string
		for key := range v {
			if !writtenKeys[key] && key != "location" {
				remainingKeys = append(remainingKeys, key)
			}
		}
		keysToWrite = append(keysToWrite, remainingKeys...)
		
		// Add location at the very end if it exists
		hasLocation := false
		if _, exists := v["location"]; exists {
			hasLocation = true
		}
		
		// Write all keys except location
		for i, key := range keysToWrite {
			buf.WriteString(nextIndentStr)
			buf.WriteString(`"` + key + `": `)
			writeOrderedJSON(buf, v[key], indent+1)
			if i < len(keysToWrite)-1 || hasLocation {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		
		// Write location at the very end
		if hasLocation {
			buf.WriteString(nextIndentStr)
			buf.WriteString(`"location": `)
			writeOrderedJSON(buf, v["location"], indent+1)
			buf.WriteString("\n")
		}
		
		buf.WriteString(indentStr + "}")

	case []interface{}:
		buf.WriteString("[\n")
		for i, item := range v {
			buf.WriteString(nextIndentStr)
			writeOrderedJSON(buf, item, indent+1)
			if i < len(v)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString(indentStr + "]")

	case string:
		escaped := strings.ReplaceAll(v, `"`, `\"`)
		buf.WriteString(`"` + escaped + `"`)

	case float64:
		if v == float64(int64(v)) {
			buf.WriteString(fmt.Sprintf("%.0f", v))
		} else {
			buf.WriteString(fmt.Sprintf("%g", v))
		}

	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case nil:
		buf.WriteString("null")

	default:
		// Fallback to standard JSON marshaling
		jsonBytes, _ := json.Marshal(v)
		buf.Write(jsonBytes)
	}
}

package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EnvVar represents a single environment variable
type EnvVar struct {
	Key         string
	Value       string
	Comment     string
	Section     string
	IsRequired  bool
	IsSecret    bool
	Example     string
}

// EnvConfig manages environment configuration
type EnvConfig struct {
	Variables   []EnvVar
	FilePath    string
	Sections    []string
}

// LoadEnvFile loads environment variables from a .env file
func LoadEnvFile(filePath string) (*EnvConfig, error) {
	config := &EnvConfig{
		FilePath:  filePath,
		Variables: make([]EnvVar, 0),
		Sections:  make([]string, 0),
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentSection string
	var currentComment string
	var lineNumber int

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			currentComment = ""
			continue
		}

		// Handle comments
		if strings.HasPrefix(line, "#") {
			comment := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			
			// Check for section headers (comments with ===)
			if strings.Contains(comment, "===") {
				sectionName := strings.Trim(comment, "= ")
				currentSection = sectionName
				if !contains(config.Sections, currentSection) {
					config.Sections = append(config.Sections, currentSection)
				}
				currentComment = ""
				continue
			}
			
			// Accumulate comments
			if currentComment != "" {
				currentComment += " " + comment
			} else {
				currentComment = comment
			}
			continue
		}

		// Handle environment variables
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Remove quotes if present
				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					value = value[1 : len(value)-1]
				}

				envVar := EnvVar{
					Key:        key,
					Value:      value,
					Comment:    currentComment,
					Section:    currentSection,
					IsRequired: isRequiredVar(key, value),
					IsSecret:   isSecretVar(key),
				}

				config.Variables = append(config.Variables, envVar)
				currentComment = ""
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading env file: %w", err)
	}

	// Sort sections for consistent display
	sort.Strings(config.Sections)

	return config, nil
}

// SaveEnvFile saves the environment configuration back to file
func (c *EnvConfig) SaveEnvFile() error {
	// Create backup
	backupPath := c.FilePath + ".backup"
	if err := copyFile(c.FilePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	file, err := os.Create(c.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create env file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	_, _ = writer.WriteString("# DDALAB Environment Configuration\n")
	_, _ = writer.WriteString("# Edited by DDALAB Launcher Configuration Editor\n")
	_, _ = writer.WriteString("\n")

	// Group variables by section
	sectionVars := make(map[string][]EnvVar)
	for _, envVar := range c.Variables {
		section := envVar.Section
		if section == "" {
			section = "General"
		}
		sectionVars[section] = append(sectionVars[section], envVar)
	}

	// Write sections in order
	writtenSections := make(map[string]bool)
	
	// First write known sections in order
	for _, section := range c.Sections {
		if vars, exists := sectionVars[section]; exists {
			c.writeSection(writer, section, vars)
			writtenSections[section] = true
		}
	}

	// Write any remaining sections
	for section, vars := range sectionVars {
		if !writtenSections[section] {
			c.writeSection(writer, section, vars)
		}
	}

	return nil
}

// writeSection writes a section and its variables
func (c *EnvConfig) writeSection(writer *bufio.Writer, section string, vars []EnvVar) {
	if section != "" && section != "General" {
		_, _ = writer.WriteString(fmt.Sprintf("# === %s ===\n", section))
	}

	for _, envVar := range vars {
		// Write comment if exists
		if envVar.Comment != "" {
			_, _ = writer.WriteString(fmt.Sprintf("# %s\n", envVar.Comment))
		}

		// Write the variable
		value := envVar.Value
		if strings.Contains(value, " ") || strings.Contains(value, "#") {
			value = fmt.Sprintf("\"%s\"", value)
		}
		_, _ = writer.WriteString(fmt.Sprintf("%s=%s\n", envVar.Key, value))
	}
	_, _ = writer.WriteString("\n")
}

// GetVariablesBySection returns variables grouped by section
func (c *EnvConfig) GetVariablesBySection() map[string][]EnvVar {
	sectionVars := make(map[string][]EnvVar)
	for _, envVar := range c.Variables {
		section := envVar.Section
		if section == "" {
			section = "General"
		}
		sectionVars[section] = append(sectionVars[section], envVar)
	}
	return sectionVars
}

// UpdateVariable updates an environment variable value
func (c *EnvConfig) UpdateVariable(key, newValue string) bool {
	for i, envVar := range c.Variables {
		if envVar.Key == key {
			c.Variables[i].Value = newValue
			return true
		}
	}
	return false
}

// AddVariable adds a new environment variable
func (c *EnvConfig) AddVariable(envVar EnvVar) {
	c.Variables = append(c.Variables, envVar)
}

// RemoveVariable removes an environment variable
func (c *EnvConfig) RemoveVariable(key string) bool {
	for i, envVar := range c.Variables {
		if envVar.Key == key {
			c.Variables = append(c.Variables[:i], c.Variables[i+1:]...)
			return true
		}
	}
	return false
}

// Helper functions

func isRequiredVar(key, value string) bool {
	requiredVars := []string{
		"DB_PASSWORD", "MINIO_ROOT_PASSWORD", "JWT_SECRET_KEY", 
		"NEXTAUTH_SECRET", "DOMAIN", "PUBLIC_URL",
	}
	
	for _, required := range requiredVars {
		if key == required {
			return true
		}
	}
	
	// Check for placeholder values
	placeholders := []string{
		"CHANGE_ME", "GENERATE_WITH", "YOUR_", "EXAMPLE_",
	}
	
	upperValue := strings.ToUpper(value)
	for _, placeholder := range placeholders {
		if strings.Contains(upperValue, placeholder) {
			return true
		}
	}
	
	return false
}

func isSecretVar(key string) bool {
	secretKeys := []string{
		"PASSWORD", "SECRET", "KEY", "TOKEN", "BIND_PASSWORD",
	}
	
	upperKey := strings.ToUpper(key)
	for _, secret := range secretKeys {
		if strings.Contains(upperKey, secret) {
			return true
		}
	}
	
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// CopyFile is an exported version of copyFile for external use
func CopyFile(src, dst string) error {
	return copyFile(src, dst)
}

// GetEnvFilePath finds the .env file in the DDALAB installation
func GetEnvFilePath(ddalabPath string) (string, error) {
	// Try common locations for .env file
	candidates := []string{
		filepath.Join(ddalabPath, ".env"),
		filepath.Join(ddalabPath, "ddalab-deploy", ".env"),
		filepath.Join(ddalabPath, "deployments", "development-local", ".env"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// If no .env file exists, try to find .env.example and suggest copying
	exampleCandidates := []string{
		filepath.Join(ddalabPath, ".env.example"),
		filepath.Join(ddalabPath, "ddalab-deploy", ".env.example"),
		filepath.Join(ddalabPath, "deployments", "development-local", ".env.example"),
	}

	for _, candidate := range exampleCandidates {
		if _, err := os.Stat(candidate); err == nil {
			envPath := strings.Replace(candidate, ".env.example", ".env", 1)
			return envPath, fmt.Errorf("no .env file found, but .env.example exists at %s. Create .env file first", candidate)
		}
	}

	return "", fmt.Errorf("no .env or .env.example file found in DDALAB installation")
}
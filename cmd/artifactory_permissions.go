package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ArtifactoryPermission represents a permission target in Artifactory
type ArtifactoryPermission struct {
	Name string `json:"name"`
	Repo struct {
		Repositories []string `json:"repositories"`
		Actions      struct {
			Users  map[string][]string `json:"users,omitempty"`
			Groups map[string][]string `json:"groups,omitempty"`
		} `json:"actions"`
	} `json:"repo"`
	Principals struct {
		Users  map[string][]string `json:"users,omitempty"`
		Groups map[string][]string `json:"groups,omitempty"`
	} `json:"principals"`
}

// PermissionOptions holds the configuration for permission management
type PermissionOptions struct {
	ConfigFile     string
	Instance       string
	BaseURL        string
	Username       string
	Password       string
	APIKey         string
	Timeout        time.Duration
	PermissionName string
	Users          []string
	Groups         []string
	Repositories   []string
	Privileges     []string
}

// loadArtifactoryConfig loads Artifactory configuration from the specified file
func loadArtifactoryConfig(configFile string) (map[string]interface{}, error) {
	if configFile == "" {
		return nil, fmt.Errorf("config file path is required")
	}

	// Read the configuration file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", configFile, err)
	}

	// Parse the JSON configuration
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %v", configFile, err)
	}

	// Extract Artifactory configuration
	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		if artifactoryConfig, ok := mcpServers["artifactory"].(map[string]interface{}); ok {
			return artifactoryConfig, nil
		}
	}

	return nil, fmt.Errorf("no Artifactory configuration found in %s", configFile)
}

// getInstanceConfig gets configuration for a specific instance
func getInstanceConfig(config map[string]interface{}, instanceName string) (map[string]interface{}, error) {
	if config == nil {
		return nil, fmt.Errorf("no configuration provided")
	}

	// Get instances from config
	if configData, ok := config["config"].(map[string]interface{}); ok {
		if instances, ok := configData["instances"].(map[string]interface{}); ok {
			if instance, ok := instances[instanceName].(map[string]interface{}); ok {
				return instance, nil
			}
		}
	}

	return nil, fmt.Errorf("instance '%s' not found in configuration", instanceName)
}

// Helper functions to safely extract values from maps
func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if value, ok := m[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getIntFromMap(m map[string]interface{}, key string, defaultValue int) int {
	if value, ok := m[key]; ok {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			// Try to parse as int
			if i, err := fmt.Sscanf(v, "%d", &defaultValue); err == nil && i == 1 {
				return defaultValue
			}
		}
	}
	return defaultValue
}

var artifactoryPermissionsCmd = &cobra.Command{
	Use:   "artifactory-permissions",
	Short: "Manage Artifactory permissions with granular control",
	Long: `Manage Artifactory permissions with granular control over permission names, 
users, groups, and repositories. This tool allows you to create and manage permission 
targets in Artifactory with fine-grained access control.

Examples:
  mcphost artifactory-permissions create --name "dev-permissions" --users "user1,user2" --groups "developers" --repos "repo1,repo2" --privileges "READ,WRITE"
  mcphost artifactory-permissions create --name "admin-permissions" --users "admin1" --privileges "READ,WRITE,DELETE,ANNOTATE,DEPLOY" --repos "ANY"
  mcphost artifactory-permissions list
  mcphost artifactory-permissions delete --name "old-permissions"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

var createPermissionCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new permission target",
	Long: `Create a new permission target in Artifactory with specified users, groups, 
repositories, and privileges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		configFile := getStringFlag(cmd, "config-file", "local.json")
		instanceName := getStringFlag(cmd, "instance", "default")

		// Load Artifactory configuration from file
		artifactoryConfig, err := loadArtifactoryConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Get instance configuration
		instanceConfig, err := getInstanceConfig(artifactoryConfig, instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance configuration: %v", err)
		}

		// Extract configuration values with fallbacks
		baseURL := getStringFromMap(instanceConfig, "url", "http://localhost")
		username := getStringFromMap(instanceConfig, "username", "admin")
		password := getStringFromMap(instanceConfig, "password", "B@55w0rd")
		apiKey := getStringFromMap(instanceConfig, "apiKey", "")
		timeout := getIntFromMap(instanceConfig, "timeout", 30)

		// Override with command line flags if provided
		if cmdFlag := getStringFlag(cmd, "base-url", ""); cmdFlag != "" {
			baseURL = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "username", ""); cmdFlag != "" {
			username = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "password", ""); cmdFlag != "" {
			password = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "api-key", ""); cmdFlag != "" {
			apiKey = cmdFlag
		}
		if cmdFlag := getFloatFlag(cmd, "timeout", 0); cmdFlag > 0 {
			timeout = int(cmdFlag)
		}

		options := &PermissionOptions{
			ConfigFile:     configFile,
			Instance:       instanceName,
			BaseURL:        baseURL,
			Username:       username,
			Password:       password,
			APIKey:         apiKey,
			Timeout:        time.Duration(timeout) * time.Second,
			PermissionName: getStringFlag(cmd, "name", ""),
			Users:          parseCommaSeparated(getStringFlag(cmd, "users", "")),
			Groups:         parseCommaSeparated(getStringFlag(cmd, "groups", "")),
			Repositories:   parseCommaSeparated(getStringFlag(cmd, "repos", "")),
			Privileges:     parseCommaSeparated(getStringFlag(cmd, "privileges", "READ")),
		}

		// Validate timeout
		if options.Timeout > 120*time.Second {
			options.Timeout = 120 * time.Second
		}

		// Validate required parameters
		if options.PermissionName == "" {
			return fmt.Errorf("permission name is required (--name)")
		}

		if len(options.Users) == 0 && len(options.Groups) == 0 {
			return fmt.Errorf("at least one user or group must be specified (--users or --groups)")
		}

		return createPermission(context.Background(), options)
	},
}

var listPermissionsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all permission targets",
	Long:  `List all permission targets in the Artifactory instance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		configFile := getStringFlag(cmd, "config-file", "local.json")
		instanceName := getStringFlag(cmd, "instance", "default")

		// Load Artifactory configuration from file
		artifactoryConfig, err := loadArtifactoryConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Get instance configuration
		instanceConfig, err := getInstanceConfig(artifactoryConfig, instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance configuration: %v", err)
		}

		// Extract configuration values with fallbacks
		baseURL := getStringFromMap(instanceConfig, "url", "http://localhost")
		username := getStringFromMap(instanceConfig, "username", "admin")
		password := getStringFromMap(instanceConfig, "password", "B@55w0rd")
		apiKey := getStringFromMap(instanceConfig, "apiKey", "")
		timeout := getIntFromMap(instanceConfig, "timeout", 30)

		// Override with command line flags if provided
		if cmdFlag := getStringFlag(cmd, "base-url", ""); cmdFlag != "" {
			baseURL = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "username", ""); cmdFlag != "" {
			username = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "password", ""); cmdFlag != "" {
			password = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "api-key", ""); cmdFlag != "" {
			apiKey = cmdFlag
		}
		if cmdFlag := getFloatFlag(cmd, "timeout", 0); cmdFlag > 0 {
			timeout = int(cmdFlag)
		}

		options := &PermissionOptions{
			ConfigFile: configFile,
			Instance:   instanceName,
			BaseURL:    baseURL,
			Username:   username,
			Password:   password,
			APIKey:     apiKey,
			Timeout:    time.Duration(timeout) * time.Second,
		}

		// Validate timeout
		if options.Timeout > 120*time.Second {
			options.Timeout = 120 * time.Second
		}

		return listPermissions(context.Background(), options)
	},
}

var deletePermissionCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a permission target",
	Long:  `Delete a permission target from Artifactory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		configFile := getStringFlag(cmd, "config-file", "local.json")
		instanceName := getStringFlag(cmd, "instance", "default")

		// Load Artifactory configuration from file
		artifactoryConfig, err := loadArtifactoryConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Get instance configuration
		instanceConfig, err := getInstanceConfig(artifactoryConfig, instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance configuration: %v", err)
		}

		// Extract configuration values with fallbacks
		baseURL := getStringFromMap(instanceConfig, "url", "http://localhost")
		username := getStringFromMap(instanceConfig, "username", "admin")
		password := getStringFromMap(instanceConfig, "password", "B@55w0rd")
		apiKey := getStringFromMap(instanceConfig, "apiKey", "")
		timeout := getIntFromMap(instanceConfig, "timeout", 30)

		// Override with command line flags if provided
		if cmdFlag := getStringFlag(cmd, "base-url", ""); cmdFlag != "" {
			baseURL = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "username", ""); cmdFlag != "" {
			username = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "password", ""); cmdFlag != "" {
			password = cmdFlag
		}
		if cmdFlag := getStringFlag(cmd, "api-key", ""); cmdFlag != "" {
			apiKey = cmdFlag
		}
		if cmdFlag := getFloatFlag(cmd, "timeout", 0); cmdFlag > 0 {
			timeout = int(cmdFlag)
		}

		options := &PermissionOptions{
			ConfigFile:     configFile,
			Instance:       instanceName,
			BaseURL:        baseURL,
			Username:       username,
			Password:       password,
			APIKey:         apiKey,
			Timeout:        time.Duration(timeout) * time.Second,
			PermissionName: getStringFlag(cmd, "name", ""),
		}

		// Validate timeout
		if options.Timeout > 120*time.Second {
			options.Timeout = 120 * time.Second
		}

		// Validate required parameters
		if options.PermissionName == "" {
			return fmt.Errorf("permission name is required (--name)")
		}

		return deletePermission(context.Background(), options)
	},
}

func init() {
	// Add subcommands
	artifactoryPermissionsCmd.AddCommand(createPermissionCmd)
	artifactoryPermissionsCmd.AddCommand(listPermissionsCmd)
	artifactoryPermissionsCmd.AddCommand(deletePermissionCmd)

	// Common flags for all commands
	artifactoryPermissionsCmd.PersistentFlags().String("config-file", "local.json", "Configuration file path")
	artifactoryPermissionsCmd.PersistentFlags().String("instance", "default", "Artifactory instance name from configuration")
	artifactoryPermissionsCmd.PersistentFlags().String("base-url", "", "Artifactory base URL (overrides configuration)")
	artifactoryPermissionsCmd.PersistentFlags().String("username", "", "Username for authentication (overrides configuration)")
	artifactoryPermissionsCmd.PersistentFlags().String("password", "", "Password for authentication (overrides configuration)")
	artifactoryPermissionsCmd.PersistentFlags().String("api-key", "", "API key for authentication (overrides configuration)")
	artifactoryPermissionsCmd.PersistentFlags().Float64("timeout", 0, "Timeout in seconds (max 120, overrides configuration)")

	// Create command specific flags
	createPermissionCmd.Flags().String("name", "", "Permission target name (required)")
	createPermissionCmd.Flags().String("users", "", "Comma-separated list of users")
	createPermissionCmd.Flags().String("groups", "", "Comma-separated list of groups")
	createPermissionCmd.Flags().String("repos", "", "Comma-separated list of repositories (use 'ANY' for all repositories)")
	createPermissionCmd.Flags().String("privileges", "READ", "Comma-separated list of privileges (READ, WRITE, DELETE, ANNOTATE, DEPLOY, etc.)")

	// Delete command specific flags
	deletePermissionCmd.Flags().String("name", "", "Permission target name to delete (required)")

	// Mark required flags
	createPermissionCmd.MarkFlagRequired("name")
	deletePermissionCmd.MarkFlagRequired("name")
}

func createPermission(ctx context.Context, options *PermissionOptions) error {
	// Validate and construct the base URL
	parsedURL, err := url.Parse(options.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %v", err)
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		options.BaseURL = "http://" + options.BaseURL
		parsedURL, err = url.Parse(options.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid URL after adding http: %v", err)
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http:// or https://")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: options.Timeout,
	}

	// Create permission target
	permission := ArtifactoryPermission{
		Name: options.PermissionName,
	}

	// Set repositories
	if len(options.Repositories) > 0 {
		permission.Repo.Repositories = options.Repositories
	} else {
		// Use "ANY" for all repositories
		permission.Repo.Repositories = []string{"ANY"}
	}

	// Set users and their privileges
	if len(options.Users) > 0 {
		permission.Principals.Users = make(map[string][]string)
		permission.Repo.Actions.Users = make(map[string][]string)
		for _, user := range options.Users {
			user = strings.TrimSpace(user)
			if user != "" {
				permission.Principals.Users[user] = options.Privileges
				permission.Repo.Actions.Users[user] = options.Privileges
			}
		}
	}

	// Set groups and their privileges
	if len(options.Groups) > 0 {
		permission.Principals.Groups = make(map[string][]string)
		permission.Repo.Actions.Groups = make(map[string][]string)
		for _, group := range options.Groups {
			group = strings.TrimSpace(group)
			if group != "" {
				permission.Principals.Groups[group] = options.Privileges
				permission.Repo.Actions.Groups[group] = options.Privileges
			}
		}
	}

	// Create permission JSON
	permissionJSON, err := json.Marshal(permission)
	if err != nil {
		return fmt.Errorf("failed to marshal permission data: %v", err)
	}

	// Create permission request
	permissionURL := fmt.Sprintf("%s/artifactory/api/security/permissions/%s", options.BaseURL, options.PermissionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", permissionURL, bytes.NewBuffer(permissionJSON))
	if err != nil {
		return fmt.Errorf("failed to create permission request: %v", err)
	}

	// Set authentication headers
	if options.APIKey != "" {
		req.Header.Set("X-JFrog-Art-Api", options.APIKey)
	} else if options.Username != "" && options.Password != "" {
		req.SetBasicAuth(options.Username, options.Password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Permissions/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create permission: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := json.Marshal(resp.Body)
		return fmt.Errorf("failed to create permission: status %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// Return success result
	result := map[string]interface{}{
		"permission": map[string]interface{}{
			"created":      true,
			"name":         options.PermissionName,
			"users":        options.Users,
			"groups":       options.Groups,
			"repositories": options.Repositories,
			"privileges":   options.Privileges,
		},
		"url":       permissionURL,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %v", err)
	}

	fmt.Printf("✅ Permission target created successfully:\n%s\n", string(resultJSON))
	return nil
}

func listPermissions(ctx context.Context, options *PermissionOptions) error {
	// Validate and construct the base URL
	parsedURL, err := url.Parse(options.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %v", err)
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		options.BaseURL = "http://" + options.BaseURL
		parsedURL, err = url.Parse(options.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid URL after adding http: %v", err)
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http:// or https://")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: options.Timeout,
	}

	// Create request
	permissionsURL := fmt.Sprintf("%s/artifactory/api/security/permissions", options.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", permissionsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set authentication headers
	if options.APIKey != "" {
		req.Header.Set("X-JFrog-Art-Api", options.APIKey)
	} else if options.Username != "" && options.Password != "" {
		req.SetBasicAuth(options.Username, options.Password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Permissions/1.0")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to list permissions: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := json.Marshal(resp.Body)
		return fmt.Errorf("failed to list permissions: status %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var permissions []ArtifactoryPermission
	if err := json.NewDecoder(resp.Body).Decode(&permissions); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	// Display results
	if len(permissions) == 0 {
		fmt.Println("📋 No permission targets found")
		return nil
	}

	fmt.Printf("📋 Found %d permission targets:\n\n", len(permissions))
	for i, permission := range permissions {
		fmt.Printf("%d. %s\n", i+1, permission.Name)
		fmt.Printf("   Repositories: %s\n", strings.Join(permission.Repo.Repositories, ", "))

		if len(permission.Principals.Users) > 0 {
			fmt.Printf("   Users:\n")
			for user, privs := range permission.Principals.Users {
				fmt.Printf("     - %s: %s\n", user, strings.Join(privs, ", "))
			}
		}

		if len(permission.Principals.Groups) > 0 {
			fmt.Printf("   Groups:\n")
			for group, privs := range permission.Principals.Groups {
				fmt.Printf("     - %s: %s\n", group, strings.Join(privs, ", "))
			}
		}
		fmt.Println()
	}

	return nil
}

func deletePermission(ctx context.Context, options *PermissionOptions) error {
	// Validate and construct the base URL
	parsedURL, err := url.Parse(options.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %v", err)
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		options.BaseURL = "http://" + options.BaseURL
		parsedURL, err = url.Parse(options.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid URL after adding http: %v", err)
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http:// or https://")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: options.Timeout,
	}

	// Create request
	permissionURL := fmt.Sprintf("%s/artifactory/api/security/permissions/%s", options.BaseURL, options.PermissionName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", permissionURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set authentication headers
	if options.APIKey != "" {
		req.Header.Set("X-JFrog-Art-Api", options.APIKey)
	} else if options.Username != "" && options.Password != "" {
		req.SetBasicAuth(options.Username, options.Password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Permissions/1.0")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := json.Marshal(resp.Body)
		return fmt.Errorf("failed to delete permission: status %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("✅ Permission target '%s' deleted successfully\n", options.PermissionName)
	return nil
}

// Helper functions
func getStringFlag(cmd *cobra.Command, name, defaultValue string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

func getFloatFlag(cmd *cobra.Command, name string, defaultValue float64) float64 {
	value, err := cmd.Flags().GetFloat64(name)
	if err != nil {
		return defaultValue
	}
	return value
}

func parseCommaSeparated(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

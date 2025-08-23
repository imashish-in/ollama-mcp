package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bytes"
	"io"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	artifactoryDefaultTimeout = 30 * time.Second
	artifactoryMaxTimeout     = 120 * time.Second
)

// ArtifactoryHealthResponse represents the response from Artifactory health check
type ArtifactoryHealthResponse struct {
	Router struct {
		NodeID  string `json:"node_id"`
		State   string `json:"state"`
		Message string `json:"message"`
	} `json:"router"`
	Services []struct {
		ServiceID string `json:"service_id"`
		NodeID    string `json:"node_id"`
		State     string `json:"state"`
		Message   string `json:"message"`
	} `json:"services"`
}

// ArtifactoryRepository represents a repository in Artifactory
type ArtifactoryRepository struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	PackageType string `json:"packageType,omitempty"`
}

// ArtifactoryRepositoriesResponse represents the response from Artifactory repositories API
type ArtifactoryRepositoriesResponse struct {
	Repositories []ArtifactoryRepository `json:"repositories"`
}

// ArtifactoryUser represents a user in Artifactory
type ArtifactoryUser struct {
	Name                     string   `json:"name"`
	Email                    string   `json:"email,omitempty"`
	Admin                    bool     `json:"admin,omitempty"`
	ProfileUpdatable         bool     `json:"profileUpdatable,omitempty"`
	InternalPasswordDisabled bool     `json:"internalPasswordDisabled,omitempty"`
	LastLoggedIn             string   `json:"lastLoggedIn,omitempty"`
	Realm                    string   `json:"realm,omitempty"`
	Groups                   []string `json:"groups,omitempty"`
}

// ArtifactoryUsersResponse represents the response from Artifactory users API
type ArtifactoryUsersResponse struct {
	Users []ArtifactoryUser `json:"users"`
}

// ArtifactoryRepositorySize represents repository size information
type ArtifactoryRepositorySize struct {
	Key           string `json:"key"`
	Type          string `json:"type"`
	PackageType   string `json:"packageType,omitempty"`
	Description   string `json:"description,omitempty"`
	URL           string `json:"url,omitempty"`
	Size          int64  `json:"size,omitempty"`
	SizeFormatted string `json:"sizeFormatted,omitempty"`
	FileCount     int64  `json:"fileCount,omitempty"`
	FolderCount   int64  `json:"folderCount,omitempty"`
	ItemsCount    int64  `json:"itemsCount,omitempty"`
}

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

// ArtifactoryGroup represents a group in Artifactory
type ArtifactoryGroup struct {
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	AutoJoin        bool     `json:"autoJoin,omitempty"`
	AdminPrivileges bool     `json:"adminPrivileges,omitempty"`
	Realm           string   `json:"realm,omitempty"`
	RealmAttributes string   `json:"realmAttributes,omitempty"`
	UsersNames      []string `json:"usersNames,omitempty"`
}

// ArtifactoryCreateRepository represents a repository creation request
type ArtifactoryCreateRepository struct {
	Key         string `json:"key"`
	Rclass      string `json:"rclass"` // LOCAL, REMOTE, or VIRTUAL
	PackageType string `json:"packageType,omitempty"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`
	// For REMOTE repositories
	URL                            string `json:"url,omitempty"`
	Username                       string `json:"username,omitempty"`
	Password                       string `json:"password,omitempty"`
	Proxy                          string `json:"proxy,omitempty"`
	RemoteRepoLayoutRef            string `json:"remoteRepoLayoutRef,omitempty"`
	HardFail                       bool   `json:"hardFail,omitempty"`
	Offline                        bool   `json:"offline,omitempty"`
	StoreArtifactsLocally          bool   `json:"storeArtifactsLocally,omitempty"`
	SocketTimeoutMillis            int    `json:"socketTimeoutMillis,omitempty"`
	LocalAddress                   string `json:"localAddress,omitempty"`
	RetrievalCachePeriodSecs       int    `json:"retrievalCachePeriodSecs,omitempty"`
	FailedRetrievalCachePeriodSecs int    `json:"failedRetrievalCachePeriodSecs,omitempty"`
	MissedRetrievalCachePeriodSecs int    `json:"missedRetrievalCachePeriodSecs,omitempty"`
	// For VIRTUAL repositories
	Repositories          []string `json:"repositories,omitempty"`
	DefaultDeploymentRepo string   `json:"defaultDeploymentRepo,omitempty"`
	// Common settings
	BlackedOut                   bool              `json:"blackedOut,omitempty"`
	HandleReleases               bool              `json:"handleReleases,omitempty"`
	HandleSnapshots              bool              `json:"handleSnapshots,omitempty"`
	MaxUniqueSnapshots           int               `json:"maxUniqueSnapshots,omitempty"`
	SuppressPomConsistencyChecks bool              `json:"suppressPomConsistencyChecks,omitempty"`
	PropertySets                 []string          `json:"propertySets,omitempty"`
	ArchiveBrowsingEnabled       bool              `json:"archiveBrowsingEnabled,omitempty"`
	CustomProperties             map[string]string `json:"customProperties,omitempty"`
}

// NewArtifactoryServer creates a new Artifactory MCP server
func NewArtifactoryServer() (*server.MCPServer, error) {
	s := server.NewMCPServer("artifactory-server", "1.0.0", server.WithToolCapabilities(true))

	// Register the healthcheck tool
	healthcheckTool := mcp.NewTool("artifactory_healthcheck",
		mcp.WithDescription("Check the health status of an Artifactory instance using the /router/api/v1/system/health endpoint. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	// Register the repositories tool
	repositoriesTool := mcp.NewTool("artifactory_get_repositories",
		mcp.WithDescription("Get a list of all repositories in an Artifactory instance using the /artifactory/api/repositories endpoint. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	// Register the users tool
	usersTool := mcp.NewTool("artifactory_get_users",
		mcp.WithDescription("Get a list of all users in an Artifactory instance using the /artifactory/api/security/users endpoint. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	// Register the create user tool
	createUserTool := mcp.NewTool("artifactory_create_user",
		mcp.WithDescription("Create a new user in an Artifactory instance using the /artifactory/api/security/users/{username} endpoint. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithString("new_username",
			mcp.Description("The username for the new user to create (required)"),
		),
		mcp.WithString("new_password",
			mcp.Description("The password for the new user (required)"),
		),
		mcp.WithString("email",
			mcp.Description("Email address for the new user (optional)"),
		),
		mcp.WithBoolean("admin",
			mcp.Description("Whether the new user should have admin privileges (optional, defaults to false)"),
		),
		mcp.WithString("realm",
			mcp.Description("The realm for the new user (optional, defaults to 'internal')"),
		),
		mcp.WithString("groups",
			mcp.Description("Comma-separated list of groups for the new user (optional)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	// Register the repository sizes tool
	repositorySizesTool := mcp.NewTool("artifactory_get_repository_sizes",
		mcp.WithDescription("Get size information for all repositories in an Artifactory instance using the /artifactory/api/storageinfo endpoint. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	// Register the permission group management tool
	permissionGroupTool := mcp.NewTool("artifactory_manage_permission_group",
		mcp.WithDescription("Create or update a permission group in Artifactory with users, permissions, and repository access. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithString("group_name",
			mcp.Description("The name of the permission group to create or update (required)"),
		),
		mcp.WithString("group_description",
			mcp.Description("Description of the permission group (optional)"),
		),
		mcp.WithString("users",
			mcp.Description("Comma-separated list of usernames to add to the group (optional)"),
		),
		mcp.WithString("repositories",
			mcp.Description("Comma-separated list of repository names to grant access to (optional)"),
		),
		mcp.WithString("privileges",
			mcp.Description("Comma-separated list of privileges (READ, WRITE, DELETE, ANNOTATE, DEPLOY, etc.) (optional, defaults to READ)"),
		),
		mcp.WithBoolean("auto_join",
			mcp.Description("Whether users can automatically join this group (optional, defaults to false)"),
		),
		mcp.WithBoolean("admin_privileges",
			mcp.Description("Whether the group has admin privileges (optional, defaults to false)"),
		),
		mcp.WithString("realm",
			mcp.Description("The realm for the group (optional, defaults to 'internal')"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	// Register the repository creation tool
	createRepositoryTool := mcp.NewTool("artifactory_create_repository",
		mcp.WithDescription("Create a new repository in Artifactory. Supports LOCAL, REMOTE, and VIRTUAL repository types. Defaults to localhost with admin credentials if no parameters provided."),
		mcp.WithString("base_url",
			mcp.Description("The base URL of the Artifactory instance (e.g., http://localhost, https://artifactory.example.com). Defaults to http://localhost if not provided."),
		),
		mcp.WithString("username",
			mcp.Description("Username for authentication (optional, defaults to 'admin')"),
		),
		mcp.WithString("password",
			mcp.Description("Password for authentication (optional, defaults to 'B@55w0rd')"),
		),
		mcp.WithString("api_key",
			mcp.Description("API key for authentication (alternative to username/password)"),
		),
		mcp.WithString("repo_key",
			mcp.Description("The unique key/name for the repository (required)"),
		),
		mcp.WithString("repo_type",
			mcp.Description("Repository type: LOCAL, REMOTE, or VIRTUAL (required)"),
		),
		mcp.WithString("package_type",
			mcp.Description("Package type: Generic, Maven, Gradle, Ivy, Sbt, NuGet, Gems, Npm, Bower, Debian, Composer, PyPI, Docker, GitLfs, YUM, Conan, Chef, Puppet, Helm, Go, P2, R, Swift, CocoaPods, Opkg, Vagrant, Cran, Conda, P2, VCS, etc. (required)"),
		),
		mcp.WithString("description",
			mcp.Description("Description of the repository (optional)"),
		),
		mcp.WithString("notes",
			mcp.Description("Additional notes about the repository (optional)"),
		),
		// Remote repository specific parameters
		mcp.WithString("remote_url",
			mcp.Description("URL of the remote repository (required for REMOTE type)"),
		),
		mcp.WithString("remote_username",
			mcp.Description("Username for remote repository authentication (optional)"),
		),
		mcp.WithString("remote_password",
			mcp.Description("Password for remote repository authentication (optional)"),
		),
		mcp.WithString("proxy",
			mcp.Description("Proxy configuration name (optional)"),
		),
		// Virtual repository specific parameters
		mcp.WithString("virtual_repositories",
			mcp.Description("Comma-separated list of repository keys to include in virtual repository (required for VIRTUAL type)"),
		),
		mcp.WithString("default_deployment_repo",
			mcp.Description("Default deployment repository for virtual repository (optional)"),
		),
		// Common repository settings
		mcp.WithBoolean("handle_releases",
			mcp.Description("Whether to handle release artifacts (optional, defaults to true)"),
		),
		mcp.WithBoolean("handle_snapshots",
			mcp.Description("Whether to handle snapshot artifacts (optional, defaults to true)"),
		),
		mcp.WithBoolean("suppress_pom_consistency_checks",
			mcp.Description("Whether to suppress POM consistency checks (optional, defaults to false)"),
		),
		mcp.WithBoolean("blacked_out",
			mcp.Description("Whether the repository is blacked out (optional, defaults to false)"),
		),
		mcp.WithBoolean("archive_browsing_enabled",
			mcp.Description("Whether archive browsing is enabled (optional, defaults to false)"),
		),
		mcp.WithNumber("max_unique_snapshots",
			mcp.Description("Maximum number of unique snapshots to keep (optional, defaults to 0)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds (max 120)"),
			mcp.Min(0),
			mcp.Max(120),
		),
	)

	s.AddTool(healthcheckTool, executeArtifactoryHealthcheck)
	s.AddTool(repositoriesTool, executeArtifactoryGetRepositories)
	s.AddTool(usersTool, executeArtifactoryGetUsers)
	s.AddTool(createUserTool, executeArtifactoryCreateUser)
	s.AddTool(repositorySizesTool, executeArtifactoryGetRepositorySizes)
	s.AddTool(permissionGroupTool, executeArtifactoryManagePermissionGroup)
	s.AddTool(createRepositoryTool, executeArtifactoryCreateRepository)

	return s, nil
}

// executeArtifactoryHealthcheck handles the healthcheck tool execution
func executeArtifactoryHealthcheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")

	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Parse timeout (optional)
	timeout := artifactoryDefaultTimeout
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > artifactoryMaxTimeout {
			timeout = artifactoryMaxTimeout
		} else {
			timeout = timeoutDuration
		}
	}

	// Validate and construct the health check URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid base URL: %v", err)), nil
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		baseURL = "http://" + baseURL
		parsedURL, err = url.Parse(baseURL)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid URL after adding http: %v", err)), nil
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return mcp.NewToolResultError("URL must use http:// or https://"), nil
	}

	// Construct the health check endpoint URL
	healthURL := fmt.Sprintf("%s/router/api/v1/system/health", baseURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		req.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Healthcheck/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute request: %v", err)), nil
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Artifactory health check failed with status code: %d", resp.StatusCode)), nil
	}

	// Parse the response
	var healthResponse ArtifactoryHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	// Check if Artifactory reports healthy status
	if healthResponse.Router.State != "HEALTHY" {
		errorMsg := "Artifactory is not healthy"
		if healthResponse.Router.Message != "" {
			errorMsg = fmt.Sprintf("Artifactory is not healthy: %s", healthResponse.Router.Message)
		}
		return mcp.NewToolResultError(errorMsg), nil
	}

	// Check if all services are healthy
	unhealthyServices := []string{}
	for _, service := range healthResponse.Services {
		if service.State != "HEALTHY" {
			unhealthyServices = append(unhealthyServices, service.ServiceID)
		}
	}

	if len(unhealthyServices) > 0 {
		errorMsg := fmt.Sprintf("Artifactory has unhealthy services: %s", strings.Join(unhealthyServices, ", "))
		return mcp.NewToolResultError(errorMsg), nil
	}

	// Return success result
	result := map[string]interface{}{
		"status":             "healthy",
		"artifactory_status": healthResponse.Router.State,
		"router_message":     healthResponse.Router.Message,
		"services_count":     len(healthResponse.Services),
		"url":                healthURL,
		"timestamp":          time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// executeArtifactoryGetRepositories handles the repositories tool execution
func executeArtifactoryGetRepositories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")

	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Parse timeout (optional)
	timeout := artifactoryDefaultTimeout
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > artifactoryMaxTimeout {
			timeout = artifactoryMaxTimeout
		} else {
			timeout = timeoutDuration
		}
	}

	// Validate and construct the repositories API URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid base URL: %v", err)), nil
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		baseURL = "http://" + baseURL
		parsedURL, err = url.Parse(baseURL)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid URL after adding http: %v", err)), nil
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return mcp.NewToolResultError("URL must use http:// or https://"), nil
	}

	// Construct the repositories endpoint URL
	repositoriesURL := fmt.Sprintf("%s/artifactory/api/repositories", baseURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", repositoriesURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		req.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Repositories/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute request: %v", err)), nil
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Artifactory repositories request failed with status code: %d", resp.StatusCode)), nil
	}

	// Parse the response - Artifactory returns an array directly
	var repositories []ArtifactoryRepository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	// Return success result
	result := map[string]interface{}{
		"repositories": repositories,
		"count":        len(repositories),
		"url":          repositoriesURL,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// executeArtifactoryGetUsers handles the users tool execution
func executeArtifactoryGetUsers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")

	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Parse timeout (optional)
	timeout := artifactoryDefaultTimeout
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > artifactoryMaxTimeout {
			timeout = artifactoryMaxTimeout
		} else {
			timeout = timeoutDuration
		}
	}

	// Validate and construct the users API URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid base URL: %v", err)), nil
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		baseURL = "http://" + baseURL
		parsedURL, err = url.Parse(baseURL)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid URL after adding http: %v", err)), nil
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return mcp.NewToolResultError("URL must use http:// or https://"), nil
	}

	// Construct the users endpoint URL
	usersURL := fmt.Sprintf("%s/artifactory/api/security/users", baseURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", usersURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		req.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Users/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute request: %v", err)), nil
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Artifactory users request failed with status code: %d", resp.StatusCode)), nil
	}

	// Parse the response - Artifactory returns an array directly
	var users []ArtifactoryUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	// Return success result
	result := map[string]interface{}{
		"users":     users,
		"count":     len(users),
		"url":       usersURL,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// executeArtifactoryCreateUser handles the create user tool execution
func executeArtifactoryCreateUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")

	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Extract new user parameters
	newUsername := request.GetString("new_username", "")
	if newUsername == "" {
		return mcp.NewToolResultError("new_username is required"), nil
	}

	newPassword := request.GetString("new_password", "")
	if newPassword == "" {
		return mcp.NewToolResultError("new_password is required"), nil
	}

	email := request.GetString("email", "")
	admin := request.GetBool("admin", false)
	realm := request.GetString("realm", "internal")
	groupsStr := request.GetString("groups", "")

	// Parse groups if provided
	var groups []string
	if groupsStr != "" {
		groups = strings.Split(groupsStr, ",")
		for i, group := range groups {
			groups[i] = strings.TrimSpace(group)
		}
	}

	// Parse timeout (optional)
	timeout := artifactoryDefaultTimeout
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > artifactoryMaxTimeout {
			timeout = artifactoryMaxTimeout
		} else {
			timeout = timeoutDuration
		}
	}

	// Validate and construct the create user API URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid base URL: %v", err)), nil
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		baseURL = "http://" + baseURL
		parsedURL, err = url.Parse(baseURL)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid URL after adding http: %v", err)), nil
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return mcp.NewToolResultError("URL must use http:// or https://"), nil
	}

	// Construct the create user endpoint URL
	createUserURL := fmt.Sprintf("%s/artifactory/api/security/users/%s", baseURL, newUsername)

	// Create user data
	userData := map[string]interface{}{
		"name":     newUsername,
		"password": newPassword,
		"admin":    admin,
		"realm":    realm,
	}

	if email != "" {
		userData["email"] = email
	}

	if len(groups) > 0 {
		userData["groups"] = groups
	}

	// Marshal user data to JSON
	userDataJSON, err := json.Marshal(userData)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal user data: %v", err)), nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "PUT", createUserURL, bytes.NewBuffer(userDataJSON))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		req.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-CreateUser/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute request: %v", err)), nil
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("Artifactory create user request failed with status code: %d, response: %s", resp.StatusCode, string(bodyBytes))), nil
	}

	// Return success result
	result := map[string]interface{}{
		"message":     "User created successfully",
		"username":    newUsername,
		"email":       email,
		"admin":       admin,
		"realm":       realm,
		"groups":      groups,
		"url":         createUserURL,
		"status_code": resp.StatusCode,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// executeArtifactoryGetRepositorySizes handles the repository sizes tool execution
func executeArtifactoryGetRepositorySizes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")

	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Parse timeout (optional)
	timeout := artifactoryDefaultTimeout
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > artifactoryMaxTimeout {
			timeout = artifactoryMaxTimeout
		} else {
			timeout = timeoutDuration
		}
	}

	// Validate and construct the storage info API URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid base URL: %v", err)), nil
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		baseURL = "http://" + baseURL
		parsedURL, err = url.Parse(baseURL)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid URL after adding http: %v", err)), nil
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return mcp.NewToolResultError("URL must use http:// or https://"), nil
	}

	// First, get the list of repositories
	repositoriesURL := fmt.Sprintf("%s/artifactory/api/repositories", baseURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request for repositories list
	req, err := http.NewRequestWithContext(ctx, "GET", repositoriesURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create repositories request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		req.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Artifactory-Repositories/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute the repositories request
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute repositories request: %v", err)), nil
	}
	defer resp.Body.Close()

	// Check HTTP status code for repositories
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Artifactory repositories request failed with status code: %d", resp.StatusCode)), nil
	}

	// Parse the repositories response
	var repositories []ArtifactoryRepository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse repositories response: %v", err)), nil
	}

	// Get storage info for each repository
	var repositorySizes []ArtifactoryRepositorySize
	totalSize := int64(0)
	totalFiles := int64(0)
	totalFolders := int64(0)
	totalItems := int64(0)

	for _, repo := range repositories {
		// Get storage info for this repository
		storageURL := fmt.Sprintf("%s/artifactory/api/storage/%s", baseURL, repo.Key)

		storageReq, err := http.NewRequestWithContext(ctx, "GET", storageURL, nil)
		if err != nil {
			// Skip this repository if we can't create the request
			continue
		}

		// Set authentication headers
		if apiKey != "" {
			storageReq.Header.Set("X-JFrog-Art-Api", apiKey)
		} else if username != "" && password != "" {
			storageReq.SetBasicAuth(username, password)
		}

		// Set headers
		storageReq.Header.Set("User-Agent", "MCP-Artifactory-Storage/1.0")
		storageReq.Header.Set("Accept", "application/json")
		storageReq.Header.Set("Content-Type", "application/json")

		// Execute the storage request
		storageResp, err := client.Do(storageReq)
		if err != nil {
			// Skip this repository if the request fails
			continue
		}

		// Parse storage info
		var storageInfo map[string]interface{}
		if err := json.NewDecoder(storageResp.Body).Decode(&storageInfo); err != nil {
			storageResp.Body.Close()
			continue
		}
		storageResp.Body.Close()

		// Extract size information
		repoSize := ArtifactoryRepositorySize{
			Key:         repo.Key,
			Type:        repo.Type,
			PackageType: repo.PackageType,
			Description: repo.Description,
			URL:         repo.URL,
		}

		// Extract size data from storage info
		if size, ok := storageInfo["size"].(float64); ok {
			repoSize.Size = int64(size)
			repoSize.SizeFormatted = formatBytes(int64(size))
			totalSize += int64(size)
		}

		if files, ok := storageInfo["filesCount"].(float64); ok {
			repoSize.FileCount = int64(files)
			totalFiles += int64(files)
		}

		if folders, ok := storageInfo["foldersCount"].(float64); ok {
			repoSize.FolderCount = int64(folders)
			totalFolders += int64(folders)
		}

		if items, ok := storageInfo["itemsCount"].(float64); ok {
			repoSize.ItemsCount = int64(items)
			totalItems += int64(items)
		}

		repositorySizes = append(repositorySizes, repoSize)
	}

	// Return success result
	result := map[string]interface{}{
		"repositories":       repositorySizes,
		"count":              len(repositorySizes),
		"totalSize":          totalSize,
		"totalSizeFormatted": formatBytes(totalSize),
		"totalFiles":         totalFiles,
		"totalFolders":       totalFolders,
		"totalItems":         totalItems,
		"url":                repositoriesURL,
		"timestamp":          time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// executeArtifactoryCreateRepository handles the repository creation tool execution
func executeArtifactoryCreateRepository(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")
	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Extract repository parameters
	repoKey := request.GetString("repo_key", "")
	repoType := request.GetString("repo_type", "")
	packageType := request.GetString("package_type", "")
	description := request.GetString("description", "")
	notes := request.GetString("notes", "")

	// Validate required parameters
	if repoKey == "" {
		return mcp.NewToolResultError("repo_key is required"), nil
	}
	if repoType == "" {
		return mcp.NewToolResultError("repo_type is required (LOCAL, REMOTE, or VIRTUAL)"), nil
	}
	if packageType == "" {
		return mcp.NewToolResultError("package_type is required"), nil
	}

	// Validate repository type
	repoType = strings.ToUpper(repoType)
	if repoType != "LOCAL" && repoType != "REMOTE" && repoType != "VIRTUAL" {
		return mcp.NewToolResultError("repo_type must be LOCAL, REMOTE, or VIRTUAL"), nil
	}

	// Create repository configuration
	repo := ArtifactoryCreateRepository{
		Key:         repoKey,
		Rclass:      repoType,
		PackageType: packageType,
		Description: description,
		Notes:       notes,
	}

	// Set common settings
	repo.HandleReleases = request.GetBool("handle_releases", true)
	repo.HandleSnapshots = request.GetBool("handle_snapshots", true)
	repo.SuppressPomConsistencyChecks = request.GetBool("suppress_pom_consistency_checks", false)
	repo.BlackedOut = request.GetBool("blacked_out", false)
	repo.ArchiveBrowsingEnabled = request.GetBool("archive_browsing_enabled", false)

	if maxSnapshots := request.GetFloat("max_unique_snapshots", 0); maxSnapshots > 0 {
		repo.MaxUniqueSnapshots = int(maxSnapshots)
	}

	// Set type-specific parameters
	switch repoType {
	case "REMOTE":
		remoteURL := request.GetString("remote_url", "")
		if remoteURL == "" {
			return mcp.NewToolResultError("remote_url is required for REMOTE repositories"), nil
		}
		repo.URL = remoteURL
		repo.Username = request.GetString("remote_username", "")
		repo.Password = request.GetString("remote_password", "")
		repo.Proxy = request.GetString("proxy", "")
		repo.RemoteRepoLayoutRef = "simple-default"
		repo.HardFail = false
		repo.Offline = false
		repo.StoreArtifactsLocally = true
		repo.SocketTimeoutMillis = 15000
		repo.RetrievalCachePeriodSecs = 7200
		repo.FailedRetrievalCachePeriodSecs = 30
		repo.MissedRetrievalCachePeriodSecs = 7200

	case "VIRTUAL":
		virtualReposStr := request.GetString("virtual_repositories", "")
		if virtualReposStr == "" {
			return mcp.NewToolResultError("virtual_repositories is required for VIRTUAL repositories"), nil
		}
		repos := strings.Split(virtualReposStr, ",")
		for i, r := range repos {
			repos[i] = strings.TrimSpace(r)
		}
		repo.Repositories = repos
		repo.DefaultDeploymentRepo = request.GetString("default_deployment_repo", "")
	}

	// Create HTTP client with timeout
	timeout := 30 * time.Second
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > 120*time.Second {
			timeout = 120 * time.Second
		} else {
			timeout = timeoutDuration
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// Create repository JSON
	repoJSON, err := json.Marshal(repo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal repository data: %v", err)), nil
	}

	// Create repository request
	repoURL := fmt.Sprintf("%s/artifactory/api/repositories/%s", baseURL, repoKey)
	repoReq, err := http.NewRequestWithContext(ctx, "PUT", repoURL, bytes.NewBuffer(repoJSON))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create repository request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		repoReq.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		repoReq.SetBasicAuth(username, password)
	}

	// Set headers
	repoReq.Header.Set("User-Agent", "MCP-Artifactory-Repository/1.0")
	repoReq.Header.Set("Accept", "application/json")
	repoReq.Header.Set("Content-Type", "application/json")

	// Execute repository creation request
	repoResp, err := client.Do(repoReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create repository: %v", err)), nil
	}
	defer repoResp.Body.Close()

	// Check repository creation status
	if repoResp.StatusCode != http.StatusOK && repoResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(repoResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("failed to create repository: status %d, response: %s", repoResp.StatusCode, string(bodyBytes))), nil
	}

	// Return success result
	result := map[string]interface{}{
		"repository": map[string]interface{}{
			"key":         repoKey,
			"type":        repoType,
			"packageType": packageType,
			"description": description,
			"notes":       notes,
			"url":         fmt.Sprintf("%s/artifactory/%s", baseURL, repoKey),
			"created":     true,
		},
		"settings": map[string]interface{}{
			"handleReleases":         repo.HandleReleases,
			"handleSnapshots":        repo.HandleSnapshots,
			"blackedOut":             repo.BlackedOut,
			"archiveBrowsingEnabled": repo.ArchiveBrowsingEnabled,
		},
	}

	if repoType == "REMOTE" {
		result["remote"] = map[string]interface{}{
			"url":      repo.URL,
			"username": repo.Username,
			"proxy":    repo.Proxy,
		}
	} else if repoType == "VIRTUAL" {
		result["virtual"] = map[string]interface{}{
			"repositories":          repo.Repositories,
			"defaultDeploymentRepo": repo.DefaultDeploymentRepo,
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// executeArtifactoryManagePermissionGroup handles the permission group management tool execution
func executeArtifactoryManagePermissionGroup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters with localhost default
	baseURL := request.GetString("base_url", "http://localhost")
	username := request.GetString("username", "admin")
	password := request.GetString("password", "B@55w0rd")
	apiKey := request.GetString("api_key", "")

	// Extract group parameters
	groupName := request.GetString("group_name", "")
	if groupName == "" {
		return mcp.NewToolResultError("group_name is required"), nil
	}

	groupDescription := request.GetString("group_description", "")
	usersStr := request.GetString("users", "")
	repositoriesStr := request.GetString("repositories", "")
	privilegesStr := request.GetString("privileges", "READ")
	autoJoin := request.GetBool("auto_join", false)
	adminPrivileges := request.GetBool("admin_privileges", false)
	realm := request.GetString("realm", "internal")

	// Parse timeout (optional)
	timeout := artifactoryDefaultTimeout
	if timeoutSec := request.GetFloat("timeout", 0); timeoutSec > 0 {
		timeoutDuration := time.Duration(timeoutSec) * time.Second
		if timeoutDuration > artifactoryMaxTimeout {
			timeout = artifactoryMaxTimeout
		} else {
			timeout = timeoutDuration
		}
	}

	// Validate and construct the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid base URL: %v", err)), nil
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		baseURL = "http://" + baseURL
		parsedURL, err = url.Parse(baseURL)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid URL after adding http: %v", err)), nil
		}
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return mcp.NewToolResultError("URL must use http:// or https://"), nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Step 1: Create or update the group
	group := ArtifactoryGroup{
		Name:            groupName,
		Description:     groupDescription,
		AutoJoin:        autoJoin,
		AdminPrivileges: adminPrivileges,
		Realm:           realm,
		UsersNames:      []string{},
	}

	// Parse users if provided
	if usersStr != "" {
		users := strings.Split(usersStr, ",")
		for _, user := range users {
			user = strings.TrimSpace(user)
			if user != "" {
				group.UsersNames = append(group.UsersNames, user)
			}
		}
	}

	// Create group JSON
	groupJSON, err := json.Marshal(group)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal group data: %v", err)), nil
	}

	// Create group request
	groupURL := fmt.Sprintf("%s/artifactory/api/security/groups/%s", baseURL, groupName)
	groupReq, err := http.NewRequestWithContext(ctx, "PUT", groupURL, bytes.NewBuffer(groupJSON))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create group request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		groupReq.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		groupReq.SetBasicAuth(username, password)
	}

	// Set headers
	groupReq.Header.Set("User-Agent", "MCP-Artifactory-Group/1.0")
	groupReq.Header.Set("Accept", "application/json")
	groupReq.Header.Set("Content-Type", "application/json")

	// Execute group request
	groupResp, err := client.Do(groupReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create/update group: %v", err)), nil
	}
	defer groupResp.Body.Close()

	// Check group creation status
	if groupResp.StatusCode != http.StatusOK && groupResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(groupResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("failed to create/update group: status %d, response: %s", groupResp.StatusCode, string(bodyBytes))), nil
	}

	// Step 2: Create permission target (always create one)
	// Parse repositories and privileges
	var repositories []string
	if repositoriesStr != "" {
		repositories = strings.Split(repositoriesStr, ",")
		// Trim whitespace
		for i, repo := range repositories {
			repositories[i] = strings.TrimSpace(repo)
		}
	}

	privileges := strings.Split(privilegesStr, ",")
	// Trim whitespace
	for i, priv := range privileges {
		privileges[i] = strings.TrimSpace(priv)
	}

	// Create permission
	permission := ArtifactoryPermission{
		Name: fmt.Sprintf("%s-permission", groupName),
		Principals: struct {
			Users  map[string][]string `json:"users,omitempty"`
			Groups map[string][]string `json:"groups,omitempty"`
		}{
			Groups: map[string][]string{
				groupName: privileges,
			},
		},
	}

	// Set repositories
	if len(repositories) > 0 {
		permission.Repo.Repositories = repositories
	} else {
		// Use "ANY" for all repositories
		permission.Repo.Repositories = []string{"ANY"}
	}

	// Set actions for the group
	permission.Repo.Actions.Groups = map[string][]string{
		groupName: privileges,
	}

	// Create permission JSON
	permissionJSON, err := json.Marshal(permission)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal permission data: %v", err)), nil
	}

	// Create permission request
	permissionURL := fmt.Sprintf("%s/artifactory/api/security/permissions/%s", baseURL, permission.Name)
	permissionReq, err := http.NewRequestWithContext(ctx, "PUT", permissionURL, bytes.NewBuffer(permissionJSON))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create permission request: %v", err)), nil
	}

	// Set authentication headers
	if apiKey != "" {
		permissionReq.Header.Set("X-JFrog-Art-Api", apiKey)
	} else if username != "" && password != "" {
		permissionReq.SetBasicAuth(username, password)
	}

	// Set headers
	permissionReq.Header.Set("User-Agent", "MCP-Artifactory-Permission/1.0")
	permissionReq.Header.Set("Accept", "application/json")
	permissionReq.Header.Set("Content-Type", "application/json")

	// Execute permission request
	permissionResp, err := client.Do(permissionReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create permission: %v", err)), nil
	}
	defer permissionResp.Body.Close()

	// Check permission creation status
	if permissionResp.StatusCode != http.StatusOK && permissionResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(permissionResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("failed to create permission: status %d, response: %s", permissionResp.StatusCode, string(bodyBytes))), nil
	}

	// Return success result
	result := map[string]interface{}{
		"group": map[string]interface{}{
			"name":            groupName,
			"description":     groupDescription,
			"autoJoin":        autoJoin,
			"adminPrivileges": adminPrivileges,
			"realm":           realm,
			"users":           group.UsersNames,
		},
		"permission": map[string]interface{}{
			"created":    true,
			"name":       fmt.Sprintf("%s-permission", groupName),
			"repository": permission.Repo,
			"repositories": func() []string {
				if repositoriesStr == "" {
					return []string{}
				}
				repos := strings.Split(repositoriesStr, ",")
				for i, repo := range repos {
					repos[i] = strings.TrimSpace(repo)
				}
				return repos
			}(),
			"privileges": func() []string {
				privs := strings.Split(privilegesStr, ",")
				for i, priv := range privs {
					privs[i] = strings.TrimSpace(priv)
				}
				return privs
			}(),
		},
		"url":       fmt.Sprintf("%s/artifactory/api/security/groups/%s", baseURL, groupName),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

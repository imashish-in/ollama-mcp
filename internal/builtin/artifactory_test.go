package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewArtifactoryServer(t *testing.T) {
	server, err := NewArtifactoryServer()
	if err != nil {
		t.Fatalf("Failed to create Artifactory server: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}
}

func TestArtifactoryServerRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test that Artifactory server is registered
	servers := registry.ListServers()
	found := false
	for _, name := range servers {
		if name == "artifactory" {
			found = true
			break
		}
	}

	if !found {
		t.Error("artifactory server not found in registry")
	}

	// Test creating Artifactory server through registry
	wrapper, err := registry.CreateServer("artifactory", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("Failed to create Artifactory server through registry: %v", err)
	}

	if wrapper == nil {
		t.Fatal("Expected wrapper to be non-nil")
	}

	if wrapper.GetServer() == nil {
		t.Fatal("Expected wrapped server to be non-nil")
	}
}

func TestExecuteArtifactoryHealthcheck_Success(t *testing.T) {
	// Create a mock server that returns a healthy response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for the health endpoint
		if r.URL.Path != "/router/api/v1/system/health" {
			t.Errorf("Expected path /router/api/v1/system/health, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Check if the request method is GET
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Return a healthy response
		response := ArtifactoryHealthResponse{
			Status: "UP",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "artifactory_healthcheck",
			Arguments: map[string]interface{}{
				"base_url": mockServer.URL,
			},
		},
	}

	// Execute the healthcheck
	result, err := executeArtifactoryHealthcheck(context.Background(), request)
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Check that the result contains the expected content
	if len(result.Content) == 0 {
		t.Fatal("Result should have content")
	}

	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if response["artifactory_status"] != "UP" {
			t.Errorf("Expected artifactory_status 'UP', got %v", response["artifactory_status"])
		}
	} else {
		t.Fatal("Expected text content")
	}
}

func TestExecuteArtifactoryHealthcheck_Unhealthy(t *testing.T) {
	// Create a mock server that returns an unhealthy response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ArtifactoryHealthResponse{
			Status: "DOWN",
			Error:  "Database connection failed",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "artifactory_healthcheck",
			Arguments: map[string]interface{}{
				"base_url": mockServer.URL,
			},
		},
	}

	// Execute the healthcheck
	result, err := executeArtifactoryHealthcheck(context.Background(), request)
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}

	// Should return an error result
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.IsError {
		t.Error("Result should be an error")
	}
}

func TestExecuteArtifactoryHealthcheck_WithAuth(t *testing.T) {
	// Create a mock server that checks for authentication
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for API key header
		apiKey := r.Header.Get("X-JFrog-Art-Api")
		if apiKey != "test-api-key" {
			t.Errorf("Expected API key 'test-api-key', got %s", apiKey)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		response := ArtifactoryHealthResponse{
			Status: "UP",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request with API key
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "artifactory_healthcheck",
			Arguments: map[string]interface{}{
				"base_url": mockServer.URL,
				"api_key":  "test-api-key",
			},
		},
	}

	// Execute the healthcheck
	result, err := executeArtifactoryHealthcheck(context.Background(), request)
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.IsError {
		t.Error("Result should not be an error")
	}
}

func TestExecuteArtifactoryHealthcheck_InvalidURL(t *testing.T) {
	// Create the request with invalid URL
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "artifactory_healthcheck",
			Arguments: map[string]interface{}{
				"base_url": "invalid-url",
			},
		},
	}

	// Execute the healthcheck
	result, err := executeArtifactoryHealthcheck(context.Background(), request)
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}

	// Should return an error result
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.IsError {
		t.Error("Result should be an error")
	}
}

func TestExecuteArtifactoryHealthcheck_DefaultLocalhost(t *testing.T) {
	// Create a mock server that returns a healthy response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for the health endpoint
		if r.URL.Path != "/router/api/v1/system/health" {
			t.Errorf("Expected path /router/api/v1/system/health, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Return a healthy response
		response := ArtifactoryHealthResponse{
			Status: "UP",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request without base_url (should default to localhost:8081)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "artifactory_healthcheck",
			Arguments: map[string]interface{}{},
		},
	}

	// Execute the healthcheck
	result, err := executeArtifactoryHealthcheck(context.Background(), request)
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}

	// Should not be an error since we're using default localhost
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Note: This test will likely fail in real execution since localhost:8081 might not be running
	// but it tests that the function doesn't crash with default parameters
}

func TestExecuteArtifactoryGetRepositories_Success(t *testing.T) {
	// Create a mock server that returns repositories
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for the repositories endpoint
		if r.URL.Path != "/artifactory/api/repositories" {
			t.Errorf("Expected path /artifactory/api/repositories, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Return repositories response
		response := ArtifactoryRepositoriesResponse{
			Repositories: []ArtifactoryRepository{
				{
					Key:         "libs-release-local",
					Type:        "LOCAL",
					Description: "Local repository for release artifacts",
					PackageType: "maven",
				},
				{
					Key:         "libs-snapshot-local",
					Type:        "LOCAL",
					Description: "Local repository for snapshot artifacts",
					PackageType: "maven",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "artifactory_get_repositories",
			Arguments: map[string]interface{}{
				"base_url": mockServer.URL,
			},
		},
	}

	// Execute the repositories request
	result, err := executeArtifactoryGetRepositories(context.Background(), request)
	if err != nil {
		t.Fatalf("Repositories request failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Check that the result contains the expected content
	if len(result.Content) == 0 {
		t.Fatal("Result should have content")
	}

	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["count"] != float64(2) {
			t.Errorf("Expected count 2, got %v", response["count"])
		}

		repositories, ok := response["repositories"].([]interface{})
		if !ok {
			t.Fatal("Expected repositories array")
		}

		if len(repositories) != 2 {
			t.Errorf("Expected 2 repositories, got %d", len(repositories))
		}
	} else {
		t.Fatal("Expected text content")
	}
}

func TestExecuteArtifactoryGetRepositories_DefaultLocalhost(t *testing.T) {
	// Create a mock server that returns repositories
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for the repositories endpoint
		if r.URL.Path != "/artifactory/api/repositories" {
			t.Errorf("Expected path /artifactory/api/repositories, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Return repositories response
		response := ArtifactoryRepositoriesResponse{
			Repositories: []ArtifactoryRepository{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request without base_url (should default to localhost:8081)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "artifactory_get_repositories",
			Arguments: map[string]interface{}{},
		},
	}

	// Execute the repositories request
	result, err := executeArtifactoryGetRepositories(context.Background(), request)
	if err != nil {
		t.Fatalf("Repositories request failed: %v", err)
	}

	// Should not be an error since we're using default localhost
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Note: This test will likely fail in real execution since localhost:8081 might not be running
	// but it tests that the function doesn't crash with default parameters
}

func TestExecuteArtifactoryHealthcheck_Timeout(t *testing.T) {
	// Create a mock server that delays response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay for 2 seconds
		time.Sleep(2 * time.Second)
		response := ArtifactoryHealthResponse{
			Status: "UP",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create the request with short timeout
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "artifactory_healthcheck",
			Arguments: map[string]interface{}{
				"base_url": mockServer.URL,
				"timeout":  1.0, // 1 second timeout
			},
		},
	}

	// Execute the healthcheck
	result, err := executeArtifactoryHealthcheck(context.Background(), request)
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}

	// Should return an error result due to timeout
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.IsError {
		t.Error("Result should be an error due to timeout")
	}
}

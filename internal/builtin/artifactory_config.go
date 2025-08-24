package builtin

import (
	"fmt"
	"strings"
)

// ArtifactoryInstanceConfig represents configuration for a single Artifactory instance
type ArtifactoryInstanceConfig struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	APIKey      string `json:"apiKey,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	VerifySSL   bool   `json:"verifySSL,omitempty"`
	Description string `json:"description,omitempty"`
}

// ArtifactoryCommonSettings represents common settings for all Artifactory instances
type ArtifactoryCommonSettings struct {
	MaxRetries  int    `json:"maxRetries,omitempty"`
	RetryDelay  int    `json:"retryDelay,omitempty"`
	UserAgent   string `json:"userAgent,omitempty"`
	LogLevel    string `json:"logLevel,omitempty"`
}

// ArtifactoryConfig represents the complete Artifactory configuration
type ArtifactoryConfig struct {
	Instances       map[string]ArtifactoryInstanceConfig `json:"instances"`
	DefaultInstance string                              `json:"defaultInstance"`
	CommonSettings  ArtifactoryCommonSettings           `json:"commonSettings"`
}

// GetInstanceConfig retrieves configuration for a specific instance
func (c *ArtifactoryConfig) GetInstanceConfig(instanceName string) (*ArtifactoryInstanceConfig, error) {
	if instanceName == "" {
		instanceName = c.DefaultInstance
	}
	
	if instanceName == "" {
		// Fall back to first available instance
		for _, config := range c.Instances {
			return &config, nil
		}
		return nil, fmt.Errorf("no Artifactory instances configured")
	}
	
	config, exists := c.Instances[instanceName]
	if !exists {
		return nil, fmt.Errorf("Artifactory instance '%s' not found in configuration", instanceName)
	}
	
	return &config, nil
}

// LoadArtifactoryConfig loads Artifactory configuration from options
func LoadArtifactoryConfig(options map[string]any) (*ArtifactoryConfig, error) {
	if options == nil {
		return nil, fmt.Errorf("no configuration options provided")
	}
	
	// Check if config is directly in options
	if configData, ok := options["config"]; ok {
		if configMap, ok := configData.(map[string]any); ok {
			return parseArtifactoryConfig(configMap)
		}
	}
	
	// Fallback: create minimal config from individual options
	config := &ArtifactoryConfig{
		Instances: make(map[string]ArtifactoryInstanceConfig),
		CommonSettings: ArtifactoryCommonSettings{
			MaxRetries: 3,
			RetryDelay: 5,
			UserAgent:  "MCPHost-Artifactory-Client/1.0",
			LogLevel:   "info",
		},
	}
	
	// Extract instance configuration from options
	instance := ArtifactoryInstanceConfig{
		Name:      "default",
		URL:       getStringOption(options, "url", "http://localhost"),
		Username:  getStringOption(options, "username", "admin"),
		Password:  getStringOption(options, "password", "B@55w0rd"),
		APIKey:    getStringOption(options, "apiKey", ""),
		Timeout:   getIntOption(options, "timeout", 30),
		VerifySSL: getBoolOption(options, "verifySSL", true),
	}
	
	config.Instances["default"] = instance
	config.DefaultInstance = "default"
	
	return config, nil
}

// parseArtifactoryConfig parses the configuration from a map
func parseArtifactoryConfig(configMap map[string]any) (*ArtifactoryConfig, error) {
	config := &ArtifactoryConfig{
		Instances: make(map[string]ArtifactoryInstanceConfig),
		CommonSettings: ArtifactoryCommonSettings{
			MaxRetries: 3,
			RetryDelay: 5,
			UserAgent:  "MCPHost-Artifactory-Client/1.0",
			LogLevel:   "info",
		},
	}
	
	// Parse instances
	if instancesData, ok := configMap["instances"]; ok {
		if instancesMap, ok := instancesData.(map[string]any); ok {
			for instanceName, instanceData := range instancesMap {
				if instanceMap, ok := instanceData.(map[string]any); ok {
					instance := ArtifactoryInstanceConfig{
						Name:        getStringOption(instanceMap, "name", instanceName),
						URL:         getStringOption(instanceMap, "url", ""),
						Username:    getStringOption(instanceMap, "username", ""),
						Password:    getStringOption(instanceMap, "password", ""),
						APIKey:      getStringOption(instanceMap, "apiKey", ""),
						Timeout:     getIntOption(instanceMap, "timeout", 30),
						VerifySSL:   getBoolOption(instanceMap, "verifySSL", true),
						Description: getStringOption(instanceMap, "description", ""),
					}
					config.Instances[instanceName] = instance
				}
			}
		}
	}
	
	// Parse default instance
	config.DefaultInstance = getStringOption(configMap, "defaultInstance", "default")
	
	// Parse common settings
	if commonData, ok := configMap["commonSettings"]; ok {
		if commonMap, ok := commonData.(map[string]any); ok {
			config.CommonSettings.MaxRetries = getIntOption(commonMap, "maxRetries", 3)
			config.CommonSettings.RetryDelay = getIntOption(commonMap, "retryDelay", 5)
			config.CommonSettings.UserAgent = getStringOption(commonMap, "userAgent", "MCPHost-Artifactory-Client/1.0")
			config.CommonSettings.LogLevel = getStringOption(commonMap, "logLevel", "info")
		}
	}
	
	// Validate configuration
	if len(config.Instances) == 0 {
		return nil, fmt.Errorf("no Artifactory instances configured")
	}
	
	return config, nil
}

// Helper functions to safely extract values from maps
func getStringOption(m map[string]any, key, defaultValue string) string {
	if value, ok := m[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getIntOption(m map[string]any, key string, defaultValue int) int {
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

func getBoolOption(m map[string]any, key string, defaultValue bool) bool {
	if value, ok := m[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetCredentials returns the appropriate authentication credentials for an instance
func (ic *ArtifactoryInstanceConfig) GetCredentials() (string, string, string) {
	// Prefer API key over username/password
	if ic.APIKey != "" {
		return "", "", ic.APIKey
	}
	return ic.Username, ic.Password, ""
}

// Validate validates the instance configuration
func (ic *ArtifactoryInstanceConfig) Validate() error {
	if ic.URL == "" {
		return fmt.Errorf("Artifactory URL is required")
	}
	
	if ic.APIKey == "" && (ic.Username == "" || ic.Password == "") {
		return fmt.Errorf("either API key or username/password is required")
	}
	
	if ic.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	
	return nil
}

// GetFullURL returns the full URL for an API endpoint
func (ic *ArtifactoryInstanceConfig) GetFullURL(endpoint string) string {
	baseURL := strings.TrimSuffix(ic.URL, "/")
	endpoint = strings.TrimPrefix(endpoint, "/")
	return fmt.Sprintf("%s/%s", baseURL, endpoint)
}

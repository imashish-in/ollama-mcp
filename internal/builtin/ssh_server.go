package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/crypto/ssh"
)

// SSHServerConfig represents SSH server configuration
type SSHServerConfig struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"`
	KeyPath     string `json:"key_path,omitempty"`
	Timeout     int    `json:"timeout"`
	Description string `json:"description"`
}

// SSHConfig represents the overall SSH configuration
type SSHConfig struct {
	Instances       map[string]SSHServerConfig `json:"instances"`
	DefaultInstance string                     `json:"defaultInstance"`
	CommonSettings  struct {
		MaxRetries int    `json:"maxRetries"`
		RetryDelay int    `json:"retryDelay"`
		LogLevel   string `json:"logLevel"`
		UserAgent  string `json:"userAgent"`
	} `json:"commonSettings"`
}

// SystemResourceInfo represents system resource information
type SystemResourceInfo struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUUsage    float64   `json:"cpu_usage_percent"`
	MemoryTotal int64     `json:"memory_total_mb"`
	MemoryUsed  int64     `json:"memory_used_mb"`
	MemoryFree  int64     `json:"memory_free_mb"`
	DiskTotal   int64     `json:"disk_total_gb"`
	DiskUsed    int64     `json:"disk_used_gb"`
	DiskFree    int64     `json:"disk_free_gb"`
	LoadAverage []float64 `json:"load_average"`
	Uptime      string    `json:"uptime"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Command   string    `json:"command"`
	Output    string    `json:"output"`
	Error     string    `json:"error,omitempty"`
	ExitCode  int       `json:"exit_code"`
	Duration  string    `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}

// SSHOperationResult represents the result of SSH operations
type SSHOperationResult struct {
	ServerName    string              `json:"server_name"`
	Operation     string              `json:"operation"`
	Success       bool                `json:"success"`
	Message       string              `json:"message"`
	Duration      string              `json:"duration"`
	Timestamp     time.Time           `json:"timestamp"`
	SystemInfo    *SystemResourceInfo `json:"system_info,omitempty"`
	CommandResult *CommandResult      `json:"command_result,omitempty"`
	Errors        []string            `json:"errors,omitempty"`
}

// Global SSH configuration
var globalSSHConfig *SSHConfig

// NewSSHServer creates a new SSH server MCP server
func NewSSHServer(options map[string]any) (*server.MCPServer, error) {
	s := server.NewMCPServer("ssh-server", "1.0.0", server.WithToolCapabilities(true))

	// Load SSH configuration
	config, err := LoadSSHConfig(options)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH config: %v", err)
	}
	globalSSHConfig = config

	// Register SSH tools
	sshConnectTool := mcp.NewTool("ssh_connect",
		mcp.WithDescription("Connect to a remote SSH server and verify connectivity"),
		mcp.WithString("server_name",
			mcp.Description("Name of the server from configuration (e.g., 'production', 'staging')"),
		),
		mcp.WithString("host",
			mcp.Description("Override host IP/domain (optional)"),
		),
		mcp.WithString("username",
			mcp.Description("Override username (optional)"),
		),
		mcp.WithString("password",
			mcp.Description("Override password (optional)"),
		),
	)

	sshSystemInfoTool := mcp.NewTool("ssh_system_info",
		mcp.WithDescription("Get system resource information (CPU, memory, disk) from remote SSH server"),
		mcp.WithString("server_name",
			mcp.Description("Name of the server from configuration"),
		),
		mcp.WithString("host",
			mcp.Description("Override host IP/domain (optional)"),
		),
		mcp.WithString("username",
			mcp.Description("Override username (optional)"),
		),
		mcp.WithString("password",
			mcp.Description("Override password (optional)"),
		),
	)

	sshExecuteCommandTool := mcp.NewTool("ssh_execute_command",
		mcp.WithDescription("Execute a command on remote SSH server (rm command is blocked for safety)"),
		mcp.WithString("server_name",
			mcp.Description("Name of the server from configuration"),
		),
		mcp.WithString("command",
			mcp.Description("Command to execute (rm commands are blocked for safety)"),
		),
		mcp.WithString("host",
			mcp.Description("Override host IP/domain (optional)"),
		),
		mcp.WithString("username",
			mcp.Description("Override username (optional)"),
		),
		mcp.WithString("password",
			mcp.Description("Override password (optional)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Command timeout in seconds (default: 30)"),
		),
	)

	sshExecuteMultipleCommandsTool := mcp.NewTool("ssh_execute_multiple_commands",
		mcp.WithDescription("Execute multiple commands on remote SSH server (rm commands are blocked for safety)"),
		mcp.WithString("server_name",
			mcp.Description("Name of the server from configuration"),
		),
		mcp.WithString("commands",
			mcp.Description("Semicolon-separated list of commands to execute"),
		),
		mcp.WithString("host",
			mcp.Description("Override host IP/domain (optional)"),
		),
		mcp.WithString("username",
			mcp.Description("Override username (optional)"),
		),
		mcp.WithString("password",
			mcp.Description("Override password (optional)"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Command timeout in seconds (default: 30)"),
		),
	)

	s.AddTool(sshConnectTool, executeSSHConnect)
	s.AddTool(sshSystemInfoTool, executeSSHSystemInfo)
	s.AddTool(sshExecuteCommandTool, executeSSHExecuteCommand)
	s.AddTool(sshExecuteMultipleCommandsTool, executeSSHExecuteMultipleCommands)

	return s, nil
}

// LoadSSHConfig loads SSH configuration from options
func LoadSSHConfig(options map[string]any) (*SSHConfig, error) {
	if options == nil {
		return nil, fmt.Errorf("no SSH configuration provided")
	}

	configData, ok := options["config"]
	if !ok {
		return nil, fmt.Errorf("SSH configuration not found in options")
	}

	configBytes, err := json.Marshal(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SSH config: %v", err)
	}

	var config SSHConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %v", err)
	}

	return &config, nil
}

// getSSHInstanceConfig retrieves instance-specific configuration
func getSSHInstanceConfig(instanceName string) (*SSHServerConfig, error) {
	if globalSSHConfig == nil {
		return nil, fmt.Errorf("SSH configuration not loaded")
	}

	config, exists := globalSSHConfig.Instances[instanceName]
	if !exists {
		return nil, fmt.Errorf("SSH instance '%s' not found in configuration", instanceName)
	}

	return &config, nil
}

// createSSHClient creates an SSH client connection
func createSSHClient(config *SSHServerConfig) (*ssh.Client, error) {
	var authMethod ssh.AuthMethod

	// Try private key first
	if config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(config.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else if config.KeyPath != "" {
		keyBytes, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else if config.Password != "" {
		authMethod = ssh.Password(config.Password)
	} else {
		return nil, fmt.Errorf("no authentication method provided (password or private key required)")
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(config.Timeout) * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	return client, nil
}

// executeSSHConnect handles SSH connection verification
func executeSSHConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	serverName := request.GetString("server_name", "")
	host := request.GetString("host", "")
	username := request.GetString("username", "")
	password := request.GetString("password", "")

	if serverName == "" {
		return mcp.NewToolResultError("server_name is required"), nil
	}

	// Get instance configuration
	instanceConfig, err := getSSHInstanceConfig(serverName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get SSH config: %v", err)), nil
	}

	// Override with provided parameters
	if host != "" {
		instanceConfig.Host = host
	}
	if username != "" {
		instanceConfig.Username = username
	}
	if password != "" {
		instanceConfig.Password = password
	}

	// Test connection
	client, err := createSSHClient(instanceConfig)
	if err != nil {
		result := &SSHOperationResult{
			ServerName: serverName,
			Operation:  "connect",
			Success:    false,
			Message:    fmt.Sprintf("Failed to connect: %v", err),
			Duration:   time.Since(startTime).String(),
			Timestamp:  time.Now(),
			Errors:     []string{err.Error()},
		}
		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}
	defer client.Close()

	result := &SSHOperationResult{
		ServerName: serverName,
		Operation:  "connect",
		Success:    true,
		Message:    fmt.Sprintf("Successfully connected to %s:%d", instanceConfig.Host, instanceConfig.Port),
		Duration:   time.Since(startTime).String(),
		Timestamp:  time.Now(),
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// executeSSHSystemInfo handles system resource monitoring
func executeSSHSystemInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	serverName := request.GetString("server_name", "")
	host := request.GetString("host", "")
	username := request.GetString("username", "")
	password := request.GetString("password", "")

	if serverName == "" {
		return mcp.NewToolResultError("server_name is required"), nil
	}

	// Get instance configuration
	instanceConfig, err := getSSHInstanceConfig(serverName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get SSH config: %v", err)), nil
	}

	// Override with provided parameters
	if host != "" {
		instanceConfig.Host = host
	}
	if username != "" {
		instanceConfig.Username = username
	}
	if password != "" {
		instanceConfig.Password = password
	}

	// Create SSH client
	client, err := createSSHClient(instanceConfig)
	if err != nil {
		result := &SSHOperationResult{
			ServerName: serverName,
			Operation:  "system_info",
			Success:    false,
			Message:    fmt.Sprintf("Failed to connect: %v", err),
			Duration:   time.Since(startTime).String(),
			Timestamp:  time.Now(),
			Errors:     []string{err.Error()},
		}
		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}
	defer client.Close()

	// Get system information
	systemInfo, err := getSystemInfo(client)
	if err != nil {
		result := &SSHOperationResult{
			ServerName: serverName,
			Operation:  "system_info",
			Success:    false,
			Message:    fmt.Sprintf("Failed to get system info: %v", err),
			Duration:   time.Since(startTime).String(),
			Timestamp:  time.Now(),
			Errors:     []string{err.Error()},
		}
		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}

	result := &SSHOperationResult{
		ServerName: serverName,
		Operation:  "system_info",
		Success:    true,
		Message:    "Successfully retrieved system information",
		Duration:   time.Since(startTime).String(),
		Timestamp:  time.Now(),
		SystemInfo: systemInfo,
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// executeSSHExecuteCommand handles single command execution
func executeSSHExecuteCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	serverName := request.GetString("server_name", "")
	command := request.GetString("command", "")
	host := request.GetString("host", "")
	username := request.GetString("username", "")
	password := request.GetString("password", "")
	timeout := int(request.GetFloat("timeout", 30))

	if serverName == "" {
		return mcp.NewToolResultError("server_name is required"), nil
	}
	if command == "" {
		return mcp.NewToolResultError("command is required"), nil
	}

	// Check for dangerous commands
	if isDangerousCommand(command) {
		return mcp.NewToolResultError("Command contains dangerous operations (rm, format, etc.) and is blocked for safety"), nil
	}

	// Get instance configuration
	instanceConfig, err := getSSHInstanceConfig(serverName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get SSH config: %v", err)), nil
	}

	// Override with provided parameters
	if host != "" {
		instanceConfig.Host = host
	}
	if username != "" {
		instanceConfig.Username = username
	}
	if password != "" {
		instanceConfig.Password = password
	}

	// Create SSH client
	client, err := createSSHClient(instanceConfig)
	if err != nil {
		result := &SSHOperationResult{
			ServerName: serverName,
			Operation:  "execute_command",
			Success:    false,
			Message:    fmt.Sprintf("Failed to connect: %v", err),
			Duration:   time.Since(startTime).String(),
			Timestamp:  time.Now(),
			Errors:     []string{err.Error()},
		}
		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}
	defer client.Close()

	// Execute command
	cmdResult, err := executeCommand(client, command, timeout)
	if err != nil {
		result := &SSHOperationResult{
			ServerName: serverName,
			Operation:  "execute_command",
			Success:    false,
			Message:    fmt.Sprintf("Failed to execute command: %v", err),
			Duration:   time.Since(startTime).String(),
			Timestamp:  time.Now(),
			Errors:     []string{err.Error()},
		}
		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}

	result := &SSHOperationResult{
		ServerName:    serverName,
		Operation:     "execute_command",
		Success:       true,
		Message:       "Command executed successfully",
		Duration:      time.Since(startTime).String(),
		Timestamp:     time.Now(),
		CommandResult: cmdResult,
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// executeSSHExecuteMultipleCommands handles multiple command execution
func executeSSHExecuteMultipleCommands(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	serverName := request.GetString("server_name", "")
	commands := request.GetString("commands", "")
	host := request.GetString("host", "")
	username := request.GetString("username", "")
	password := request.GetString("password", "")
	timeout := int(request.GetFloat("timeout", 30))

	if serverName == "" {
		return mcp.NewToolResultError("server_name is required"), nil
	}
	if commands == "" {
		return mcp.NewToolResultError("commands is required"), nil
	}

	// Split commands
	commandList := strings.Split(commands, ";")
	var results []*CommandResult
	var errors []string

	// Get instance configuration
	instanceConfig, err := getSSHInstanceConfig(serverName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get SSH config: %v", err)), nil
	}

	// Override with provided parameters
	if host != "" {
		instanceConfig.Host = host
	}
	if username != "" {
		instanceConfig.Username = username
	}
	if password != "" {
		instanceConfig.Password = password
	}

	// Create SSH client
	client, err := createSSHClient(instanceConfig)
	if err != nil {
		result := &SSHOperationResult{
			ServerName: serverName,
			Operation:  "execute_multiple_commands",
			Success:    false,
			Message:    fmt.Sprintf("Failed to connect: %v", err),
			Duration:   time.Since(startTime).String(),
			Timestamp:  time.Now(),
			Errors:     []string{err.Error()},
		}
		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}
	defer client.Close()

	// Execute each command
	for i, cmd := range commandList {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		// Check for dangerous commands
		if isDangerousCommand(cmd) {
			errors = append(errors, fmt.Sprintf("Command %d contains dangerous operations and is blocked: %s", i+1, cmd))
			continue
		}

		cmdResult, err := executeCommand(client, cmd, timeout)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Command %d failed: %v", i+1, err))
		} else {
			results = append(results, cmdResult)
		}
	}

	success := len(errors) == 0
	message := fmt.Sprintf("Executed %d commands successfully", len(results))
	if len(errors) > 0 {
		message = fmt.Sprintf("Executed %d commands, %d failed", len(results), len(errors))
	}

	result := &SSHOperationResult{
		ServerName: serverName,
		Operation:  "execute_multiple_commands",
		Success:    success,
		Message:    message,
		Duration:   time.Since(startTime).String(),
		Timestamp:  time.Now(),
		Errors:     errors,
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// getSystemInfo retrieves system resource information
func getSystemInfo(client *ssh.Client) (*SystemResourceInfo, error) {
	// Get CPU and memory info
	cpuMemCmd := "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1"
	cpuOutput, err := executeCommandString(client, cpuMemCmd, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %v", err)
	}

	// Get memory info
	memCmd := "free -m | grep '^Mem:' | awk '{print $2, $3, $4}'"
	memOutput, err := executeCommandString(client, memCmd, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %v", err)
	}

	// Get disk info
	diskCmd := "df -h / | tail -1 | awk '{print $2, $3, $4}'"
	diskOutput, err := executeCommandString(client, diskCmd, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk info: %v", err)
	}

	// Get load average
	loadCmd := "uptime | awk -F'load average:' '{print $2}' | awk '{print $1, $2, $3}'"
	loadOutput, err := executeCommandString(client, loadCmd, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get load average: %v", err)
	}

	// Get uptime
	uptimeCmd := "uptime -p"
	uptimeOutput, err := executeCommandString(client, uptimeCmd, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime: %v", err)
	}

	// Parse CPU usage
	cpuUsage, _ := strconv.ParseFloat(strings.TrimSpace(cpuOutput), 64)

	// Parse memory info
	memParts := strings.Fields(strings.TrimSpace(memOutput))
	memoryTotal, _ := strconv.ParseInt(memParts[0], 10, 64)
	memoryUsed, _ := strconv.ParseInt(memParts[1], 10, 64)
	memoryFree, _ := strconv.ParseInt(memParts[2], 10, 64)

	// Parse disk info (convert GB to bytes)
	diskParts := strings.Fields(strings.TrimSpace(diskOutput))
	diskTotalStr := strings.TrimSuffix(diskParts[0], "G")
	diskUsedStr := strings.TrimSuffix(diskParts[1], "G")
	diskFreeStr := strings.TrimSuffix(diskParts[2], "G")

	diskTotal, _ := strconv.ParseInt(diskTotalStr, 10, 64)
	diskUsed, _ := strconv.ParseInt(diskUsedStr, 10, 64)
	diskFree, _ := strconv.ParseInt(diskFreeStr, 10, 64)

	// Parse load average
	loadParts := strings.Fields(strings.TrimSpace(loadOutput))
	var loadAverage []float64
	for _, load := range loadParts {
		if loadFloat, err := strconv.ParseFloat(strings.TrimSuffix(load, ","), 64); err == nil {
			loadAverage = append(loadAverage, loadFloat)
		}
	}

	return &SystemResourceInfo{
		Timestamp:   time.Now(),
		CPUUsage:    cpuUsage,
		MemoryTotal: memoryTotal,
		MemoryUsed:  memoryUsed,
		MemoryFree:  memoryFree,
		DiskTotal:   diskTotal,
		DiskUsed:    diskUsed,
		DiskFree:    diskFree,
		LoadAverage: loadAverage,
		Uptime:      strings.TrimSpace(uptimeOutput),
	}, nil
}

// executeCommand executes a command on the SSH client
func executeCommand(client *ssh.Client, command string, timeout int) (*CommandResult, error) {
	startTime := time.Now()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Execute command
	output, err := session.CombinedOutput(command)
	duration := time.Since(startTime)

	result := &CommandResult{
		Command:   command,
		Output:    string(output),
		Duration:  duration.String(),
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitError.ExitStatus()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// executeCommandString executes a command and returns the output as string
func executeCommandString(client *ssh.Client, command string, timeout int) (string, error) {
	result, err := executeCommand(client, command, timeout)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Error)
	}
	return result.Output, nil
}

// isDangerousCommand checks if a command contains dangerous operations
func isDangerousCommand(command string) bool {
	dangerousPatterns := []string{
		"rm ", "rm -rf", "rm -r", "rm -f",
		"format", "mkfs", "dd if=",
		"shutdown", "halt", "reboot",
		"> /dev/", "> /proc/", "> /sys/",
		"chmod 000", "chmod 777",
		"sudo rm", "sudo format", "sudo mkfs",
	}

	commandLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(commandLower, pattern) {
			return true
		}
	}

	return false
}

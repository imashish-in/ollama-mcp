# SSH Server Management Tool Guide

The SSH Server Management tool provides comprehensive remote server management capabilities including system monitoring, command execution, and secure connectivity.

## 🚀 Features

- **Secure SSH Connections**: Connect to remote servers using password or private key authentication
- **System Resource Monitoring**: Monitor CPU, memory, disk usage, and system load
- **Safe Command Execution**: Execute commands with built-in safety checks
- **Multiple Server Support**: Manage multiple servers from a single configuration
- **Dangerous Command Protection**: Automatically blocks dangerous operations (rm, format, etc.)

## 📋 Available Tools

### 1. **ssh_connect** - Test SSH Connectivity
Verifies connection to a remote SSH server.

### 2. **ssh_system_info** - System Resource Monitoring
Retrieves comprehensive system resource information:
- CPU usage percentage
- Memory usage (total, used, free)
- Disk usage (total, used, free)
- System load average
- Server uptime

### 3. **ssh_execute_command** - Single Command Execution
Executes a single command on the remote server with safety checks.

### 4. **ssh_execute_multiple_commands** - Multiple Command Execution
Executes multiple commands (semicolon-separated) on the remote server.

## 🔧 Configuration

### Server Configuration in `local.json`

```json
{
  "mcpServers": {
    "ssh-server": {
      "type": "builtin",
      "name": "ssh-server",
      "options": {
        "config": {
          "instances": {
            "default": {
              "name": "Default SSH Server",
              "host": "192.168.1.100",
              "port": 22,
              "username": "admin",
              "password": "password123",
              "timeout": 30,
              "description": "Default SSH server instance"
            },
            "production": {
              "name": "Production Server",
              "host": "prod.example.com",
              "port": 22,
              "username": "deploy",
              "key_path": "/path/to/private_key.pem",
              "timeout": 30,
              "description": "Production server with key authentication"
            }
          },
          "defaultInstance": "default",
          "commonSettings": {
            "maxRetries": 3,
            "retryDelay": 5,
            "logLevel": "info",
            "userAgent": "MCPHost-SSH-Client/1.0"
          }
        }
      }
    }
  }
}
```

### Authentication Methods

#### Password Authentication
```json
{
  "host": "192.168.1.100",
  "port": 22,
  "username": "admin",
  "password": "your_password",
  "timeout": 30
}
```

#### Private Key Authentication
```json
{
  "host": "prod.example.com",
  "port": 22,
  "username": "deploy",
  "key_path": "/path/to/private_key.pem",
  "timeout": 30
}
```

#### Inline Private Key
```json
{
  "host": "prod.example.com",
  "port": 22,
  "username": "deploy",
  "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----",
  "timeout": 30
}
```

## 📖 Usage Examples

### 1. Test SSH Connection

```bash
# Test connection to default server
./mcphost --config=local.json -m ollama:qwen3:8b -p "Test SSH connection to the default server"

# Test connection to specific server
./mcphost --config=local.json -m ollama:qwen3:8b -p "Test SSH connection to the production server"
```

### 2. Monitor System Resources

```bash
# Get system information from default server
./mcphost --config=local.json -m ollama:qwen3:8b -p "Get system resource information from the default server"

# Monitor production server
./mcphost --config=local.json -m ollama:qwen3:8b -p "Monitor CPU, memory, and disk usage on production server"
```

### 3. Execute Commands

```bash
# Execute single command
./mcphost --config=local.json -m ollama:qwen3:8b -p "Execute 'df -h' command on the default server"

# Execute multiple commands
./mcphost --config=local.json -m ollama:qwen3:8b -p "Execute commands: 'whoami; pwd; ls -la' on production server"

# Check running processes
./mcphost --config=local.json -m ollama:qwen3:8b -p "Check running processes on the server using 'ps aux'"
```

### 4. System Administration

```bash
# Check disk space
./mcphost --config=local.json -m ollama:qwen3:8b -p "Check disk space usage on all mounted filesystems"

# Monitor system load
./mcphost --config=local.json -m ollama:qwen3:8b -p "Check system load average and uptime"

# View system logs
./mcphost --config=local.json -m ollama:qwen3:8b -p "View recent system logs using 'tail -n 50 /var/log/syslog'"
```

## 🛡️ Safety Features

### Blocked Commands
The tool automatically blocks dangerous commands for safety:
- `rm`, `rm -rf`, `rm -r`, `rm -f`
- `format`, `mkfs`, `dd if=`
- `shutdown`, `halt`, `reboot`
- `> /dev/`, `> /proc/`, `> /sys/`
- `chmod 000`, `chmod 777`
- `sudo rm`, `sudo format`, `sudo mkfs`

### Timeout Protection
- Default command timeout: 30 seconds
- Configurable timeout per command
- Automatic session cleanup

### Error Handling
- Comprehensive error reporting
- Connection failure handling
- Command execution error details
- Exit code tracking

## 📊 Response Format

### System Information Response
```json
{
  "server_name": "default",
  "operation": "system_info",
  "success": true,
  "message": "Successfully retrieved system information",
  "duration": "1.234s",
  "timestamp": "2024-01-15T10:30:00Z",
  "system_info": {
    "timestamp": "2024-01-15T10:30:00Z",
    "cpu_usage_percent": 15.2,
    "memory_total_mb": 8192,
    "memory_used_mb": 4096,
    "memory_free_mb": 4096,
    "disk_total_gb": 100,
    "disk_used_gb": 45,
    "disk_free_gb": 55,
    "load_average": [1.2, 1.1, 0.9],
    "uptime": "up 5 days, 3 hours, 45 minutes"
  }
}
```

### Command Execution Response
```json
{
  "server_name": "default",
  "operation": "execute_command",
  "success": true,
  "message": "Command executed successfully",
  "duration": "0.5s",
  "timestamp": "2024-01-15T10:30:00Z",
  "command_result": {
    "command": "df -h",
    "output": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       100G   45G   55G  45% /\n",
    "exit_code": 0,
    "duration": "0.1s",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## 🔍 Common Use Cases

### 1. Server Health Monitoring
```bash
# Comprehensive health check
./mcphost --config=local.json -m ollama:qwen3:8b -p "Perform a comprehensive health check on the server including CPU, memory, disk, and running processes"
```

### 2. Log Analysis
```bash
# Check application logs
./mcphost --config=local.json -m ollama:qwen3:8b -p "Check recent application logs for errors"
```

### 3. System Maintenance
```bash
# Check for updates
./mcphost --config=local.json -m ollama:qwen3:8b -p "Check for available system updates"
```

### 4. Performance Monitoring
```bash
# Monitor performance metrics
./mcphost --config=local.json -m ollama:qwen3:8b -p "Monitor system performance including load average and resource usage"
```

## ⚠️ Important Notes

### Security Considerations
- Store sensitive credentials securely
- Use private key authentication when possible
- Regularly rotate passwords and keys
- Limit server access to necessary users only

### Performance Considerations
- Set appropriate timeouts for long-running commands
- Monitor connection pool usage
- Use connection pooling for multiple operations

### Troubleshooting
- Verify network connectivity
- Check SSH service status on target server
- Validate authentication credentials
- Review firewall settings

## 🔧 Advanced Configuration

### Custom Timeouts
```json
{
  "timeout": 60,
  "command_timeout": 120
}
```

### Retry Configuration
```json
{
  "maxRetries": 5,
  "retryDelay": 10
}
```

### Logging Configuration
```json
{
  "logLevel": "debug",
  "userAgent": "Custom-SSH-Client/1.0"
}
```

## 📚 Related Tools

- **Artifactory Tools**: Manage Artifactory repositories and permissions
- **Support Bundle Tools**: Analyze support bundles and logs
- **Archive Extractor**: Extract and analyze compressed files
- **Log Analyzer**: Analyze log files for errors and patterns

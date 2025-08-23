# Artifactory Tools Guide - From Scratch to Production

This comprehensive guide will walk you through setting up and using the Artifactory tools from scratch, including repository cloning, configuration, and practical usage examples.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Getting Started](#getting-started)
3. [Installation & Setup](#installation--setup)
4. [Configuration](#configuration)
5. [Artifactory Tools Overview](#artifactory-tools-overview)
6. [Usage Examples](#usage-examples)
7. [Troubleshooting](#troubleshooting)
8. [Advanced Usage](#advanced-usage)

## Prerequisites

Before you begin, ensure you have the following installed:

### Required Software
- **Go** (version 1.21 or higher)
- **Git**
- **Ollama** (for local model inference)
- **Artifactory** instance (local or remote)

### Optional Software
- **Docker** (for containerized Artifactory)
- **Make** (for build automation)

### System Requirements
- **Operating System**: macOS, Linux, or Windows
- **Memory**: Minimum 4GB RAM (8GB recommended)
- **Storage**: At least 2GB free space

## Getting Started

### 1. Clone the Repository

```bash
# Clone the MCPHost repository
git clone https://github.com/your-username/mcphost.git
cd mcphost

# Verify the repository structure
ls -la
```

### 2. Verify Go Installation

```bash
# Check Go version
go version

# Should output something like: go version go1.21.0 darwin/amd64
```

### 3. Install Ollama (if not already installed)

#### macOS
```bash
# Using Homebrew
brew install ollama

# Or download from https://ollama.ai
```

#### Linux
```bash
# Download and install
curl -fsSL https://ollama.ai/install.sh | sh
```

#### Windows
```bash
# Download from https://ollama.ai and run the installer
```

## Installation & Setup

### 1. Build the Application

```bash
# Navigate to the project directory
cd mcphost

# Build the application
go build -o mcphost .

# Verify the build
./mcphost --help
```

### 2. Pull Required Models

```bash
# Pull the recommended model
ollama pull qwen2.5:7b

# Or pull other models as needed
ollama pull llama3.1:8b
ollama pull mistral:7b
```

### 3. Set Up Artifactory

#### Option A: Local Artifactory (Docker)

```bash
# Pull and run Artifactory Community Edition
docker run -d \
  --name artifactory \
  -p 8081:8081 \
  -p 8082:8082 \
  releases-docker.jfrog.io/jfrog/artifactory-oss:latest

# Wait for Artifactory to start (usually 1-2 minutes)
docker logs artifactory

# Access Artifactory at http://localhost:8081
# Default credentials: admin / password
```

#### Option B: Remote Artifactory

If you have an existing Artifactory instance, note down:
- **URL**: `https://your-artifactory.company.com`
- **Username**: Your Artifactory username
- **Password/API Key**: Your authentication credentials

## Configuration

### 1. Create Configuration File

Create a `local.json` file in the project root:

```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "base_url": "http://localhost:8081",
        "username": "admin",
        "password": "password",
        "timeout": 30
      }
    }
  }
}
```

### 2. Environment-Specific Configurations

#### Development Environment
```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "base_url": "http://localhost:8081",
        "username": "admin",
        "password": "password"
      }
    }
  }
}
```

#### Production Environment
```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "base_url": "https://artifactory.company.com",
        "username": "${env://ARTIFACTORY_USER}",
        "password": "${env://ARTIFACTORY_PASS}",
        "timeout": 60
      }
    }
  }
}
```

### 3. Environment Variables (Optional)

For production environments, set environment variables:

```bash
# Set environment variables
export ARTIFACTORY_USER="your-username"
export ARTIFACTORY_PASS="your-password"
export ARTIFACTORY_URL="https://artifactory.company.com"
```

## Artifactory Tools Overview

The Artifactory tools provide comprehensive management capabilities:

### Available Tools

1. **`artifactory_healthcheck`** - System health monitoring
2. **`artifactory_get_repositories`** - List all repositories
3. **`artifactory_get_users`** - List all users
4. **`artifactory_create_user`** - Create new users
5. **`artifactory_get_repository_sizes`** - Storage analytics
6. **`artifactory_manage_permission_group`** - Permission management
7. **`artifactory_create_repository`** - Repository creation

### Tool Categories

- **System Management**: Health checks, monitoring
- **Repository Management**: Creation, listing, sizing
- **User Management**: User creation and listing
- **Permission Management**: Groups and access control

## Usage Examples

### 1. Basic Health Check

```bash
# Check if Artifactory is running
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check the health status of Artifactory"
```

**Expected Output:**
```json
{
  "status": "HEALTHY",
  "services": ["artifactory", "access", "metadata"],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 2. Repository Management

#### List All Repositories
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "List all repositories in Artifactory"
```

#### Create a Local Repository
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create a LOCAL repository called 'my-app-releases' with package type 'Generic' and description 'Application release artifacts'"
```

#### Create a Remote Repository
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create a REMOTE repository called 'maven-central' with package type 'Maven', remote URL 'https://repo1.maven.org/maven2/', and description 'Maven Central proxy'"
```

#### Create a Virtual Repository
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create a VIRTUAL repository called 'all-maven' with package type 'Maven', include repositories 'maven-local,maven-central', and description 'All Maven repositories'"
```

### 3. User Management

#### List All Users
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "List all users in Artifactory"
```

#### Create a New User
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create a new user called 'developer1' with email 'dev1@company.com' and description 'Development team member'"
```

### 4. Permission Management

#### Create a Permission Group
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create a permission group called 'developers' with description 'Development team access' and add users 'developer1,developer2' to it. Grant READ,WRITE permissions to repositories 'my-app-releases,my-app-snapshots'"
```

#### Create Admin Group
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create a permission group called 'admins' with admin privileges and auto-join enabled"
```

### 5. Storage Analytics

#### Get Repository Sizes
```bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Get size information for all repositories"
```

## Advanced Usage

### 1. Batch Operations

#### Create Multiple Repositories
```bash
# Create development repositories
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repositories: 'dev-releases' (Generic), 'dev-snapshots' (Generic), 'dev-maven' (Maven) with descriptions for development artifacts"
```

#### Create Multiple Users
```bash
# Create development team
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create users: 'dev1' (dev1@company.com), 'dev2' (dev2@company.com), 'qa1' (qa1@company.com) with appropriate descriptions"
```

### 2. CI/CD Integration

#### Automated Repository Setup
```bash
#!/bin/bash
# setup-artifactory.sh

echo "Setting up Artifactory repositories..."

# Create repositories
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repository 'ci-releases' with package type 'Generic' for CI artifacts"
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repository 'ci-snapshots' with package type 'Generic' for CI snapshots"

# Create users
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create user 'ci-bot' with email 'ci@company.com' for CI/CD automation"

# Create permission group
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create permission group 'ci-team' with users 'ci-bot' and grant READ,WRITE,DEPLOY permissions to repositories 'ci-releases,ci-snapshots'"

echo "Artifactory setup complete!"
```

### 3. Monitoring Scripts

#### Health Monitoring
```bash
#!/bin/bash
# monitor-health.sh

echo "Checking Artifactory health..."

# Run health check
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check Artifactory health status" > health_check.json

# Parse result and send alert if unhealthy
if grep -q "UNHEALTHY" health_check.json; then
    echo "ALERT: Artifactory is unhealthy!"
    # Add your alerting logic here
fi
```

#### Storage Monitoring
```bash
#!/bin/bash
# monitor-storage.sh

echo "Checking repository storage..."

# Get storage information
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Get repository size information" > storage_report.json

# Generate storage report
echo "Storage Report:"
cat storage_report.json | jq '.repositories[] | "\(.key): \(.sizeFormatted)"'
```

## Troubleshooting

### Common Issues

#### 1. Connection Refused
```bash
# Error: connection refused
# Solution: Check if Artifactory is running
docker ps | grep artifactory
# or
curl http://localhost:8081/artifactory/api/system/ping
```

#### 2. Authentication Failed
```bash
# Error: 401 Unauthorized
# Solution: Verify credentials in local.json
cat local.json | jq '.mcpServers.artifactory.options'
```

#### 3. Model Not Found
```bash
# Error: model not found
# Solution: Pull the required model
ollama pull qwen2.5:7b
```

#### 4. Tool Not Recognized
```bash
# Error: tool not found
# Solution: Rebuild the application
go build -o mcphost .
```

### Debug Mode

Enable debug logging:

```bash
# Run with debug output
DEBUG=true ./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check Artifactory health"
```

### Log Files

Check logs for detailed error information:

```bash
# Check application logs
tail -f /var/log/mcphost.log

# Check Artifactory logs (if using Docker)
docker logs artifactory
```

## Best Practices

### 1. Security
- Use environment variables for sensitive data
- Regularly rotate passwords and API keys
- Implement least-privilege access
- Use HTTPS in production

### 2. Performance
- Set appropriate timeouts
- Use connection pooling
- Monitor resource usage
- Implement caching where appropriate

### 3. Maintenance
- Regular health checks
- Storage monitoring
- User access reviews
- Repository cleanup

### 4. Documentation
- Document repository purposes
- Maintain user access lists
- Record configuration changes
- Keep runbooks updated

## Next Steps

### 1. Explore Advanced Features
- Custom repository layouts
- Advanced permission configurations
- Integration with CI/CD pipelines
- Automated backup and recovery

### 2. Integration Examples
- Jenkins integration
- GitHub Actions workflows
- Kubernetes deployments
- Monitoring dashboards

### 3. Community Resources
- [Artifactory Documentation](https://www.jfrog.com/confluence/)
- [MCP Protocol Documentation](https://modelcontextprotocol.io/)
- [Ollama Documentation](https://ollama.ai/docs)

## Support

For issues and questions:

1. **Check the troubleshooting section above**
2. **Review the main README.md file**
3. **Search existing GitHub issues**
4. **Create a new issue with detailed information**

### Issue Template

When creating an issue, include:

```markdown
**Environment:**
- OS: [e.g., macOS 14.0]
- Go version: [e.g., 1.21.0]
- Ollama version: [e.g., 0.1.0]
- Artifactory version: [e.g., 7.68.0]

**Configuration:**
```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "base_url": "http://localhost:8081",
        "username": "admin"
      }
    }
  }
}
```

**Error:**
[Paste the exact error message]

**Steps to Reproduce:**
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Expected Behavior:**
[What you expected to happen]

**Actual Behavior:**
[What actually happened]
```

---

**Happy Artifactory Management! 🚀**

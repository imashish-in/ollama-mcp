# Artifactory Tools - Quick Reference

## 🚀 Quick Start

```bash
# 1. Clone and build
git clone https://github.com/your-username/mcphost.git
cd mcphost
go build -o mcphost .

# 2. Install Ollama and pull model
brew install ollama  # macOS
ollama pull qwen2.5:7b

# 3. Start Artifactory (Docker)
docker run -d --name artifactory -p 8081:8081 releases-docker.jfrog.io/jfrog/artifactory-oss:latest

# 4. Create config
cat > local.json << EOF
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
EOF

# 5. Test connection
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check Artifactory health"
```

## 📋 Essential Commands

### Health & Status
```bash
# Health check
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check Artifactory health"

# List repositories
./mcphost -m ollama:qwen2.5:7b --config local.json -p "List all repositories"

# Get storage info
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Get repository sizes"
```

### Repository Management
```bash
# Create LOCAL repository
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repository 'my-repo' with package type 'Generic'"

# Create REMOTE repository
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create REMOTE repository 'maven-central' with package type 'Maven' and remote URL 'https://repo1.maven.org/maven2/'"

# Create VIRTUAL repository
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create VIRTUAL repository 'all-maven' with package type 'Maven' and include repositories 'local,remote'"
```

### User Management
```bash
# List users
./mcphost -m ollama:qwen2.5:7b --config local.json -p "List all users"

# Create user
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create user 'developer1' with email 'dev1@company.com'"
```

### Permission Management
```bash
# Create permission group
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create permission group 'developers' with users 'user1,user2' and grant READ,WRITE permissions to repositories 'repo1,repo2'"

# Create admin group
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create permission group 'admins' with admin privileges"
```

## ⚙️ Configuration Templates

### Development
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

### Production
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

## 🔧 Environment Variables

```bash
# Set for production
export ARTIFACTORY_USER="your-username"
export ARTIFACTORY_PASS="your-password"
export ARTIFACTORY_URL="https://artifactory.company.com"
```

## 📦 Package Types

| Type | Description | Use Case |
|------|-------------|----------|
| `Generic` | Universal binary storage | General artifacts, binaries |
| `Maven` | Java/Maven packages | Java libraries, applications |
| `Docker` | Docker images | Container images |
| `Npm` | Node.js packages | JavaScript libraries |
| `PyPI` | Python packages | Python libraries |
| `NuGet` | .NET packages | .NET libraries |
| `Gems` | Ruby packages | Ruby gems |
| `Debian` | Debian packages | Linux packages |
| `RPM` | RPM packages | Red Hat packages |

## 🏗️ Repository Types

| Type | Description | Use Case |
|------|-------------|----------|
| `LOCAL` | Local storage | Your own artifacts |
| `REMOTE` | Proxy to external | Cache external repos |
| `VIRTUAL` | Aggregates multiple | Unified access |

## 🔐 Permission Types

| Permission | Description |
|------------|-------------|
| `READ` | Download artifacts |
| `WRITE` | Upload artifacts |
| `DELETE` | Delete artifacts |
| `ANNOTATE` | Add metadata |
| `DEPLOY` | Deploy artifacts |
| `MANAGE` | Manage repository |

## 🚨 Troubleshooting

### Common Issues

```bash
# Connection refused
docker ps | grep artifactory
curl http://localhost:8081/artifactory/api/system/ping

# Authentication failed
cat local.json | jq '.mcpServers.artifactory.options'

# Model not found
ollama pull qwen2.5:7b

# Tool not recognized
go build -o mcphost .
```

### Debug Mode
```bash
DEBUG=true ./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check health"
```

## 📊 Monitoring Scripts

### Health Check
```bash
#!/bin/bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Check Artifactory health" > health.json
if grep -q "UNHEALTHY" health.json; then
    echo "ALERT: Artifactory unhealthy!"
fi
```

### Storage Report
```bash
#!/bin/bash
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Get repository sizes" > storage.json
cat storage.json | jq '.repositories[] | "\(.key): \(.sizeFormatted)"'
```

## 🔄 CI/CD Integration

### Setup Script
```bash
#!/bin/bash
# setup-artifactory.sh

echo "Setting up Artifactory..."

# Create repositories
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repository 'ci-releases' with package type 'Generic'"
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repository 'ci-snapshots' with package type 'Generic'"

# Create CI user
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create user 'ci-bot' with email 'ci@company.com'"

# Create permission group
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create permission group 'ci-team' with users 'ci-bot' and grant READ,WRITE,DEPLOY permissions to repositories 'ci-releases,ci-snapshots'"

echo "Setup complete!"
```

## 📚 Useful Commands

### Batch Operations
```bash
# Create multiple repositories
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repositories: 'dev-releases' (Generic), 'dev-snapshots' (Generic), 'dev-maven' (Maven)"

# Create multiple users
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create users: 'dev1' (dev1@company.com), 'dev2' (dev2@company.com), 'qa1' (qa1@company.com)"
```

### Advanced Usage
```bash
# Create repository with custom settings
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create LOCAL repository 'secure-repo' with package type 'Generic', description 'Secure artifacts', blacked out false, archive browsing enabled"

# Create permission group with specific privileges
./mcphost -m ollama:qwen2.5:7b --config local.json -p "Create permission group 'readonly-users' with users 'viewer1,viewer2' and grant READ permissions to all repositories"
```

## 🎯 Best Practices

### Security
- ✅ Use environment variables for credentials
- ✅ Implement least-privilege access
- ✅ Use HTTPS in production
- ✅ Regular password rotation

### Performance
- ✅ Set appropriate timeouts
- ✅ Monitor storage usage
- ✅ Regular health checks
- ✅ Implement caching

### Maintenance
- ✅ Document repository purposes
- ✅ Regular access reviews
- ✅ Storage cleanup
- ✅ Keep configurations updated

---

**📖 Full Documentation**: See `ARTIFACTORY_GUIDE.md` for complete guide  
**🐛 Issues**: Check troubleshooting section or create GitHub issue  
**🚀 Happy Artifactory Management!**

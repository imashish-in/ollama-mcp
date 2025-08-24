# Artifactory Configuration Guide

This guide explains how to configure multiple Artifactory instances in the `local.json` configuration file.

## 📋 Configuration Structure

The `local.json` file supports multiple Artifactory instances with comprehensive configuration options.

### Basic Structure

```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "instances": {
          "instance-name": {
            // Instance configuration
          }
        },
        "defaultInstance": "instance-name",
        "commonSettings": {
          // Common settings for all instances
        }
      }
    }
  },
  "globalSettings": {
    "artifactory": {
      // Global Artifactory settings
    }
  }
}
```

## 🔧 Instance Configuration

Each Artifactory instance can be configured with the following parameters:

### Required Parameters
- **`name`** - Human-readable name for the instance
- **`url`** - Artifactory server URL (e.g., `http://localhost`, `https://artifactory.example.com`)

### Authentication Parameters (choose one)
- **`username`** - Username for authentication
- **`password`** - Password for authentication
- **`apiKey`** - API key for authentication (alternative to username/password)

### Optional Parameters
- **`timeout`** - Request timeout in seconds (default: 30, max: 120)
- **`verifySSL`** - Whether to verify SSL certificates (default: true)
- **`description`** - Description of the instance

## 📝 Example Configurations

### Single Localhost Instance

```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "instances": {
          "default": {
            "name": "Default Artifactory",
            "url": "http://localhost",
            "username": "admin",
            "password": "B@55w0rd",
            "timeout": 30,
            "verifySSL": false,
            "description": "Local Artifactory instance"
          }
        },
        "defaultInstance": "default",
        "commonSettings": {
          "maxRetries": 3,
          "retryDelay": 5,
          "userAgent": "MCPHost-Artifactory-Client/1.0",
          "logLevel": "info"
        }
      }
    }
  }
}
```

### Multiple Instances (Development, Staging, Production)

```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "instances": {
          "development": {
            "name": "Development Artifactory",
            "url": "http://localhost:8081",
            "username": "admin",
            "password": "B@55w0rd",
            "timeout": 30,
            "verifySSL": false,
            "description": "Development environment Artifactory instance"
          },
          "staging": {
            "name": "Staging Artifactory",
            "url": "https://staging-artifactory.company.com",
            "username": "ci-user",
            "password": "staging-password",
            "timeout": 45,
            "verifySSL": true,
            "description": "Staging environment Artifactory instance"
          },
          "production": {
            "name": "Production Artifactory",
            "url": "https://artifactory.company.com",
            "apiKey": "AKCp8...",
            "timeout": 60,
            "verifySSL": true,
            "description": "Production environment Artifactory instance"
          }
        },
        "defaultInstance": "development",
        "commonSettings": {
          "maxRetries": 3,
          "retryDelay": 5,
          "userAgent": "MCPHost-Artifactory-Client/1.0",
          "logLevel": "info"
        }
      }
    }
  }
}
```

### Enterprise Setup with Multiple Environments

```json
{
  "mcpServers": {
    "artifactory": {
      "type": "builtin",
      "name": "artifactory",
      "options": {
        "instances": {
          "local": {
            "name": "Local Development",
            "url": "http://localhost:8081",
            "username": "admin",
            "password": "B@55w0rd",
            "timeout": 30,
            "verifySSL": false,
            "description": "Local development instance"
          },
          "dev": {
            "name": "Development Server",
            "url": "https://dev-artifactory.company.com",
            "username": "dev-user",
            "password": "dev-password",
            "timeout": 45,
            "verifySSL": true,
            "description": "Shared development server"
          },
          "qa": {
            "name": "QA Server",
            "url": "https://qa-artifactory.company.com",
            "username": "qa-user",
            "password": "qa-password",
            "timeout": 45,
            "verifySSL": true,
            "description": "QA testing server"
          },
          "staging": {
            "name": "Staging Server",
            "url": "https://staging-artifactory.company.com",
            "apiKey": "AKCp8staging...",
            "timeout": 60,
            "verifySSL": true,
            "description": "Pre-production staging server"
          },
          "prod": {
            "name": "Production Server",
            "url": "https://artifactory.company.com",
            "apiKey": "AKCp8production...",
            "timeout": 60,
            "verifySSL": true,
            "description": "Production server"
          }
        },
        "defaultInstance": "local",
        "commonSettings": {
          "maxRetries": 3,
          "retryDelay": 5,
          "userAgent": "MCPHost-Artifactory-Client/1.0",
          "logLevel": "info"
        }
      }
    }
  }
}
```

## 🚀 Usage Examples

### MCP Server Usage

```bash
# Use default instance
./mcphost --config=local.json -m ollama:qwen3:8b -p "List repositories from Artifactory"

# Use specific instance
./mcphost --config=local.json -m ollama:qwen3:8b -p "List repositories from staging Artifactory instance"

# Create repository in specific environment
./mcphost --config=local.json -m ollama:qwen3:8b -p "Create a LOCAL repository called 'my-app-releases' in the production Artifactory instance"
```

### CLI Tool Usage

```bash
# Use default instance
mcphost artifactory-permissions list

# Use specific instance
mcphost artifactory-permissions list --instance staging

# Create permission in specific environment
mcphost artifactory-permissions create --name "prod-perm" --users "prod-user" --instance production

# Override configuration values
mcphost artifactory-permissions create --name "test-perm" --users "user1" --base-url "http://custom-host:8081"
```

## 🔐 Security Best Practices

1. **Use API Keys** - Prefer API keys over username/password when possible
2. **Environment Variables** - Consider using environment variables for sensitive data
3. **SSL Verification** - Enable SSL verification for production instances
4. **Timeouts** - Set appropriate timeouts for your network conditions
5. **Access Control** - Limit access to configuration files

## 🔧 Common Settings

The `commonSettings` section allows you to configure behavior across all instances:

- **`maxRetries`** - Maximum number of retry attempts (default: 3)
- **`retryDelay`** - Delay between retries in seconds (default: 5)
- **`userAgent`** - User agent string for requests
- **`logLevel`** - Logging level (info, debug, warn, error)

## 📚 Related Documentation

- [MCPHost User Guide](./README.md)
- [Artifactory REST API Documentation](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API)

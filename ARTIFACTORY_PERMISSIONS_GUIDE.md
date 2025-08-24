# Artifactory Permissions Management Tool

The Artifactory permissions management tool provides granular control over permission targets in JFrog Artifactory. This tool allows you to create, list, and delete permission targets with fine-grained access control over users, groups, repositories, and privileges.

## Overview

This tool complements the existing Artifactory builtin server by providing a dedicated CLI interface for permission management. It supports:

- **Permission Names**: Custom permission target names
- **Users**: Individual user access control
- **Groups**: Group-based access control
- **Repositories**: Repository-specific permissions
- **Privileges**: Granular permission levels

## Basic Usage

```bash
mcphost artifactory-permissions [command] [options]
```

## Commands

### 1. **Create Permission Target**

Create a new permission target with specified users, groups, repositories, and privileges.

```bash
mcphost artifactory-permissions create [flags]
```

#### Required Flags
- `--name`: Permission target name (required)

#### Optional Flags
- `--users`: Comma-separated list of users
- `--groups`: Comma-separated list of groups
- `--repos`: Comma-separated list of repositories (use 'ANY' for all repositories)
- `--privileges`: Comma-separated list of privileges (default: READ)

#### Examples

**Create developer permissions:**
```bash
mcphost artifactory-permissions create \
  --name "dev-permissions" \
  --users "user1,user2,user3" \
  --groups "developers" \
  --repos "maven-local,npm-local" \
  --privileges "READ,WRITE"
```

**Create admin permissions:**
```bash
mcphost artifactory-permissions create \
  --name "admin-permissions" \
  --users "admin1,admin2" \
  --privileges "READ,WRITE,DELETE,ANNOTATE,DEPLOY" \
  --repos "ANY"
```

**Create read-only permissions:**
```bash
mcphost artifactory-permissions create \
  --name "read-only-permissions" \
  --groups "viewers,guests" \
  --repos "maven-local,npm-local,docker-local" \
  --privileges "READ"
```

### 2. **List Permission Targets**

List all permission targets in the Artifactory instance.

```bash
mcphost artifactory-permissions list [flags]
```

#### Example
```bash
mcphost artifactory-permissions list
```

**Output:**
```
📋 Found 3 permission targets:

1. dev-permissions
   Repositories: maven-local, npm-local
   Users:
     - user1: READ, WRITE
     - user2: READ, WRITE
     - user3: READ, WRITE
   Groups:
     - developers: READ, WRITE

2. admin-permissions
   Repositories: ANY
   Users:
     - admin1: READ, WRITE, DELETE, ANNOTATE, DEPLOY
     - admin2: READ, WRITE, DELETE, ANNOTATE, DEPLOY

3. read-only-permissions
   Repositories: maven-local, npm-local, docker-local
   Groups:
     - viewers: READ
     - guests: READ
```

### 3. **Delete Permission Target**

Delete a permission target from Artifactory.

```bash
mcphost artifactory-permissions delete [flags]
```

#### Required Flags
- `--name`: Permission target name to delete (required)

#### Example
```bash
mcphost artifactory-permissions delete --name "old-permissions"
```

## Common Flags

All commands support these common flags:

- `--base-url`: Artifactory base URL (default: http://localhost)
- `--username`: Username for authentication (default: admin)
- `--password`: Password for authentication (default: B@55w0rd)
- `--api-key`: API key for authentication (alternative to username/password)
- `--timeout`: Timeout in seconds (max 120, default: 30)

## Privilege Types

The following privileges are supported:

| Privilege | Description | Use Case |
|-----------|-------------|----------|
| `READ` | Read access to artifacts | View and download artifacts |
| `WRITE` | Write access to artifacts | Upload and deploy artifacts |
| `DELETE` | Delete artifacts | Remove artifacts from repositories |
| `ANNOTATE` | Add metadata and properties | Add properties to artifacts |
| `DEPLOY` | Deploy artifacts | Deploy to repositories |
| `MANAGE` | Manage repository settings | Configure repository properties |
| `ADMIN` | Full administrative access | Complete repository control |

## Repository Access

### Specific Repositories
```bash
--repos "maven-local,npm-local,docker-local"
```

### All Repositories
```bash
--repos "ANY"
```

### No Repositories (Build permissions)
```bash
--repos ""
```

## Advanced Examples

### 1. **Multi-Environment Permissions**

**Development Environment:**
```bash
mcphost artifactory-permissions create \
  --name "dev-team-permissions" \
  --users "dev1,dev2,dev3" \
  --groups "development-team" \
  --repos "dev-maven-local,dev-npm-local" \
  --privileges "READ,WRITE,DEPLOY" \
  --base-url "http://localhost:8081"
```

**Production Environment:**
```bash
mcphost artifactory-permissions create \
  --name "prod-team-permissions" \
  --users "prod-admin1,prod-admin2" \
  --groups "production-team" \
  --repos "prod-maven-local,prod-npm-local" \
  --privileges "READ,WRITE,DELETE,ANNOTATE,DEPLOY" \
  --base-url "https://artifactory.company.com"
```

### 2. **Role-Based Access Control**

**Developers:**
```bash
mcphost artifactory-permissions create \
  --name "developer-permissions" \
  --groups "developers" \
  --repos "maven-local,npm-local" \
  --privileges "READ,WRITE,DEPLOY"
```

**QA Team:**
```bash
mcphost artifactory-permissions create \
  --name "qa-permissions" \
  --groups "qa-team" \
  --repos "maven-local,npm-local,qa-releases" \
  --privileges "READ,WRITE"
```

**Release Managers:**
```bash
mcphost artifactory-permissions create \
  --name "release-permissions" \
  --users "release-manager1,release-manager2" \
  --repos "maven-releases,npm-releases" \
  --privileges "READ,WRITE,DELETE,ANNOTATE,DEPLOY"
```

### 3. **Project-Specific Permissions**

**Frontend Project:**
```bash
mcphost artifactory-permissions create \
  --name "frontend-project-permissions" \
  --users "frontend-dev1,frontend-dev2" \
  --groups "frontend-team" \
  --repos "frontend-npm-local,frontend-npm-remote" \
  --privileges "READ,WRITE,DEPLOY"
```

**Backend Project:**
```bash
mcphost artifactory-permissions create \
  --name "backend-project-permissions" \
  --users "backend-dev1,backend-dev2" \
  --groups "backend-team" \
  --repos "backend-maven-local,backend-maven-remote" \
  --privileges "READ,WRITE,DEPLOY"
```

## Configuration Examples

### Development Environment
```bash
# Use default localhost settings
mcphost artifactory-permissions create \
  --name "dev-permissions" \
  --users "developer1" \
  --repos "maven-local" \
  --privileges "READ,WRITE"
```

### Production Environment
```bash
# Use production Artifactory instance
mcphost artifactory-permissions create \
  --name "prod-permissions" \
  --users "admin1" \
  --repos "ANY" \
  --privileges "READ,WRITE,DELETE,ANNOTATE,DEPLOY" \
  --base-url "https://artifactory.company.com" \
  --username "admin" \
  --password "secure-password" \
  --timeout 60
```

### Using API Key Authentication
```bash
mcphost artifactory-permissions create \
  --name "api-permissions" \
  --users "service-account" \
  --repos "maven-local" \
  --privileges "READ,WRITE" \
  --base-url "https://artifactory.company.com" \
  --api-key "AKCp8..."
```

## Error Handling

The tool provides clear error messages for common issues:

- **Invalid URL**: Shows URL validation errors
- **Authentication Failed**: Indicates credential issues
- **Permission Denied**: Shows when user lacks admin privileges
- **Repository Not Found**: Indicates non-existent repositories
- **User/Group Not Found**: Shows when specified users/groups don't exist

## Best Practices

### 1. **Naming Conventions**
- Use descriptive permission names: `dev-team-permissions`, `prod-admin-permissions`
- Include environment in names: `dev-`, `staging-`, `prod-`
- Use consistent naming patterns across your organization

### 2. **Principle of Least Privilege**
- Grant only necessary privileges
- Start with minimal permissions and add as needed
- Regularly review and audit permissions

### 3. **Group-Based Management**
- Prefer groups over individual users when possible
- Use groups for role-based access control
- Keep user lists in groups manageable

### 4. **Repository Organization**
- Use specific repositories rather than "ANY" when possible
- Organize repositories by project, team, or environment
- Use repository naming conventions

### 5. **Regular Maintenance**
- List permissions regularly to audit access
- Remove unused permission targets
- Update permissions when team structures change

## Integration with Existing Tools

This permissions tool works alongside the existing Artifactory builtin server:

- **Health Check**: Verify Artifactory is running before managing permissions
- **Repository Management**: Create repositories before setting permissions
- **User Management**: Create users before adding them to permissions
- **Group Management**: Use the existing group management tool for complex group setups

## Troubleshooting

### Common Issues

1. **"Permission denied" errors**
   - Ensure you're using admin credentials
   - Check if your user has admin privileges

2. **"Repository not found" errors**
   - Verify repository names are correct
   - Use `mcphost artifactory-permissions list` to see available repositories

3. **"User not found" errors**
   - Ensure users exist in Artifactory
   - Check user names for typos

4. **"Group not found" errors**
   - Ensure groups exist in Artifactory
   - Use the existing group management tool to create groups first

### Debug Mode
Enable debug logging for detailed error information:
```bash
mcphost artifactory-permissions create --debug --name "test-permissions" ...
```

## Security Considerations

1. **Credential Management**
   - Use environment variables for production credentials
   - Avoid hardcoding passwords in scripts
   - Use API keys for service accounts

2. **Network Security**
   - Use HTTPS for production environments
   - Configure proper firewall rules
   - Use VPN access when needed

3. **Access Auditing**
   - Regularly review permission targets
   - Monitor access logs
   - Implement least privilege access

4. **Backup and Recovery**
   - Document permission configurations
   - Backup permission settings
   - Have recovery procedures in place

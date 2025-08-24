# Support Bundle Analysis Tool

The Support Bundle Analysis tool is designed to analyze Artifactory support bundles by searching for error logs, warnings, and exceptions in unzipped nested folders. This tool is particularly useful for troubleshooting Artifactory issues by automatically extracting and analyzing log files from support bundles.

## Overview

Support bundles from Artifactory contain comprehensive diagnostic information including logs, configuration files, and system information. This tool helps you:

- **Extract nested archives** automatically
- **Search for error patterns** across all log files
- **Categorize findings** by error type (ERROR, WARNING, EXCEPTION)
- **Provide context** around each match
- **Handle large bundles** efficiently with configurable limits

## Features

### 🔍 **Intelligent Search**
- Multiple search patterns support
- Case-sensitive and case-insensitive search
- Regular expression support
- Configurable file type filtering

### 📦 **Archive Handling**
- Automatic extraction of nested ZIP archives
- Support for multiple archive formats
- Temporary extraction with automatic cleanup
- Archive path tracking in results

### 📊 **Result Categorization**
- **Error Logs**: Contains ERROR patterns
- **Warning Logs**: Contains WARNING patterns  
- **Exception Logs**: Contains EXCEPTION patterns
- **Context Lines**: Surrounding lines for better understanding

### ⚡ **Performance Optimizations**
- Binary file detection and skipping
- Configurable result limits
- Context cancellation support
- Efficient file processing

## Usage

### Basic Usage

```bash
# Analyze support bundle with default settings
mcphost -m ollama:qwen3:8b -p "Analyze the support bundle in ./support-bundle folder"
```

### Advanced Usage

```bash
# Analyze with custom search patterns
mcphost -m ollama:qwen3:8b -p "Analyze support bundle in ./support-bundle with search patterns 'ERROR,CRITICAL,FAILED' and include 5 context lines"
```

## Configuration

### Tool Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `bundle_path` | string | `./support-bundle` | Path to the support bundle folder |
| `search_patterns` | string | `ERROR,WARNING,Exception,failed,error` | Comma-separated search patterns |
| `file_types` | string | `.log,.txt,.out` | Comma-separated file types to search |
| `case_sensitive` | boolean | `false` | Case sensitive search |
| `include_archives` | boolean | `true` | Search inside nested archives |
| `max_results` | number | `100` | Maximum results per pattern |
| `context_lines` | number | `2` | Context lines around matches |
| `extract_archives` | boolean | `true` | Extract archives for analysis |

### Example Configurations

#### Basic Error Analysis
```json
{
  "mcpServers": {
    "support-bundle": {
      "type": "builtin",
      "name": "support-bundle"
    }
  }
}
```

#### Custom Search Configuration
```json
{
  "mcpServers": {
    "support-bundle": {
      "type": "builtin",
      "name": "support-bundle",
      "options": {
        "search_patterns": "ERROR,CRITICAL,FAILED,Exception",
        "file_types": ".log,.txt,.out,.err",
        "max_results": 200,
        "context_lines": 5,
        "case_sensitive": false
      }
    }
  }
}
```

## Search Patterns

### Common Error Patterns

#### Artifactory-Specific
- `ERROR` - General error messages
- `WARNING` - Warning messages
- `Exception` - Java exceptions
- `failed` - Failed operations
- `CRITICAL` - Critical errors
- `FATAL` - Fatal errors

#### System Patterns
- `OutOfMemoryError` - Memory issues
- `ConnectionException` - Connection problems
- `TimeoutException` - Timeout issues
- `PermissionDenied` - Permission errors
- `FileNotFoundException` - Missing files

#### Custom Patterns
- `database.*error` - Database-related errors
- `authentication.*failed` - Authentication issues
- `deployment.*failed` - Deployment problems

### Pattern Examples

```bash
# Search for specific error types
"ERROR,CRITICAL,FATAL"

# Search for exceptions
"Exception,Error,Throwable"

# Search for specific operations
"deploy.*failed,upload.*error,download.*failed"

# Search for system issues
"OutOfMemoryError,ConnectionException,TimeoutException"
```

## File Types

### Supported File Types
- `.log` - Log files
- `.txt` - Text files
- `.out` - Output files
- `.err` - Error files
- `.debug` - Debug files
- `.trace` - Trace files

### Archive Formats
- `.zip` - ZIP archives
- `.tar` - TAR archives
- `.gz` - Gzipped files
- `.bz2` - Bzip2 files
- `.xz` - XZ compressed files
- `.rar` - RAR archives
- `.7z` - 7-Zip archives

## Output Format

### Analysis Results

The tool returns a structured JSON response with the following structure:

```json
{
  "bundle_path": "./support-bundle",
  "total_files": 150,
  "error_logs": [
    {
      "file_path": "/path/to/artifactory.log",
      "line_number": 1234,
      "full_line": "2024-01-15 10:30:45 ERROR [http-nio-8081-exec-5] o.a.w.s.ArtifactoryFilter - Authentication failed",
      "matched_text": "ERROR",
      "context": "> 1234: 2024-01-15 10:30:45 ERROR [http-nio-8081-exec-5] o.a.w.s.ArtifactoryFilter - Authentication failed",
      "file_type": ".log",
      "archive_path": "artifactory-logs.zip/artifactory.log"
    }
  ],
  "warning_logs": [...],
  "exception_logs": [...],
  "search_patterns": ["ERROR", "WARNING", "Exception"],
  "analysis_time": "2024-01-15T10:30:45Z",
  "duration": "2.5s"
}
```

### Result Fields

| Field | Type | Description |
|-------|------|-------------|
| `file_path` | string | Full path to the file containing the match |
| `line_number` | integer | Line number where the match was found |
| `full_line` | string | Complete line containing the match |
| `matched_text` | string | The actual text that matched the pattern |
| `context` | string | Surrounding lines for context |
| `file_type` | string | File extension |
| `archive_path` | string | Path within archive (if applicable) |

## Use Cases

### 1. **Troubleshooting Artifactory Issues**
```bash
# Find all authentication errors
mcphost -m ollama:qwen3:8b -p "Analyze support bundle for authentication errors and connection issues"
```

### 2. **Performance Analysis**
```bash
# Find performance-related issues
mcphost -m ollama:qwen3:8b -p "Search for timeout errors and memory issues in the support bundle"
```

### 3. **Deployment Problems**
```bash
# Find deployment-related errors
mcphost -m ollama:qwen3:8b -p "Look for deployment failures and upload/download errors"
```

### 4. **System Health Check**
```bash
# Comprehensive system analysis
mcphost -m ollama:qwen3:8b -p "Analyze support bundle for all types of errors, warnings, and exceptions"
```

## Best Practices

### 1. **Search Pattern Selection**
- Start with broad patterns like `ERROR,Exception`
- Refine with specific patterns based on initial findings
- Use case-insensitive search for better coverage
- Include common variations of error terms

### 2. **File Type Filtering**
- Focus on `.log` files for most issues
- Include `.txt` and `.out` for additional context
- Consider `.err` files for specific error logs

### 3. **Result Management**
- Set appropriate `max_results` to avoid overwhelming output
- Use `context_lines` to get meaningful context
- Review categorized results (error_logs, warning_logs, exception_logs)

### 4. **Archive Handling**
- Enable `include_archives` for comprehensive analysis
- Use `extract_archives` for better performance
- Monitor temporary directory usage for large bundles

## Troubleshooting

### Common Issues

#### 1. **No Results Found**
- Check if the bundle path is correct
- Verify search patterns are appropriate
- Ensure file types include the relevant extensions
- Check if archives need to be extracted

#### 2. **Too Many Results**
- Reduce `max_results` parameter
- Use more specific search patterns
- Filter by specific file types
- Enable case-sensitive search

#### 3. **Performance Issues**
- Reduce `context_lines` parameter
- Disable archive extraction for large bundles
- Use more specific file type filters
- Limit search patterns to essential ones

#### 4. **Memory Issues**
- Process smaller bundles
- Reduce `max_results` significantly
- Disable archive extraction
- Use more restrictive file type filters

## Integration Examples

### With Ollama Models

```bash
# Basic analysis
./mcphost -m ollama:qwen3:8b -p "Analyze the support bundle in ./support-bundle for any errors"

# Detailed analysis with custom patterns
./mcphost -m ollama:qwen3:8b -p "Search for authentication errors, connection issues, and deployment failures in the support bundle with 5 context lines"

# Performance analysis
./mcphost -m ollama:qwen3:8b -p "Find all timeout errors, memory issues, and performance problems in the support bundle"
```

### Configuration File Example

```json
{
  "mcpServers": {
    "support-bundle": {
      "type": "builtin",
      "name": "support-bundle"
    },
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

## Security Considerations

### File Access
- The tool only reads files, never modifies them
- Temporary extraction directories are automatically cleaned up
- No sensitive data is logged or stored permanently

### Archive Safety
- Archives are extracted to temporary directories
- Extraction is done safely with proper error handling
- Malformed archives are skipped gracefully

### Resource Management
- Configurable limits prevent resource exhaustion
- Context cancellation support for long-running operations
- Memory-efficient processing of large files

## Performance Tips

### 1. **Optimize Search Patterns**
- Use specific patterns rather than broad ones
- Combine related patterns with OR logic
- Avoid overly complex regular expressions

### 2. **File Type Filtering**
- Only search relevant file types
- Exclude binary files and large data files
- Focus on log files for most issues

### 3. **Result Limits**
- Set appropriate `max_results` based on your needs
- Use smaller `context_lines` for faster processing
- Process results in batches if needed

### 4. **Archive Handling**
- Disable archive extraction for quick scans
- Use archive extraction only when necessary
- Monitor temporary directory usage

This tool provides a powerful way to analyze Artifactory support bundles and quickly identify issues that need attention. It's particularly useful for troubleshooting complex Artifactory deployments and understanding system behavior.

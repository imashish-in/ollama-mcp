# Log Analyzer MCP Tool Guide

## Overview

The Log Analyzer MCP tool is a powerful utility for analyzing log files to find errors, warnings, and other important patterns in extracted support bundle content. It provides comprehensive analysis with severity categorization, timestamp extraction, and context-aware results.

## Features

- **Multi-pattern Search**: Searches for ERROR, WARNING, Exception, failed, and other patterns
- **Severity Categorization**: Automatically categorizes by ERROR, WARNING, INFO, DEBUG levels
- **Timestamp Extraction**: Extracts timestamps from log entries for timeline analysis
- **Context Lines**: Provides surrounding context for better understanding
- **Multiple File Types**: Supports .log, .out, .err, .txt files
- **Configurable Search**: Customizable search patterns and case sensitivity
- **Performance Metrics**: Includes analysis time and file statistics

## Tool Parameters

### Required Parameters

- **`source_path`** (string): Path to the directory containing log files to analyze

### Optional Parameters

- **`search_patterns`** (string): Comma-separated search patterns (default: "ERROR,WARNING,Exception,failed,error,warn,CRITICAL,FATAL")
- **`file_types`** (string): Comma-separated file types to search (default: ".log,.out,.err,.txt")
- **`case_sensitive`** (boolean): Case sensitive search (default: false)
- **`max_results`** (number): Maximum results per pattern (default: 100)
- **`context_lines`** (number): Context lines around matches (default: 2)
- **`include_timestamps`** (boolean): Extract timestamps (default: true)
- **`severity_levels`** (string): Comma-separated severity levels (default: "ERROR,WARNING,INFO,DEBUG,CRITICAL,FATAL")

## Usage Examples

### Basic Usage

Analyze all log files in the support bundle:

```json
{
  "source_path": "./support-bundle"
}
```

### Custom Search Patterns

Search for specific patterns:

```json
{
  "source_path": "./support-bundle",
  "search_patterns": "ERROR,WARNING,Exception,failed,timeout,connection refused"
}
```

### Severity-based Analysis

Focus on specific severity levels:

```json
{
  "source_path": "./support-bundle",
  "severity_levels": "ERROR,CRITICAL,FATAL"
}
```

## Response Format

The tool returns a comprehensive JSON object:

```json
{
  "source_path": "string",
  "total_files": "number",
  "total_errors": "number",
  "total_warnings": "number",
  "total_info": "number",
  "error_logs": ["array of error results"],
  "warning_logs": ["array of warning results"],
  "info_logs": ["array of info results"],
  "search_patterns": ["array of patterns used"],
  "analysis_time": "timestamp",
  "duration": "string",
  "file_stats": {"file path": "result count"},
  "severity_stats": {"severity": "count"}
}
```

### Result Structure

Each log result contains:

```json
{
  "file_path": "string",
  "line_number": "number",
  "full_line": "string",
  "matched_text": "string",
  "severity": "string",
  "timestamp": "string",
  "context": "string"
}
```

## Real Analysis Results

Based on the extracted support bundle, the tool found:

### **Key Issues Identified:**

1. **Service Registry Connection Failures**
   - Multiple 404 errors when trying to connect to service registry
   - "Service registry ping failed" errors across multiple services

2. **Access Service Issues**
   - "Timed out waiting for Access Join" warnings
   - "The Access service is not ready" exceptions

3. **Authentication Problems**
   - 401 status codes in frontend requests
   - Authentication timeout warnings

4. **Cluster Join Failures**
   - Retry attempts for cluster join operations
   - Service unavailability exceptions

### **Error Patterns Found:**

- **ERROR**: Service registry ping failures, connection errors
- **WARNING**: Timeout warnings, service not ready warnings
- **Exception**: ServiceUnavailableException, authentication exceptions
- **Failed**: Request failures, ping failures, connection failures

## Integration with MCP

This tool is available as an MCP server named `log-analyzer` and can be used with any MCP-compatible client or AI model.

### Using with mcphost

```bash
# Basic error and warning analysis
./mcphost -m ollama:qwen3:8b -p "Use the log-analyzer server and call analyze_logs with source_path='./support-bundle' to find all errors and warnings"

# Focus on critical errors only
./mcphost -m ollama:qwen3:8b -p "Use the log-analyzer server and call analyze_logs with source_path='./support-bundle' and severity_levels='ERROR,CRITICAL,FATAL' to find critical issues"
```

## Use Cases

1. **Support Bundle Analysis**: Analyze extracted support bundles for troubleshooting
2. **Error Detection**: Find and categorize errors across multiple log files
3. **Performance Monitoring**: Identify warning patterns and performance issues
4. **Security Analysis**: Detect authentication and authorization failures
5. **Service Health**: Monitor service availability and connection issues

## Performance Considerations

- The tool processes log files sequentially for memory efficiency
- Large log files may take time to analyze
- Context lines increase processing time but provide better insights
- Consider using `max_results` to limit output for large datasets

## Dependencies

The tool uses Go's standard library for:
- File system operations
- Regular expression matching
- JSON marshaling/unmarshaling
- Time parsing and formatting

No external dependencies are required.

# Archive Extractor MCP Tool Guide

## Overview

The Archive Extractor MCP tool is a powerful utility for recursively extracting compressed files from directories. It supports multiple archive formats and can handle deeply nested archives with automatic recursive extraction until no more archives are found.

## Features

- **Multiple Archive Formats**: Supports ZIP, TAR, GZ, BZ2, XZ, and their compressed variants
- **Deep Recursive Extraction**: Automatically extracts nested archives until no more are found (up to 10 levels deep)
- **Comprehensive Error Handling**: Provides detailed error reporting for failed extractions
- **Flexible Output**: Can extract to the source directory or a custom output directory
- **Performance Tracking**: Includes timing information for extraction operations

## Supported Archive Formats

- **ZIP**: `.zip` files
- **TAR**: `.tar` files
- **Compressed TAR**: `.tar.gz`, `.tgz`, `.tar.bz2`, `.tbz2`, `.tar.xz`, `.txz`
- **GZIP**: `.gz` files (single files)
- **BZIP2**: `.bz2` files (single files)
- **XZ**: `.xz` files (single files)

## Tool Parameters

### Required Parameters

- **`source_path`** (string): Path to the directory containing compressed files to extract

### Optional Parameters

- **`output_dir`** (string): Output directory for extracted files (defaults to source directory)
- **`recursive`** (boolean): Whether to recursively extract nested archives (default: true). When enabled, continues extracting until no more archives are found.

## Usage Examples

### Basic Usage

Extract all archives from a directory:

```json
{
  "source_path": "./support-bundle"
}
```

### Custom Output Directory

Extract archives to a specific output directory:

```json
{
  "source_path": "./support-bundle",
  "output_dir": "./extracted-files"
}
```

### Non-Recursive Extraction

Extract only the top-level archives without processing nested archives:

```json
{
  "source_path": "./support-bundle",
  "recursive": false
}
```

## 🔄 Recursive Extraction Process

The tool uses an iterative approach for deep recursive extraction:

1. **Initial Scan**: Identifies all archive files in the source directory
2. **First Level Extraction**: Extracts all found archives
3. **Recursive Processing**: 
   - Scans newly extracted content for more archives
   - Extracts any found archives
   - Removes original archive files after successful extraction
   - Repeats until no more archives are found (max 10 iterations)
4. **Completion**: Reports total files extracted and any errors

### Safety Features

- **Maximum Iterations**: Prevents infinite loops (max 10 iterations)
- **File Cleanup**: Removes original archives after successful extraction
- **Error Recovery**: Continues processing even if individual archives fail

## Response Format

The tool returns a JSON object with the following structure:

```json
{
  "source_path": "string",
  "extracted_files": ["array of file paths"],
  "total_files": "number",
  "errors": ["array of error messages"],
  "message": "string",
  "duration": "string"
}
```

### Response Fields

- **`source_path`**: The original source directory path
- **`extracted_files`**: Array of paths to all successfully extracted files
- **`total_files`**: Total number of files extracted
- **`errors`**: Array of error messages for failed extractions (if any)
- **`message`**: Summary message about the extraction operation
- **`duration`**: Time taken for the extraction operation

## Example Response

```json
{
  "source_path": "./support-bundle",
  "extracted_files": [
    "./support-bundle/logs/artifactory.log",
    "./support-bundle/config/artifactory.config",
    "./support-bundle/data/artifactory.db"
  ],
  "total_files": 3,
  "errors": [],
  "message": "Successfully extracted 3 files from archives",
  "duration": "2.5s"
}
```

## Error Handling

The tool provides comprehensive error reporting:

- **File Access Errors**: When files cannot be read or accessed
- **Archive Format Errors**: When unsupported archive formats are encountered
- **Extraction Errors**: When individual files fail to extract
- **Directory Creation Errors**: When output directories cannot be created

## Integration with MCP

This tool is available as an MCP server named `archive-extractor` and can be used with any MCP-compatible client or AI model.

### Using with mcphost

```bash
# List available builtin servers
./mcphost list-builtin

# Use the archive extractor with an Ollama model
./mcphost -m ollama:qwen3:8b -p "Extract all compressed files from the support-bundle directory recursively"
```

## Use Cases

1. **Support Bundle Analysis**: Extract and analyze compressed support bundles
2. **Log Analysis**: Extract compressed log files for analysis
3. **Backup Restoration**: Extract backup archives to restore data
4. **Development**: Extract source code archives and dependencies
5. **Data Processing**: Extract compressed datasets for analysis

## Performance Considerations

- The tool processes archives sequentially to avoid memory issues
- Large archives may take significant time to extract
- Recursive extraction can be resource-intensive for deeply nested archives
- Consider using `recursive: false` for large archive collections if nested extraction is not needed

## Dependencies

The tool requires the following system utilities for certain archive formats:
- **xz**: For XZ compressed archives (`.xz`, `.tar.xz`)
- **bunzip2**: For BZIP2 compressed archives (`.bz2`, `.tar.bz2`)

Most other formats are handled natively by Go's standard library.

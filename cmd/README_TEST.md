# Archive Extraction Test Program

This directory contains a Go-based test program for testing the improved archive extraction functionality.

## Test Program: `test_archive_extraction.go`

This program creates a test structure with nested zip files to demonstrate the recursive extraction capabilities.

### Features

- Creates a 3-level nested zip structure
- Demonstrates deep recursive extraction
- Pure Go implementation (no shell scripts)
- Clean test environment setup

### Usage

1. **Build the test program:**
   ```bash
   go build -o test_archive_extraction cmd/test_archive_extraction.go
   ```

2. **Run the test setup:**
   ```bash
   ./test_archive_extraction
   ```

3. **Test the extraction:**
   ```bash
   ./mcphost --config=local.json -m ollama:qwen3:8b -p "Extract all archives recursively from the test_nested_extraction directory"
   ```

### Test Structure

The program creates the following nested structure:

```
test_nested_extraction/
├── level1.zip (contains level2.zip)
└── level2.zip (contains level3.zip)
    └── level3.zip (contains text files)
```

### Expected Results

After extraction, you should see all the original text files extracted:
- `level1.txt`
- `data1.txt`
- `level2.txt`
- `data2.txt`
- `level3.txt`
- `data3.txt`

### Benefits of Go Implementation

- **Cross-platform**: Works on all platforms that support Go
- **No dependencies**: No external tools required
- **Consistent**: Same behavior across different environments
- **Maintainable**: Easy to modify and extend
- **Integrated**: Part of the main codebase

### Integration with MCP Tools

The test program works seamlessly with the MCP archive extraction tools:
- `extract_archives` - Main extraction tool
- `support_bundle` - Support bundle analysis
- `log_analyzer` - Log analysis after extraction

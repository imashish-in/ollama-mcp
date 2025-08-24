# Search Tool Guide

The search tool allows you to search for words in specific folders across all file types and return relevant results with complete sentences and file names.

## Basic Usage

```bash
mcphost search [word] [folder]
```

### Examples

```bash
# Search for "function" in the internal directory
mcphost search "function" ./internal

# Search for "error" in the cmd directory with case sensitivity
mcphost search "error" ./cmd --case-sensitive

# Search for "test" in the current directory, matching whole words only
mcphost search "test" . --whole-word --max-results 50

# Search for "TODO" using regex pattern
mcphost search "TODO" . --regex --exclude-dirs .git,node_modules
```

## Features

### 1. **Case Sensitivity**
- `--case-sensitive`: Perform case-sensitive search
- Default: Case-insensitive search

### 2. **Whole Word Matching**
- `--whole-word`: Match whole words only (prevents partial matches)
- Example: Searching for "test" with `--whole-word` won't match "testing" or "attest"

### 3. **Regular Expression Support**
- `--regex`: Treat search term as regular expression
- Allows complex pattern matching
- Example: `mcphost search "TODO|FIXME" . --regex`

### 4. **Result Limiting**
- `--max-results <number>`: Limit the number of results returned
- Default: 100 results
- Useful for large codebases

### 5. **File Type Filtering**
- `--file-types <extensions>`: Search only specific file types
- Example: `--file-types .go,.md,.txt`
- Default: Search all text files

### 6. **Directory Exclusion**
- `--exclude-dirs <directories>`: Exclude specific directories from search
- Default: Excludes `.git`, `node_modules`, `vendor`
- Example: `--exclude-dirs .git,node_modules,dist`

## Output Format

The search tool provides detailed output for each match:

```
📁 [file_path] (line [line_number])
   [full_line_content]
   Context: [context_information]
```

### Example Output

```
Found 3 results for 'function' in './cmd':

📁 cmd/search.go (line 45)
   func searchInFolder(ctx context.Context, searchWord, folder string, options *SearchOptions) error {
   Context: func searchInFolder(ctx context.Context, searchWord, folder string, options *SearchOptions) error {

📁 cmd/root.go (line 379)
   // Create spinner function for agent creation
   Context: // Create spinner function for agent creation
```

## Advanced Usage Examples

### 1. **Search for Error Handling Patterns**
```bash
mcphost search "if err != nil" ./internal --case-sensitive
```

### 2. **Find All TODO Comments**
```bash
mcphost search "TODO|FIXME|HACK" . --regex --file-types .go,.js,.py
```

### 3. **Search for Function Definitions**
```bash
mcphost search "func [A-Z]" . --regex --file-types .go
```

### 4. **Find Configuration References**
```bash
mcphost search "config" . --whole-word --exclude-dirs .git,node_modules,vendor
```

### 5. **Search in Specific File Types Only**
```bash
mcphost search "import" . --file-types .go,.js,.ts
```

## Binary File Detection

The search tool automatically detects and skips binary files to avoid errors and improve performance. It identifies binary files by:

1. **File Extension**: Common binary extensions (.exe, .dll, .so, .jpg, .png, etc.)
2. **Content Analysis**: Checks for null bytes and printable character ratio
3. **Performance**: Skips files that are likely to be binary

## Performance Tips

1. **Use specific directories**: Instead of searching the entire project, target specific folders
2. **Limit results**: Use `--max-results` to avoid overwhelming output
3. **Filter file types**: Use `--file-types` to search only relevant file types
4. **Exclude directories**: Use `--exclude-dirs` to skip irrelevant directories

## Error Handling

The search tool gracefully handles errors:
- Invalid directories: Shows clear error message
- Permission errors: Logs warning and continues
- Invalid regex: Shows syntax error and exits
- Binary files: Automatically skipped

## Integration with MCPHost

The search tool is fully integrated with the MCPHost CLI and follows the same patterns as other commands:
- Consistent flag naming
- Help integration (`mcphost search --help`)
- Error handling
- Output formatting

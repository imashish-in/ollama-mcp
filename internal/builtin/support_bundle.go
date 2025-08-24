package builtin

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SupportBundleSearchResult represents a search result from support bundle analysis
type SupportBundleSearchResult struct {
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	FullLine    string `json:"full_line"`
	MatchedText string `json:"matched_text"`
	Context     string `json:"context"`
	FileType    string `json:"file_type"`
	ArchivePath string `json:"archive_path,omitempty"`
}

// SupportBundleAnalysis represents the overall analysis results
type SupportBundleAnalysis struct {
	BundlePath     string                      `json:"bundle_path"`
	TotalFiles     int                         `json:"total_files"`
	ErrorLogs      []SupportBundleSearchResult `json:"error_logs"`
	WarningLogs    []SupportBundleSearchResult `json:"warning_logs"`
	ExceptionLogs  []SupportBundleSearchResult `json:"exception_logs"`
	SearchPatterns []string                    `json:"search_patterns"`
	AnalysisTime   time.Time                   `json:"analysis_time"`
	Duration       time.Duration               `json:"duration"`
}

// NewSupportBundleServer creates a new Support Bundle MCP server
func NewSupportBundleServer() (*server.MCPServer, error) {
	s := server.NewMCPServer("support-bundle-server", "1.0.0", server.WithToolCapabilities(true))

	// Register the support bundle analysis tool
	supportBundleTool := mcp.NewTool("support_bundle_analyze",
		mcp.WithDescription("Analyze Artifactory support bundles by searching for error logs, warnings, and exceptions in unzipped nested folders."),
		mcp.WithString("bundle_path",
			mcp.Description("Path to the support bundle folder (e.g., ./support-bundle, /path/to/support-bundle)"),
		),
		mcp.WithString("search_patterns",
			mcp.Description("Comma-separated list of search patterns (e.g., 'ERROR,WARNING,Exception,failed,error')"),
		),
		mcp.WithString("file_types",
			mcp.Description("Comma-separated list of file types to search (e.g., '.log,.txt,.out')"),
		),
		mcp.WithBoolean("case_sensitive",
			mcp.Description("Case sensitive search (default: false)"),
		),
		mcp.WithBoolean("include_archives",
			mcp.Description("Search inside nested zip/tar archives (default: true)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return per pattern (default: 100)"),
		),
		mcp.WithNumber("context_lines",
			mcp.Description("Number of context lines to include around matches (default: 2)"),
		),
		mcp.WithBoolean("extract_archives",
			mcp.Description("Extract archives to temporary directory for analysis (default: true)"),
		),
	)

	s.AddTool(supportBundleTool, executeSupportBundleAnalyze)
	return s, nil
}

// executeSupportBundleAnalyze handles the support bundle analysis tool execution
func executeSupportBundleAnalyze(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract parameters
	bundlePath := request.GetString("bundle_path", "./support-bundle")
	searchPatternsStr := request.GetString("search_patterns", "ERROR,WARNING,Exception,failed,error")
	fileTypesStr := request.GetString("file_types", ".log,.txt,.out")
	caseSensitive := request.GetBool("case_sensitive", false)
	includeArchives := request.GetBool("include_archives", true)
	maxResults := int(request.GetFloat("max_results", 100))
	contextLines := int(request.GetFloat("context_lines", 2))
	extractArchives := request.GetBool("extract_archives", true)

	// Validate bundle path
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("support bundle path does not exist: %s", bundlePath)), nil
	}

	// Parse search patterns
	searchPatterns := parseCommaSeparated(searchPatternsStr)
	if len(searchPatterns) == 0 {
		return mcp.NewToolResultError("at least one search pattern is required"), nil
	}

	// Parse file types
	fileTypes := parseCommaSeparated(fileTypesStr)
	if len(fileTypes) == 0 {
		fileTypes = []string{".log", ".txt", ".out"}
	}

	// Create analysis result
	analysis := &SupportBundleAnalysis{
		BundlePath:     bundlePath,
		SearchPatterns: searchPatterns,
		AnalysisTime:   startTime,
	}

	// Perform the analysis
	err := analyzeSupportBundle(ctx, bundlePath, searchPatterns, fileTypes, caseSensitive, includeArchives, extractArchives, maxResults, contextLines, analysis)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to analyze support bundle: %v", err)), nil
	}

	analysis.Duration = time.Since(startTime)

	// Convert to JSON
	resultJSON, err := json.Marshal(analysis)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// analyzeSupportBundle performs the main analysis
func analyzeSupportBundle(ctx context.Context, bundlePath string, searchPatterns, fileTypes []string, caseSensitive, includeArchives, extractArchives bool, maxResults, contextLines int, analysis *SupportBundleAnalysis) error {
	// Create temporary directory for extracted archives
	var tempDir string
	var err error
	if extractArchives {
		tempDir, err = os.MkdirTemp("", "support-bundle-analysis-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)
	}

	// First pass: Extract all compressed files recursively
	if extractArchives {
		err = extractAllCompressedFiles(ctx, bundlePath, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract compressed files: %v", err)
		}
	}

	// Second pass: Search through all files (including extracted ones)
	searchPaths := []string{bundlePath}
	if extractArchives {
		searchPaths = append(searchPaths, tempDir)
	}

	for _, searchPath := range searchPaths {
		err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if d.IsDir() {
				return nil
			}

			// Check if file type matches
			if !isMatchingFileType(path, fileTypes) {
				return nil
			}

			// Skip archive files in the second pass since they've been extracted
			if extractArchives && isArchiveFile(path) {
				return nil
			}

			// Process file
			archiveContext := ""
			if extractArchives && strings.HasPrefix(path, tempDir) {
				// Determine the original archive name from the extracted path
				relativePath, _ := filepath.Rel(tempDir, path)
				parts := strings.Split(relativePath, string(filepath.Separator))
				if len(parts) > 0 {
					archiveContext = parts[0]
				}
			}
			return processFile(ctx, path, archiveContext, searchPatterns, caseSensitive, maxResults, contextLines, analysis)
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// extractAllCompressedFiles recursively extracts all compressed files in the bundle
func extractAllCompressedFiles(ctx context.Context, bundlePath, tempDir string) error {
	extractedArchives := make(map[string]bool)

	return filepath.WalkDir(bundlePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			return nil
		}

		// Check if this is a compressed file
		if !isArchiveFile(path) {
			return nil
		}

		// Create a unique name for this archive
		archiveName := filepath.Base(path)
		archiveKey := fmt.Sprintf("%s_%s", archiveName, filepath.Dir(path))

		// Skip if already extracted
		if extractedArchives[archiveKey] {
			return nil
		}

		// Extract the archive
		extractPath := filepath.Join(tempDir, archiveName+"_extracted")
		err = extractArchive(path, extractPath)
		if err != nil {
			// Log error but continue with other files
			return nil
		}

		extractedArchives[archiveKey] = true

		// Recursively extract any nested archives
		return extractAllCompressedFiles(ctx, extractPath, tempDir)
	})
}

// processArchive handles archive files (zip, tar, etc.) - kept for backward compatibility
func processArchive(ctx context.Context, archivePath, tempDir string, searchPatterns, fileTypes []string, caseSensitive bool, maxResults, contextLines int, analysis *SupportBundleAnalysis) error {
	// Extract archive to temporary directory
	extractPath := filepath.Join(tempDir, filepath.Base(archivePath)+"_extracted")
	err := extractArchive(archivePath, extractPath)
	if err != nil {
		// Log error but continue with other files
		return nil
	}

	// Walk through extracted files
	return filepath.WalkDir(extractPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			return nil
		}

		// Check if file type matches
		if !isMatchingFileType(path, fileTypes) {
			return nil
		}

		// Process file with archive context
		relativePath, _ := filepath.Rel(extractPath, path)
		archiveContext := fmt.Sprintf("%s/%s", filepath.Base(archivePath), relativePath)
		return processFile(ctx, path, archiveContext, searchPatterns, caseSensitive, maxResults, contextLines, analysis)
	})
}

// processFile processes a single file for search patterns
func processFile(ctx context.Context, filePath, archiveContext string, searchPatterns []string, caseSensitive bool, maxResults, contextLines int, analysis *SupportBundleAnalysis) error {
	file, err := os.Open(filePath)
	if err != nil {
		return nil // Skip files we can't open
	}
	defer file.Close()

	// Check if file is binary
	if isBinaryFile(file) {
		return nil
	}

	analysis.TotalFiles++

	// Process each search pattern
	for _, pattern := range searchPatterns {
		// Check if we've reached max results for this pattern
		if len(analysis.ErrorLogs) >= maxResults && len(analysis.WarningLogs) >= maxResults && len(analysis.ExceptionLogs) >= maxResults {
			break
		}

		// Create regex pattern
		regexPattern := pattern
		if !caseSensitive {
			regexPattern = "(?i)" + regexp.QuoteMeta(pattern)
		}
		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		// Search in file
		results := searchInFile(file, regex, filePath, archiveContext, contextLines)

		// Categorize results
		for _, result := range results {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Categorize based on pattern
			switch {
			case strings.Contains(strings.ToUpper(pattern), "ERROR"):
				if len(analysis.ErrorLogs) < maxResults {
					analysis.ErrorLogs = append(analysis.ErrorLogs, result)
				}
			case strings.Contains(strings.ToUpper(pattern), "WARNING"):
				if len(analysis.WarningLogs) < maxResults {
					analysis.WarningLogs = append(analysis.WarningLogs, result)
				}
			case strings.Contains(strings.ToUpper(pattern), "EXCEPTION"):
				if len(analysis.ExceptionLogs) < maxResults {
					analysis.ExceptionLogs = append(analysis.ExceptionLogs, result)
				}
			default:
				// Add to error logs as default
				if len(analysis.ErrorLogs) < maxResults {
					analysis.ErrorLogs = append(analysis.ErrorLogs, result)
				}
			}
		}
	}

	return nil
}

// searchInFile searches for patterns in a file
func searchInFile(file *os.File, regex *regexp.Regexp, filePath, archiveContext string, contextLines int) []SupportBundleSearchResult {
	var results []SupportBundleSearchResult

	// Reset file pointer
	file.Seek(0, 0)

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var lines []string

	// Read all lines first for context
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Search through lines
	for i, line := range lines {
		lineNumber = i + 1
		matches := regex.FindAllStringIndex(line, -1)

		for _, match := range matches {
			start, end := match[0], match[1]
			matchedText := line[start:end]

			// Get context lines
			context := getContextLines(lines, i, contextLines)

			result := SupportBundleSearchResult{
				FilePath:    filePath,
				LineNumber:  lineNumber,
				FullLine:    strings.TrimSpace(line),
				MatchedText: matchedText,
				Context:     context,
				FileType:    filepath.Ext(filePath),
				ArchivePath: archiveContext,
			}

			results = append(results, result)
		}
	}

	return results
}

// getContextLines gets context lines around a match
func getContextLines(lines []string, lineIndex, contextLines int) string {
	start := max(0, lineIndex-contextLines)
	end := min(len(lines), lineIndex+contextLines+1)

	var contextLinesList []string
	for i := start; i < end; i++ {
		prefix := "  "
		if i == lineIndex {
			prefix = "> "
		}
		contextLinesList = append(contextLinesList, fmt.Sprintf("%s%d: %s", prefix, i+1, lines[i]))
	}

	return strings.Join(contextLinesList, "\n")
}

// extractArchive extracts various types of archives
func extractArchive(archivePath, extractPath string) error {
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".zip":
		return extractZipArchive(archivePath, extractPath)
	case ".tar":
		return extractTarArchive(archivePath, extractPath)
	case ".gz":
		return extractGzipArchive(archivePath, extractPath)
	case ".bz2":
		return extractBzip2Archive(archivePath, extractPath)
	case ".xz":
		return extractXzArchive(archivePath, extractPath)
	default:
		// Try zip as default
		return extractZipArchive(archivePath, extractPath)
	}
}

// extractZipArchive extracts a zip archive
func extractZipArchive(archivePath, extractPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(extractPath, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarArchive extracts a tar archive
func extractTarArchive(archivePath, extractPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file

	// Try to detect if it's gzipped
	var gzReader *gzip.Reader
	if strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz") || strings.HasSuffix(strings.ToLower(archivePath), ".tgz") {
		gzReader, err = gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		filePath := filepath.Join(extractPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// extractGzipArchive extracts a gzip archive
func extractGzipArchive(archivePath, extractPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	// Determine output filename
	baseName := filepath.Base(archivePath)
	if strings.HasSuffix(baseName, ".gz") {
		baseName = strings.TrimSuffix(baseName, ".gz")
	}
	outputPath := filepath.Join(extractPath, baseName)

	outFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, gzReader)
	return err
}

// extractBzip2Archive extracts a bzip2 archive
func extractBzip2Archive(archivePath, extractPath string) error {
	// For bzip2, we'll use the system command as Go doesn't have a built-in bzip2 package
	cmd := exec.Command("bunzip2", "-c", archivePath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("bunzip2 failed: %v", err)
	}

	// Determine output filename
	baseName := filepath.Base(archivePath)
	if strings.HasSuffix(baseName, ".bz2") {
		baseName = strings.TrimSuffix(baseName, ".bz2")
	}
	outputPath := filepath.Join(extractPath, baseName)

	return os.WriteFile(outputPath, output, 0644)
}

// extractXzArchive extracts an xz archive
func extractXzArchive(archivePath, extractPath string) error {
	// For xz, we'll use the system command as Go doesn't have a built-in xz package
	cmd := exec.Command("xz", "-d", "-c", archivePath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("xz failed: %v", err)
	}

	// Determine output filename
	baseName := filepath.Base(archivePath)
	if strings.HasSuffix(baseName, ".xz") {
		baseName = strings.TrimSuffix(baseName, ".xz")
	}
	outputPath := filepath.Join(extractPath, baseName)

	return os.WriteFile(outputPath, output, 0644)
}

// Helper functions
func parseCommaSeparated(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func isMatchingFileType(filePath string, fileTypes []string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, fileType := range fileTypes {
		if strings.HasPrefix(fileType, ".") {
			if ext == strings.ToLower(fileType) {
				return true
			}
		} else {
			if ext == "."+strings.ToLower(fileType) {
				return true
			}
		}
	}
	return false
}

func isArchiveFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	archiveExtensions := []string{".zip", ".tar", ".gz", ".bz2", ".xz", ".rar", ".7z"}
	for _, archiveExt := range archiveExtensions {
		if ext == archiveExt {
			return true
		}
	}
	return false
}

func isBinaryFile(file *os.File) bool {
	// Read first 512 bytes to check if file is binary
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return true // Assume binary if we can't read
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	// Check if mostly printable characters
	printableCount := 0
	for i := 0; i < n; i++ {
		if (buffer[i] >= 32 && buffer[i] <= 126) || buffer[i] == 9 || buffer[i] == 10 || buffer[i] == 13 {
			printableCount++
		}
	}

	// If less than 70% are printable, consider it binary
	return float64(printableCount)/float64(n) < 0.7
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

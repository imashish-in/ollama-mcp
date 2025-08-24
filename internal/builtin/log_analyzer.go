package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LogAnalysisResult represents a single log analysis result
type LogAnalysisResult struct {
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	FullLine    string `json:"full_line"`
	MatchedText string `json:"matched_text"`
	Severity    string `json:"severity"`
	Timestamp   string `json:"timestamp,omitempty"`
	Context     string `json:"context"`
}

// LogAnalysisSummary represents the overall analysis results
type LogAnalysisSummary struct {
	SourcePath     string              `json:"source_path"`
	TotalFiles     int                 `json:"total_files"`
	TotalErrors    int                 `json:"total_errors"`
	TotalWarnings  int                 `json:"total_warnings"`
	TotalInfo      int                 `json:"total_info"`
	ErrorLogs      []LogAnalysisResult `json:"error_logs"`
	WarningLogs    []LogAnalysisResult `json:"warning_logs"`
	InfoLogs       []LogAnalysisResult `json:"info_logs"`
	SearchPatterns []string            `json:"search_patterns"`
	AnalysisTime   time.Time           `json:"analysis_time"`
	Duration       string              `json:"duration"`
	FileStats      map[string]int      `json:"file_stats"`
	SeverityStats  map[string]int      `json:"severity_stats"`
}

// NewLogAnalyzerServer creates a new log analyzer MCP server
func NewLogAnalyzerServer() (*server.MCPServer, error) {
	s := server.NewMCPServer("log-analyzer-server", "1.0.0", server.WithToolCapabilities(true))

	// Register the log analysis tool
	logAnalyzerTool := mcp.NewTool("analyze_logs",
		mcp.WithDescription("Analyze log files to find errors, warnings, and other important patterns in extracted support bundle content"),
		mcp.WithString("source_path",
			mcp.Description("Path to the directory containing log files to analyze (e.g., ./support-bundle)"),
		),
		mcp.WithString("search_patterns",
			mcp.Description("Comma-separated list of search patterns (e.g., 'ERROR,WARNING,Exception,failed,error,warn')"),
		),
		mcp.WithString("file_types",
			mcp.Description("Comma-separated list of file types to search (e.g., '.log,.out,.err,.txt')"),
		),
		mcp.WithBoolean("case_sensitive",
			mcp.Description("Case sensitive search (default: false)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return per pattern (default: 100)"),
		),
		mcp.WithNumber("context_lines",
			mcp.Description("Number of context lines to include around matches (default: 2)"),
		),
		mcp.WithBoolean("include_timestamps",
			mcp.Description("Extract timestamps from log entries (default: true)"),
		),
		mcp.WithString("severity_levels",
			mcp.Description("Comma-separated severity levels to search (e.g., 'ERROR,WARNING,INFO,DEBUG')"),
		),
	)

	s.AddTool(logAnalyzerTool, executeLogAnalyzer)
	return s, nil
}

// executeLogAnalyzer handles the log analysis tool execution
func executeLogAnalyzer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract parameters
	sourcePath := request.GetString("source_path", "./support-bundle")
	searchPatternsStr := request.GetString("search_patterns", "ERROR,WARNING,Exception,failed,error,warn,CRITICAL,FATAL")
	fileTypesStr := request.GetString("file_types", ".log,.out,.err,.txt")
	caseSensitive := request.GetBool("case_sensitive", false)
	maxResults := int(request.GetFloat("max_results", 100))
	contextLines := int(request.GetFloat("context_lines", 2))
	includeTimestamps := request.GetBool("include_timestamps", true)
	severityLevelsStr := request.GetString("severity_levels", "ERROR,WARNING,INFO,DEBUG,CRITICAL,FATAL")

	// Validate source path
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("source path does not exist: %s", sourcePath)), nil
	}

	// Parse search patterns
	searchPatterns := strings.Split(searchPatternsStr, ",")
	for i, pattern := range searchPatterns {
		searchPatterns[i] = strings.TrimSpace(pattern)
	}

	// Parse file types
	fileTypes := strings.Split(fileTypesStr, ",")
	for i, fileType := range fileTypes {
		fileTypes[i] = strings.TrimSpace(fileType)
	}

	// Parse severity levels
	severityLevels := strings.Split(severityLevelsStr, ",")
	for i, level := range severityLevels {
		severityLevels[i] = strings.TrimSpace(level)
	}

	// Analyze logs
	summary, err := analyzeLogFiles(sourcePath, searchPatterns, fileTypes, caseSensitive, maxResults, contextLines, includeTimestamps, severityLevels)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to analyze logs: %v", err)), nil
	}

	summary.AnalysisTime = time.Now()
	summary.Duration = time.Since(startTime).String()
	summary.SourcePath = sourcePath
	summary.SearchPatterns = searchPatterns

	// Marshal result
	resultJSON, err := json.Marshal(summary)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// analyzeLogFiles performs the actual log analysis
func analyzeLogFiles(sourcePath string, searchPatterns []string, fileTypes []string, caseSensitive bool, maxResults, contextLines int, includeTimestamps bool, severityLevels []string) (*LogAnalysisSummary, error) {
	summary := &LogAnalysisSummary{
		ErrorLogs:     []LogAnalysisResult{},
		WarningLogs:   []LogAnalysisResult{},
		InfoLogs:      []LogAnalysisResult{},
		FileStats:     make(map[string]int),
		SeverityStats: make(map[string]int),
	}

	// Compile regex patterns
	var patterns []*regexp.Regexp
	for _, pattern := range searchPatterns {
		if !caseSensitive {
			pattern = "(?i)" + pattern
		}
		regex, err := regexp.Compile(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		patterns = append(patterns, regex)
	}

	// Compile severity patterns
	severityPatterns := make(map[string]*regexp.Regexp)
	for _, level := range severityLevels {
		pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(level))
		regex, err := regexp.Compile(pattern)
		if err == nil {
			severityPatterns[level] = regex
		}
	}

	// Walk through directory
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if info.IsDir() {
			return nil
		}

		// Check if file type matches
		ext := strings.ToLower(filepath.Ext(path))
		fileTypeMatch := false
		for _, fileType := range fileTypes {
			if ext == strings.ToLower(fileType) {
				fileTypeMatch = true
				break
			}
		}

		if !fileTypeMatch {
			return nil
		}

		summary.TotalFiles++
		fileResults := analyzeLogFile(path, patterns, severityPatterns, maxResults, contextLines, includeTimestamps)

		// Categorize results by severity
		for _, result := range fileResults {
			severity := strings.ToUpper(result.Severity)
			summary.SeverityStats[severity]++

			switch severity {
			case "ERROR", "CRITICAL", "FATAL":
				summary.ErrorLogs = append(summary.ErrorLogs, result)
				summary.TotalErrors++
			case "WARNING", "WARN":
				summary.WarningLogs = append(summary.WarningLogs, result)
				summary.TotalWarnings++
			case "INFO", "DEBUG":
				summary.InfoLogs = append(summary.InfoLogs, result)
				summary.TotalInfo++
			}
		}

		summary.FileStats[path] = len(fileResults)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}

	return summary, nil
}

// analyzeLogFile analyzes a single log file
func analyzeLogFile(filePath string, patterns []*regexp.Regexp, severityPatterns map[string]*regexp.Regexp, maxResults, contextLines int, includeTimestamps bool) []LogAnalysisResult {
	var results []LogAnalysisResult

	file, err := os.Open(filePath)
	if err != nil {
		return results
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string

	// Read all lines first for context
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Analyze each line
	for lineNumber, line := range lines {
		lineNumber++ // 1-based line numbers

		// Check for pattern matches
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				// Determine severity
				severity := "UNKNOWN"
				for level, severityPattern := range severityPatterns {
					if severityPattern.MatchString(line) {
						severity = level
						break
					}
				}

				// Extract timestamp if requested
				timestamp := ""
				if includeTimestamps {
					timestamp = extractTimestamp(line)
				}

				// Get context lines
				context := getLogContextLines(lines, lineNumber-1, contextLines)

				// Find the matched text
				matches := pattern.FindString(line)
				if matches == "" {
					matches = line
				}

				result := LogAnalysisResult{
					FilePath:    filePath,
					LineNumber:  lineNumber,
					FullLine:    line,
					MatchedText: matches,
					Severity:    severity,
					Timestamp:   timestamp,
					Context:     context,
				}

				results = append(results, result)

				if len(results) >= maxResults {
					return results
				}
			}
		}
	}

	return results
}

// extractTimestamp attempts to extract timestamp from log line
func extractTimestamp(line string) string {
	// Common timestamp patterns
	timestampPatterns := []string{
		`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,
		`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`,
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`,
		`\d{2}:\d{2}:\d{2}`,
	}

	for _, pattern := range timestampPatterns {
		regex := regexp.MustCompile(pattern)
		if match := regex.FindString(line); match != "" {
			return match
		}
	}

	return ""
}

// getLogContextLines gets context lines around a specific line
func getLogContextLines(lines []string, lineIndex, contextLines int) string {
	start := lineIndex - contextLines
	if start < 0 {
		start = 0
	}

	end := lineIndex + contextLines + 1
	if end > len(lines) {
		end = len(lines)
	}

	var contextLinesList []string
	for i := start; i < end; i++ {
		if i == lineIndex {
			contextLinesList = append(contextLinesList, ">>> "+lines[i]+" <<<")
		} else {
			contextLinesList = append(contextLinesList, lines[i])
		}
	}

	return strings.Join(contextLinesList, "\n")
}

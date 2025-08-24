package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

type SearchResult struct {
	FilePath    string
	LineNumber  int
	FullLine    string
	MatchedText string
	Context     string
}

type SearchOptions struct {
	CaseSensitive bool
	WholeWord     bool
	Regex         bool
	MaxResults    int
	FileTypes     []string
	ExcludeDirs   []string
}

var searchCmd = &cobra.Command{
	Use:   "search [word] [folder]",
	Short: "Search for words in specific folders across all file types",
	Long: `Search for words in specific folders across all file types and return relevant results 
with complete sentences and file names.

Examples:
  mcphost search "function" ./internal
  mcphost search "error" ./cmd --case-sensitive
  mcphost search "test" . --whole-word --max-results 50
  mcphost search "TODO" . --regex --exclude-dirs .git,node_modules`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		searchWord := args[0]
		searchFolder := args[1]

		// Parse flags
		caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
		wholeWord, _ := cmd.Flags().GetBool("whole-word")
		regex, _ := cmd.Flags().GetBool("regex")
		maxResults, _ := cmd.Flags().GetInt("max-results")
		fileTypes, _ := cmd.Flags().GetStringSlice("file-types")
		excludeDirs, _ := cmd.Flags().GetStringSlice("exclude-dirs")

		options := &SearchOptions{
			CaseSensitive: caseSensitive,
			WholeWord:     wholeWord,
			Regex:         regex,
			MaxResults:    maxResults,
			FileTypes:     fileTypes,
			ExcludeDirs:   excludeDirs,
		}

		return searchInFolder(context.Background(), searchWord, searchFolder, options)
	},
}

func init() {
	searchCmd.Flags().Bool("case-sensitive", false, "Case sensitive search")
	searchCmd.Flags().Bool("whole-word", false, "Match whole words only")
	searchCmd.Flags().Bool("regex", false, "Treat search term as regular expression")
	searchCmd.Flags().Int("max-results", 100, "Maximum number of results to return")
	searchCmd.Flags().StringSlice("file-types", []string{}, "Specific file types to search (e.g., .go,.md,.txt)")
	searchCmd.Flags().StringSlice("exclude-dirs", []string{".git", "node_modules", "vendor"}, "Directories to exclude from search")
}

func searchInFolder(ctx context.Context, searchWord, folder string, options *SearchOptions) error {
	// Validate folder exists
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %s", folder)
	}

	// Convert exclude dirs to map for faster lookup
	excludeMap := make(map[string]bool)
	for _, dir := range options.ExcludeDirs {
		excludeMap[dir] = true
	}

	// Convert file types to map for faster lookup
	fileTypeMap := make(map[string]bool)
	for _, ext := range options.FileTypes {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		fileTypeMap[ext] = true
	}

	// Build search pattern
	var pattern *regexp.Regexp
	var err error

	if options.Regex {
		if !options.CaseSensitive {
			pattern, err = regexp.Compile("(?i)" + searchWord)
		} else {
			pattern, err = regexp.Compile(searchWord)
		}
		if err != nil {
			return fmt.Errorf("invalid regular expression: %v", err)
		}
	} else {
		searchPattern := regexp.QuoteMeta(searchWord)
		if options.WholeWord {
			searchPattern = "\\b" + searchPattern + "\\b"
		}
		if !options.CaseSensitive {
			searchPattern = "(?i)" + searchPattern
		}
		pattern, err = regexp.Compile(searchPattern)
		if err != nil {
			return fmt.Errorf("error compiling search pattern: %v", err)
		}
	}

	results := []SearchResult{}
	resultCount := 0

	// Walk through the directory
	err = filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if we should skip this directory
		if d.IsDir() {
			dirName := filepath.Base(path)
			if excludeMap[dirName] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we should skip this file based on file type
		if len(options.FileTypes) > 0 {
			ext := filepath.Ext(path)
			if !fileTypeMap[ext] {
				return nil
			}
		}

		// Skip binary files and other non-text files
		if isBinaryFile(path) {
			return nil
		}

		// Search in the file
		fileResults, err := searchInFile(path, pattern, options)
		if err != nil {
			// Log error but continue with other files
			fmt.Fprintf(os.Stderr, "Error searching in %s: %v\n", path, err)
			return nil
		}

		results = append(results, fileResults...)
		resultCount += len(fileResults)

		// Check if we've reached the maximum results
		if options.MaxResults > 0 && resultCount >= options.MaxResults {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %v", err)
	}

	// Display results
	if len(results) == 0 {
		fmt.Printf("No results found for '%s' in '%s'\n", searchWord, folder)
		return nil
	}

	fmt.Printf("Found %d results for '%s' in '%s':\n\n", len(results), searchWord, folder)

	for i, result := range results {
		if options.MaxResults > 0 && i >= options.MaxResults {
			break
		}

		fmt.Printf("📁 %s (line %d)\n", result.FilePath, result.LineNumber)
		fmt.Printf("   %s\n", result.FullLine)
		if result.Context != "" {
			fmt.Printf("   Context: %s\n", result.Context)
		}
		fmt.Println()
	}

	return nil
}

func searchInFile(filePath string, pattern *regexp.Regexp, options *SearchOptions) ([]SearchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []SearchResult
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Find all matches in the line
		matches := pattern.FindAllStringIndex(line, -1)
		for _, match := range matches {
			start, end := match[0], match[1]
			matchedText := line[start:end]

			// Get context (previous and next lines if available)
			context := getContext(filePath, lineNumber, line)

			result := SearchResult{
				FilePath:    filePath,
				LineNumber:  lineNumber,
				FullLine:    strings.TrimSpace(line),
				MatchedText: matchedText,
				Context:     context,
			}
			results = append(results, result)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func getContext(filePath string, lineNumber int, currentLine string) string {
	// For now, return a simple context. In a more advanced version,
	// we could read previous and next lines to provide more context.
	if len(currentLine) > 100 {
		return currentLine[:100] + "..."
	}
	return currentLine
}

func isBinaryFile(filePath string) bool {
	// Check file extension for common binary files
	binaryExtensions := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".obj": true, ".o": true, ".a": true,
		".class": true, ".jar": true, ".war": true,
		".pyc": true, ".pyo": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".tiff": true, ".ico": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".wav": true, ".flac": true, ".ogg": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".rar": true, ".7z": true,
		".db": true, ".sqlite": true, ".sqlite3": true,
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if binaryExtensions[ext] {
		return true
	}

	// Try to read first few bytes to detect binary files
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	// Check if the file contains null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	// Check if the file contains mostly printable characters
	printableCount := 0
	for i := 0; i < n; i++ {
		if unicode.IsPrint(rune(buffer[i])) || unicode.IsSpace(rune(buffer[i])) {
			printableCount++
		}
	}

	// If less than 70% of characters are printable, consider it binary
	return float64(printableCount)/float64(n) < 0.7
}

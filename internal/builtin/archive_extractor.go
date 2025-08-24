package builtin

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ArchiveExtractionResult represents the result of archive extraction
type ArchiveExtractionResult struct {
	SourcePath     string   `json:"source_path"`
	ExtractedFiles []string `json:"extracted_files"`
	TotalFiles     int      `json:"total_files"`
	Errors         []string `json:"errors,omitempty"`
	Message        string   `json:"message"`
	Duration       string   `json:"duration"`
}

// NewArchiveExtractorServer creates a new archive extractor MCP server
func NewArchiveExtractorServer() (*server.MCPServer, error) {
	s := server.NewMCPServer("archive-extractor-server", "1.0.0", server.WithToolCapabilities(true))

	// Register the archive extraction tool
	archiveExtractorTool := mcp.NewTool("extract_archives",
		mcp.WithDescription("Extract all compressed files recursively from a directory, handling nested archives of various formats (ZIP, TAR, GZ, BZ2, XZ). Performs deep recursive extraction until no more archives are found."),
		mcp.WithString("source_path",
			mcp.Description("Path to the directory containing compressed files to extract"),
		),
		mcp.WithString("output_dir",
			mcp.Description("Output directory for extracted files (optional, defaults to source directory)"),
		),
		mcp.WithBoolean("recursive",
			mcp.Description("Whether to recursively extract nested archives (default: true). When enabled, continues extracting until no more archives are found."),
		),
	)

	s.AddTool(archiveExtractorTool, executeArchiveExtractor)
	return s, nil
}

// executeArchiveExtractor handles the archive extraction tool execution
func executeArchiveExtractor(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract parameters
	sourcePath := request.GetString("source_path", "")
	outputDir := request.GetString("output_dir", "")
	recursive := request.GetBool("recursive", true)

	if sourcePath == "" {
		return mcp.NewToolResultError("source_path is required"), nil
	}

	if outputDir == "" {
		outputDir = sourcePath
	}

	// Validate source path
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("source path does not exist: %s", sourcePath)), nil
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create output directory: %v", err)), nil
	}

	// Extract all archives recursively
	extractedFiles, errors := extractAllArchivesRecursively(sourcePath, outputDir, recursive)
	result := &ArchiveExtractionResult{
		SourcePath:     sourcePath,
		ExtractedFiles: extractedFiles,
		Errors:         errors,
		TotalFiles:     len(extractedFiles),
		Duration:       time.Since(startTime).String(),
	}

	if len(errors) > 0 {
		result.Message = fmt.Sprintf("Extraction completed with %d files extracted and %d errors", len(extractedFiles), len(errors))
	} else {
		result.Message = fmt.Sprintf("Successfully extracted %d files from archives", len(extractedFiles))
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// extractAllArchivesRecursively extracts all archives in a directory recursively
func extractAllArchivesRecursively(sourcePath, outputDir string, recursive bool) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	// First pass: extract all archives in the source directory
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error accessing %s: %v", path, err))
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check if file is a compressed archive
		if isCompressedFile(path) {
			relativePath, _ := filepath.Rel(sourcePath, path)
			targetDir := filepath.Join(outputDir, filepath.Dir(relativePath))

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create target directory %s: %v", targetDir, err))
				return nil
			}

			files, extractErrors := extractArchiveFile(path, targetDir)
			extractedFiles = append(extractedFiles, files...)
			errors = append(errors, extractErrors...)
		}

		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Sprintf("Error walking directory: %v", err))
	}

	// If recursive extraction is enabled, continue extracting nested archives
	if recursive {
		nestedFiles, nestedErrors := extractNestedArchivesRecursively(outputDir)
		extractedFiles = append(extractedFiles, nestedFiles...)
		errors = append(errors, nestedErrors...)
	}

	return extractedFiles, errors
}

// extractNestedArchivesRecursively recursively extracts nested archives until no more are found
func extractNestedArchivesRecursively(rootDir string) ([]string, []string) {
	var allExtractedFiles []string
	var allErrors []string
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++
		var extractedFiles []string
		var errors []string
		archivesFound := false

		// Walk through the entire directory tree
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error accessing %s: %v", path, err))
				return nil
			}

			if info.IsDir() {
				return nil
			}

			// Check if file is a compressed archive
			if isCompressedFile(path) {
				archivesFound = true
				targetDir := filepath.Dir(path)

				files, extractErrors := extractArchiveFile(path, targetDir)
				extractedFiles = append(extractedFiles, files...)
				errors = append(errors, extractErrors...)

				// Remove the original archive file after extraction
				if err := os.Remove(path); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to remove original archive %s: %v", path, err))
				}
			}

			return nil
		})

		if err != nil {
			errors = append(errors, fmt.Sprintf("Error walking directory: %v", err))
		}

		allExtractedFiles = append(allExtractedFiles, extractedFiles...)
		allErrors = append(allErrors, errors...)

		// If no archives were found in this iteration, we're done
		if !archivesFound {
			break
		}
	}

	if iteration >= maxIterations {
		allErrors = append(allErrors, fmt.Sprintf("Reached maximum iterations (%d) for recursive extraction", maxIterations))
	}

	return allExtractedFiles, allErrors
}

// extractNestedArchives recursively extracts nested archives (deprecated - use extractNestedArchivesRecursively)
func extractNestedArchives(sourceDir, outputDir string) ([]string, []string) {
	// This function is kept for backward compatibility but now delegates to the improved version
	return extractNestedArchivesRecursively(sourceDir)
}

// isCompressedFile checks if a file is a compressed archive
func isCompressedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".zip", ".tar", ".gz", ".tgz", ".bz2", ".tbz2", ".xz", ".txz":
		return true
	}

	// Check for double extensions
	base := strings.TrimSuffix(filePath, ext)
	doubleExt := strings.ToLower(filepath.Ext(base))
	switch doubleExt + ext {
	case ".tar.gz", ".tar.bz2", ".tar.xz":
		return true
	}

	return false
}

// extractArchiveFile extracts a single archive file
func extractArchiveFile(archivePath, targetDir string) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	ext := strings.ToLower(filepath.Ext(archivePath))
	base := strings.TrimSuffix(archivePath, ext)
	doubleExt := strings.ToLower(filepath.Ext(base))

	switch {
	case ext == ".zip":
		files, errs := extractZipArchiveFile(archivePath, targetDir)
		extractedFiles = append(extractedFiles, files...)
		errors = append(errors, errs...)
	case ext == ".tar" || doubleExt+ext == ".tar.gz" || doubleExt+ext == ".tar.bz2" || doubleExt+ext == ".tar.xz":
		files, errs := extractTarArchiveFile(archivePath, targetDir)
		extractedFiles = append(extractedFiles, files...)
		errors = append(errors, errs...)
	case ext == ".gz" && doubleExt != ".tar":
		files, errs := extractGzipArchiveFile(archivePath, targetDir)
		extractedFiles = append(extractedFiles, files...)
		errors = append(errors, errs...)
	case ext == ".bz2" && doubleExt != ".tar":
		files, errs := extractBzip2ArchiveFile(archivePath, targetDir)
		extractedFiles = append(extractedFiles, files...)
		errors = append(errors, errs...)
	case ext == ".xz" && doubleExt != ".tar":
		files, errs := extractXzArchiveFile(archivePath, targetDir)
		extractedFiles = append(extractedFiles, files...)
		errors = append(errors, errs...)
	default:
		errors = append(errors, fmt.Sprintf("Unsupported archive format: %s", archivePath))
	}

	return extractedFiles, errors
}

// extractZipArchiveFile extracts a ZIP archive
func extractZipArchiveFile(archivePath, targetDir string) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to open ZIP archive %s: %v", archivePath, err))
		return extractedFiles, errors
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(targetDir, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, 0755); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create directory %s: %v", filePath, err))
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to create directory for %s: %v", filePath, err))
			continue
		}

		rc, err := file.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to open file %s in archive: %v", file.Name, err))
			continue
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			rc.Close()
			errors = append(errors, fmt.Sprintf("Failed to create file %s: %v", filePath, err))
			continue
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to extract file %s: %v", file.Name, err))
		} else {
			extractedFiles = append(extractedFiles, filePath)
		}
	}

	return extractedFiles, errors
}

// extractTarArchiveFile extracts a TAR archive (including compressed variants)
func extractTarArchiveFile(archivePath, targetDir string) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	file, err := os.Open(archivePath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to open TAR archive %s: %v", archivePath, err))
		return extractedFiles, errors
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle compression
	ext := strings.ToLower(filepath.Ext(archivePath))
	base := strings.TrimSuffix(archivePath, ext)
	doubleExt := strings.ToLower(filepath.Ext(base))

	switch doubleExt + ext {
	case ".tar.gz", ".tgz":
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to create gzip reader for %s: %v", archivePath, err))
			return extractedFiles, errors
		}
		defer gzReader.Close()
		reader = gzReader
	case ".tar.bz2", ".tbz2":
		bz2Reader := bzip2.NewReader(file)
		reader = bz2Reader
	case ".tar.xz", ".txz":
		// Use xz command for XZ compression
		cmd := exec.Command("xz", "-d", "-c", archivePath)
		cmd.Stdin = file
		output, err := cmd.Output()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to decompress XZ archive %s: %v", archivePath, err))
			return extractedFiles, errors
		}
		reader = strings.NewReader(string(output))
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to read TAR header: %v", err))
			break
		}

		filePath := filepath.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, 0755); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create directory %s: %v", filePath, err))
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create directory for %s: %v", filePath, err))
				continue
			}

			outFile, err := os.Create(filePath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create file %s: %v", filePath, err))
				continue
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()

			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to extract file %s: %v", header.Name, err))
			} else {
				extractedFiles = append(extractedFiles, filePath)
			}
		}
	}

	return extractedFiles, errors
}

// extractGzipArchiveFile extracts a GZIP archive
func extractGzipArchiveFile(archivePath, targetDir string) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	file, err := os.Open(archivePath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to open GZIP archive %s: %v", archivePath, err))
		return extractedFiles, errors
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to create gzip reader for %s: %v", archivePath, err))
		return extractedFiles, errors
	}
	defer gzReader.Close()

	// Determine output filename
	baseName := filepath.Base(archivePath)
	outputName := strings.TrimSuffix(baseName, ".gz")
	outputPath := filepath.Join(targetDir, outputName)

	outFile, err := os.Create(outputPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to create output file %s: %v", outputPath, err))
		return extractedFiles, errors
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, gzReader)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to extract GZIP archive %s: %v", archivePath, err))
	} else {
		extractedFiles = append(extractedFiles, outputPath)
	}

	return extractedFiles, errors
}

// extractBzip2ArchiveFile extracts a BZIP2 archive
func extractBzip2ArchiveFile(archivePath, targetDir string) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	file, err := os.Open(archivePath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to open BZIP2 archive %s: %v", archivePath, err))
		return extractedFiles, errors
	}
	defer file.Close()

	bz2Reader := bzip2.NewReader(file)

	// Determine output filename
	baseName := filepath.Base(archivePath)
	outputName := strings.TrimSuffix(baseName, ".bz2")
	outputPath := filepath.Join(targetDir, outputName)

	outFile, err := os.Create(outputPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to create output file %s: %v", outputPath, err))
		return extractedFiles, errors
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, bz2Reader)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to extract BZIP2 archive %s: %v", archivePath, err))
	} else {
		extractedFiles = append(extractedFiles, outputPath)
	}

	return extractedFiles, errors
}

// extractXzArchiveFile extracts an XZ archive
func extractXzArchiveFile(archivePath, targetDir string) ([]string, []string) {
	var extractedFiles []string
	var errors []string

	// Use xz command for XZ compression
	cmd := exec.Command("xz", "-d", "-c", archivePath)
	output, err := cmd.Output()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to decompress XZ archive %s: %v", archivePath, err))
		return extractedFiles, errors
	}

	// Determine output filename
	baseName := filepath.Base(archivePath)
	outputName := strings.TrimSuffix(baseName, ".xz")
	outputPath := filepath.Join(targetDir, outputName)

	outFile, err := os.Create(outputPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to create output file %s: %v", outputPath, err))
		return extractedFiles, errors
	}
	defer outFile.Close()

	_, err = outFile.Write(output)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to write extracted XZ archive %s: %v", archivePath, err))
	} else {
		extractedFiles = append(extractedFiles, outputPath)
	}

	return extractedFiles, errors
}

package cmd

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// createNestedTestArchives creates a test structure with nested zip files
func createNestedTestArchives() error {
	testDir := "test_nested_extraction"

	// Clean up existing test directory
	if err := os.RemoveAll(testDir); err != nil {
		return fmt.Errorf("failed to remove existing test directory: %v", err)
	}

	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("failed to create test directory: %v", err)
	}

	fmt.Println("📁 Creating test directory structure...")

	// Create level 1 content
	if err := os.WriteFile(filepath.Join(testDir, "level1.txt"), []byte("Level 1 content"), 0644); err != nil {
		return fmt.Errorf("failed to create level1.txt: %v", err)
	}

	if err := os.WriteFile(filepath.Join(testDir, "data1.txt"), []byte("Level 1 data"), 0644); err != nil {
		return fmt.Errorf("failed to create data1.txt: %v", err)
	}

	// Create level 2 directory and content
	level2Dir := filepath.Join(testDir, "level2")
	if err := os.MkdirAll(level2Dir, 0755); err != nil {
		return fmt.Errorf("failed to create level2 directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(level2Dir, "level2.txt"), []byte("Level 2 content"), 0644); err != nil {
		return fmt.Errorf("failed to create level2.txt: %v", err)
	}

	if err := os.WriteFile(filepath.Join(level2Dir, "data2.txt"), []byte("Level 2 data"), 0644); err != nil {
		return fmt.Errorf("failed to create data2.txt: %v", err)
	}

	// Create level 3 directory and content
	level3Dir := filepath.Join(level2Dir, "level3")
	if err := os.MkdirAll(level3Dir, 0755); err != nil {
		return fmt.Errorf("failed to create level3 directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(level3Dir, "level3.txt"), []byte("Level 3 content"), 0644); err != nil {
		return fmt.Errorf("failed to create level3.txt: %v", err)
	}

	if err := os.WriteFile(filepath.Join(level3Dir, "data3.txt"), []byte("Level 3 data"), 0644); err != nil {
		return fmt.Errorf("failed to create data3.txt: %v", err)
	}

	fmt.Println("📦 Creating nested zip files...")

	// Create level 3 zip
	if err := createZipArchive(filepath.Join(level3Dir, "level3.zip"), level3Dir, []string{"level3.txt", "data3.txt"}); err != nil {
		return fmt.Errorf("failed to create level3.zip: %v", err)
	}

	// Create level 2 zip (contains level3.zip)
	if err := createZipArchive(filepath.Join(level2Dir, "level2.zip"), level2Dir, []string{"level2.txt", "data2.txt", "level3/level3.zip"}); err != nil {
		return fmt.Errorf("failed to create level2.zip: %v", err)
	}

	// Create level 1 zip (contains level2.zip)
	if err := createZipArchive(filepath.Join(testDir, "level1.zip"), testDir, []string{"level1.txt", "data1.txt", "level2/level2.zip"}); err != nil {
		return fmt.Errorf("failed to create level1.zip: %v", err)
	}

	// Clean up original files (keep only the zip files)
	if err := os.RemoveAll(level2Dir); err != nil {
		return fmt.Errorf("failed to remove level2 directory: %v", err)
	}

	if err := os.Remove(filepath.Join(testDir, "level1.txt")); err != nil {
		return fmt.Errorf("failed to remove level1.txt: %v", err)
	}

	if err := os.Remove(filepath.Join(testDir, "data1.txt")); err != nil {
		return fmt.Errorf("failed to remove data1.txt: %v", err)
	}

	fmt.Println("📋 Test directory structure created:")
	fmt.Println("   test_nested_extraction/")
	fmt.Println("   ├── level1.zip (contains level2.zip)")
	fmt.Println("   └── level2.zip (contains level3.zip)")
	fmt.Println("       └── level3.zip (contains text files)")

	return nil
}

// createZipArchive creates a zip file with the specified files
func createZipArchive(zipPath, baseDir string, files []string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file %s: %v", zipPath, err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		filePath := filepath.Join(baseDir, file)

		// Check if it's a directory
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %v", filePath, err)
		}

		if info.IsDir() {
			// Add directory entry
			header := &zip.FileHeader{
				Name:     file + "/",
				Method:   zip.Deflate,
				Modified: info.ModTime(),
			}
			_, err = zipWriter.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("failed to create directory entry %s: %v", file, err)
			}
		} else {
			// Add file entry
			header := &zip.FileHeader{
				Name:     file,
				Method:   zip.Deflate,
				Modified: info.ModTime(),
			}
			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("failed to create file entry %s: %v", file, err)
			}

			// Read and write file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", filePath, err)
			}

			if _, err := writer.Write(content); err != nil {
				return fmt.Errorf("failed to write file content %s: %v", file, err)
			}
		}
	}

	return nil
}

// listExtractedFiles lists all .txt files in the test directory
func listExtractedFiles() error {
	testDir := "test_nested_extraction"

	fmt.Println("📁 Final directory structure:")

	err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".txt") {
			relativePath, _ := filepath.Rel(testDir, path)
			fmt.Printf("   %s\n", relativePath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %v", err)
	}

	return nil
}

func main() {
	fmt.Println("🧪 Testing Nested Archive Extraction (Go Implementation)")
	fmt.Println()

	// Create test archives
	if err := createNestedTestArchives(); err != nil {
		fmt.Printf("❌ Error creating test archives: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("🔧 Test archives created successfully!")
	fmt.Println("   You can now test the extraction using:")
	fmt.Println("   ./mcphost --config=local.json -m ollama:qwen3:8b -p \"Extract all archives recursively from the test_nested_extraction directory\"")
	fmt.Println()
	fmt.Println("✅ Test setup completed!")
}

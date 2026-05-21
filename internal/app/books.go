package app

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type VolumeInfo struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	IsZip             bool   `json:"isZip"`
	IsDir             bool   `json:"isDir"`
	Images            int    `json:"images"`
	HasSubdirectories bool   `json:"hasSubdirectories"` // True if zip contains subdirectories or nested zips
}

func (a *App) GetSeriesList() ([]string, error) {
	a.mu.Lock()
	rootPath := a.rootPath
	a.mu.Unlock()

	fmt.Printf("GetSeriesList called, rootPath: %s\n", rootPath)

	if rootPath == "" {
		fmt.Println("GetSeriesList: rootPath is empty")
		return []string{}, nil
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var series []string
	for _, entry := range entries {
		if entry.IsDir() {
			series = append(series, entry.Name())
		}
	}

	sort.Strings(series)
	fmt.Printf("GetSeriesList: found %d series\n", len(series))
	return series, nil
}

func (a *App) GetVolumeList(series string) ([]VolumeInfo, error) {
	a.mu.Lock()
	rootPath := a.rootPath
	a.mu.Unlock()

	if rootPath == "" || series == "" {
		return []VolumeInfo{}, nil
	}

	seriesPath := filepath.Join(rootPath, series)
	entries, err := os.ReadDir(seriesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read series directory: %w", err)
	}

	var volumes []VolumeInfo
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(seriesPath, name)

		if entry.IsDir() {
			imageCount, err := countImagesInDir(path)
			if err == nil && imageCount > 0 {
				volumes = append(volumes, VolumeInfo{
					Name:   name,
					Path:   path,
					IsZip:  false,
					IsDir:  true,
					Images: imageCount,
				})
			}
		} else {
			if isArchiveFile(name) {
				imageCount, hasSubdirs, err := countImagesInArchiveWithSubdirs(path)
				if err == nil && imageCount > 0 {
					volumes = append(volumes, VolumeInfo{
						Name:              name,
						Path:              path,
						IsZip:             true, // Keep as true for compatibility
						IsDir:             false,
						Images:            imageCount,
						HasSubdirectories: hasSubdirs,
					})
				}
			}
		}
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	return volumes, nil
}

func countImagesInDir(dirPath string) (int, error) {
	count := 0
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isImageFile(path) {
			count++
		}
		return nil
	})
	return count, err
}

func countImagesInZip(zipPath string) (int, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	count := 0
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && isImageFile(f.Name) {
			count++
		}
	}
	return count, nil
}

func countImagesInZipWithSubdirs(zipPath string) (int, bool, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, false, err
	}
	defer r.Close()

	count := 0
	hasSubdirs := false
	hasNestedZips := false

	for _, f := range r.File {
		name := f.Name

		// Check for subdirectories
		dir := filepath.Dir(name)
		if dir != "." && dir != "/" {
			parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
			if len(parts) > 1 {
				hasSubdirs = true
			}
		}

		// Check for nested archives
		if !f.FileInfo().IsDir() && isArchiveFile(name) {
			hasNestedZips = true
		}

		// Count images
		if !f.FileInfo().IsDir() && isImageFile(name) {
			count++
		}
	}

	return count, hasSubdirs || hasNestedZips, nil
}

func countImagesInArchiveWithSubdirs(archivePath string) (int, bool, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))

	// For now, only ZIP is fully supported
	// RAR and 7z will be treated as simple archives without subdirectory detection
	if ext == ".zip" || ext == ".cbz" {
		return countImagesInZipWithSubdirs(archivePath)
	}

	// For RAR and 7z, just count images (no subdirectory support yet)
	// Return hasSubdirs=false to open directly in viewer
	return countImagesInArchive(archivePath), false, nil
}

func countImagesInArchive(archivePath string) int {
	ext := strings.ToLower(filepath.Ext(archivePath))

	if ext == ".zip" || ext == ".cbz" {
		count, _, _ := countImagesInZipWithSubdirs(archivePath)
		return count
	}

	// For RAR and 7z, return 1 as placeholder
	// Actual counting will be done when opening
	return 1
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp":
		return true
	}
	return false
}

func isZipFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".zip" || ext == ".cbz"
}

func isArchiveFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".zip" || ext == ".cbz" || ext == ".rar" || ext == ".cbr" || ext == ".7z" || ext == ".cb7"
}

// GetZipContents returns the contents of a zip file (nested zips, directories, or images)
func (a *App) GetZipContents(zipPath string) ([]VolumeInfo, error) {
	fmt.Printf("GetZipContents called for: %s\n", zipPath)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// First pass: collect directories and files
	dirMap := make(map[string]int) // directory -> image count
	var nestedZips []VolumeInfo
	rootImages := 0

	fmt.Printf("Scanning zip contents...\n")
	for _, f := range r.File {
		name := f.Name
		fmt.Printf("  File: %s (IsDir: %v)\n", name, f.FileInfo().IsDir())

		// Check if it's in a subdirectory
		dir := filepath.Dir(name)
		if dir != "." && dir != "/" {
			// Extract top-level directory
			parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
			if len(parts) > 1 {
				topDir := parts[0]
				if !f.FileInfo().IsDir() && isImageFile(name) {
					dirMap[topDir]++
					fmt.Printf("    -> Added to directory '%s' (count: %d)\n", topDir, dirMap[topDir])
				}
				continue
			}
		}

		// Root level files
		if !f.FileInfo().IsDir() {
			if isZipFile(name) {
				// Nested zip file
				fmt.Printf("    -> Nested zip detected\n")
				nestedZips = append(nestedZips, VolumeInfo{
					Name:   filepath.Base(name),
					Path:   zipPath + "::" + name,
					IsZip:  true,
					IsDir:  false,
					Images: 0,
				})
			} else if isImageFile(name) {
				rootImages++
				fmt.Printf("    -> Root level image (count: %d)\n", rootImages)
			}
		}
	}

	var volumes []VolumeInfo

	fmt.Printf("Analysis results:\n")
	fmt.Printf("  Directories found: %d\n", len(dirMap))
	fmt.Printf("  Nested zips found: %d\n", len(nestedZips))
	fmt.Printf("  Root images: %d\n", rootImages)

	// If there are directories with images, treat each as a volume
	if len(dirMap) > 0 {
		fmt.Printf("Creating volumes from directories:\n")
		for dirName, imageCount := range dirMap {
			if imageCount > 0 {
				fmt.Printf("  - %s (%d images)\n", dirName, imageCount)
				volumes = append(volumes, VolumeInfo{
					Name:   dirName,
					Path:   zipPath + "::" + dirName,
					IsZip:  false,
					IsDir:  true,
					Images: imageCount,
				})
			}
		}
	}

	// Add nested zips
	if len(nestedZips) > 0 {
		fmt.Printf("Adding nested zips:\n")
		for _, nz := range nestedZips {
			fmt.Printf("  - %s\n", nz.Name)
		}
	}
	volumes = append(volumes, nestedZips...)

	// If no directories or nested zips, treat the whole zip as a volume
	if len(volumes) == 0 && rootImages > 0 {
		fmt.Printf("Treating entire zip as single volume (%d images)\n", rootImages)
		volumes = append(volumes, VolumeInfo{
			Name:   filepath.Base(zipPath),
			Path:   zipPath,
			IsZip:  true,
			IsDir:  false,
			Images: rootImages,
		})
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	fmt.Printf("Returning %d volumes\n", len(volumes))
	return volumes, nil
}

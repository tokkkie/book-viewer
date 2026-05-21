package app

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
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

	if rootPath == "" {
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
	var directImages []string

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(seriesPath, name)

		if entry.IsDir() {
			imageCount, hasSubdirs, err := checkDirContents(path)
			if err == nil && imageCount > 0 {
				volumes = append(volumes, VolumeInfo{
					Name:              name,
					Path:              path,
					IsZip:             false,
					IsDir:             true,
					Images:            imageCount,
					HasSubdirectories: hasSubdirs,
				})
			}
		} else if isArchiveFile(name) {
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
		} else if isImageFile(name) {
			directImages = append(directImages, name)
		}
	}

	// If there are images directly in the series directory, add them as a single volume
	if len(directImages) > 0 {
		volumes = append(volumes, VolumeInfo{
			Name:   series + " (images)",
			Path:   seriesPath,
			IsZip:  false,
			IsDir:  true,
			Images: len(directImages),
		})
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	return volumes, nil
}

func checkDirContents(dirPath string) (imageCount int, hasSubdirs bool, err error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, false, err
	}

	hasSubdirs = false
	hasImages := false
	hasArchives := false

	// Check immediate children
	for _, entry := range entries {
		if entry.IsDir() {
			hasSubdirs = true
		} else if isImageFile(entry.Name()) {
			hasImages = true
		} else if isArchiveFile(entry.Name()) {
			hasArchives = true
		}
	}

	// If has subdirectories, count all images and archives recursively
	if hasSubdirs {
		count := 0
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (isImageFile(path) || isArchiveFile(path)) {
				count++
			}
			return nil
		})
		return count, true, err
	}

	// If only images/archives, count them
	// Set hasSubdirs=true if archives exist so GetDirContents can be used to view them
	if hasImages || hasArchives {
		count := 0
		for _, entry := range entries {
			if !entry.IsDir() && (isImageFile(entry.Name()) || isArchiveFile(entry.Name())) {
				count++
			}
		}
		return count, hasArchives, nil
	}

	return 0, false, nil
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

	switch ext {
	case ".zip", ".cbz":
		return countImagesInZipWithSubdirs(archivePath)
	case ".rar", ".cbr":
		return countImagesInRarWithSubdirs(archivePath)
	case ".7z", ".cb7":
		return countImagesIn7zWithSubdirs(archivePath)
	default:
		return 0, false, nil
	}
}

func countImagesInRarWithSubdirs(rarPath string) (int, bool, error) {
	f, err := os.Open(rarPath)
	if err != nil {
		return 0, false, err
	}
	defer f.Close()

	r, err := rardecode.NewReader(f)
	if err != nil {
		return 0, false, err
	}

	count := 0
	hasSubdirs := false

	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, false, err
		}

		name := header.Name
		if strings.Contains(name, "/") || strings.Contains(name, "\\") {
			hasSubdirs = true
		}
		if !header.IsDir && isImageFile(name) {
			count++
		}
	}

	return count, hasSubdirs, nil
}

func countImagesIn7zWithSubdirs(sevenZipPath string) (int, bool, error) {
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return 0, false, err
	}
	defer r.Close()

	count := 0
	hasSubdirs := false

	for _, f := range r.File {
		name := f.Name
		if strings.Contains(name, "/") || strings.Contains(name, "\\") {
			hasSubdirs = true
		}
		if !f.FileInfo().IsDir() && isImageFile(name) {
			count++
		}
	}

	return count, hasSubdirs, nil
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

// GetDirContents returns the contents of a directory (subdirectories and archives)
func (a *App) GetDirContents(dirPath string) ([]VolumeInfo, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var volumes []VolumeInfo
	var directImages []string

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dirPath, name)

		if entry.IsDir() {
			imageCount, hasSubdirs, err := checkDirContents(path)
			if err == nil && imageCount > 0 {
				volumes = append(volumes, VolumeInfo{
					Name:              name,
					Path:              path,
					IsZip:             false,
					IsDir:             true,
					Images:            imageCount,
					HasSubdirectories: hasSubdirs,
				})
			}
		} else if isArchiveFile(name) {
			imageCount, hasSubdirs, err := countImagesInArchiveWithSubdirs(path)
			if err == nil && imageCount > 0 {
				volumes = append(volumes, VolumeInfo{
					Name:              name,
					Path:              path,
					IsZip:             true,
					IsDir:             false,
					Images:            imageCount,
					HasSubdirectories: hasSubdirs,
				})
			}
		} else if isImageFile(name) {
			directImages = append(directImages, name)
		}
	}

	// If there are images directly in the directory, add them as a single volume
	if len(directImages) > 0 {
		baseName := filepath.Base(dirPath)
		volumes = append(volumes, VolumeInfo{
			Name:   baseName + " (images)",
			Path:   dirPath,
			IsZip:  false,
			IsDir:  true,
			Images: len(directImages),
		})
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	return volumes, nil
}

// GetZipContents returns the contents of a zip file (nested zips, directories, or images)
func (a *App) GetZipContents(zipPath string) ([]VolumeInfo, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	var entries []string
	var nestedZips []VolumeInfo

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		entries = append(entries, name)

		// Detect nested zip files at root level
		if !strings.Contains(name, "/") && isZipFile(name) {
			nestedZips = append(nestedZips, VolumeInfo{
				Name:   filepath.Base(name),
				Path:   zipPath + "::" + name,
				IsZip:  true,
				IsDir:  false,
				Images: 0,
			})
		}
	}

	// Use common processing logic
	volumes, err := processArchiveFileEntries(zipPath, entries)
	if err != nil {
		return nil, err
	}

	// Append nested zips
	if len(nestedZips) > 0 {
		volumes = append(volumes, nestedZips...)
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Name < volumes[j].Name
		})
	}

	return volumes, nil
}

// GetArchiveContents returns the contents of any archive file (zip, rar, 7z)
func (a *App) GetArchiveContents(archivePath string) ([]VolumeInfo, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".zip", ".cbz":
		return a.GetZipContents(archivePath)
	case ".rar", ".cbr":
		return getRarContents(archivePath)
	case ".7z", ".cb7":
		return get7zContents(archivePath)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", ext)
	}
}

func getRarContents(rarPath string) ([]VolumeInfo, error) {
	f, err := os.Open(rarPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := rardecode.NewReader(f)
	if err != nil {
		return nil, err
	}

	var entries []string
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !header.IsDir {
			entries = append(entries, header.Name)
		}
	}

	return processArchiveFileEntries(rarPath, entries)
}

// processArchiveFileEntries analyzes archive entries and creates VolumeInfo list
// If there's only 1 top-level directory containing subdirectories, returns 2nd-level dirs as volumes
func processArchiveFileEntries(archivePath string, entries []string) ([]VolumeInfo, error) {
	// Analyze directory structure
	topDirs := make(map[string]bool)
	for _, entry := range entries {
		parts := strings.Split(entry, "/")
		if len(parts) > 1 {
			topDirs[parts[0]] = true
		}
	}

	// If there's only 1 top-level directory with subdirectories inside, expand it
	if len(topDirs) == 1 {
		var topDir string
		for d := range topDirs {
			topDir = d
		}

		// Count images in 2nd-level directories
		subDirs := make(map[string]int)
		for _, entry := range entries {
			parts := strings.Split(entry, "/")
			if len(parts) >= 3 && isImageFile(entry) {
				subDir := parts[1]
				subDirs[subDir]++
			}
		}

		// If there are multiple 2nd-level directories, return them as separate volumes
		if len(subDirs) > 1 {
			var volumes []VolumeInfo
			for dirName, imageCount := range subDirs {
				volumes = append(volumes, VolumeInfo{
					Name:   dirName,
					Path:   archivePath + "::" + topDir + "/" + dirName,
					IsZip:  true,
					IsDir:  true,
					Images: imageCount,
				})
			}
			sort.Slice(volumes, func(i, j int) bool {
				return volumes[i].Name < volumes[j].Name
			})
			return volumes, nil
		}
	}

	// Standard processing: top-level directories as volumes
	dirMap := make(map[string]int)
	rootImages := 0
	for _, entry := range entries {
		if strings.Contains(entry, "/") {
			parts := strings.Split(entry, "/")
			topDir := parts[0]
			if isImageFile(entry) {
				dirMap[topDir]++
			}
		} else if isImageFile(entry) {
			rootImages++
		}
	}

	var volumes []VolumeInfo
	for dirName, imageCount := range dirMap {
		if imageCount > 0 {
			volumes = append(volumes, VolumeInfo{
				Name:   dirName,
				Path:   archivePath + "::" + dirName,
				IsZip:  true,
				IsDir:  true,
				Images: imageCount,
			})
		}
	}

	if len(volumes) == 0 && rootImages > 0 {
		volumes = append(volumes, VolumeInfo{
			Name:   filepath.Base(archivePath),
			Path:   archivePath,
			IsZip:  true,
			IsDir:  false,
			Images: rootImages,
		})
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	return volumes, nil
}

func get7zContents(sevenZipPath string) ([]VolumeInfo, error) {
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var entries []string
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			entries = append(entries, f.Name)
		}
	}

	return processArchiveFileEntries(sevenZipPath, entries)
}

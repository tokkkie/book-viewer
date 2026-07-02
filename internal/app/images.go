package app

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

type ImageInfo struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	DataURL  string `json:"dataUrl"`
	MimeType string `json:"mimeType"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func (a *App) GetImageList(volumePath string, isZip bool) ([]string, error) {
	// Check if it's a path inside an archive (format: archivePath::innerPath)
	if strings.Contains(volumePath, "::") {
		parts := strings.SplitN(volumePath, "::", 2)
		if len(parts) == 2 {
			return getImageListFromArchiveSubdir(parts[0], parts[1])
		}
	}

	if isZip {
		return getImageListFromArchive(volumePath)
	}
	return getImageListFromDir(volumePath)
}

func (a *App) GetImageData(volumePath string, isZip bool, index int) (*ImageInfo, error) {
	imageList, err := a.GetImageList(volumePath, isZip)
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(imageList) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	imagePath := imageList[index]

	var data []byte
	var name string

	// Check if it's a path inside an archive (format: archivePath::innerPath)
	if strings.Contains(volumePath, "::") {
		parts := strings.SplitN(volumePath, "::", 2)
		if len(parts) == 2 {
			archivePath := parts[0]
			subdir := parts[1]
			// Use / for archive internal paths (RAR/7z use /)
			fullImagePath := subdir + "/" + imagePath
			data, name, err = readImageFromArchive(archivePath, fullImagePath)
			if err != nil {
				return nil, err
			}
		}
	} else if isZip {
		data, name, err = readImageFromArchive(volumePath, imagePath)
		if err != nil {
			return nil, err
		}
	} else {
		fullPath := filepath.Join(volumePath, imagePath)
		data, err = os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		name = filepath.Base(imagePath)
	}

	mimeType := getMimeType(name)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))

	info := &ImageInfo{
		Index:    index,
		Name:     name,
		DataURL:  dataURL,
		MimeType: mimeType,
	}

	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		bounds := img.Bounds()
		info.Width = bounds.Dx()
		info.Height = bounds.Dy()
	}

	return info, nil
}

func (a *App) GetImageRange(volumePath string, isZip bool, startIndex, count int) ([]*ImageInfo, error) {
	imageList, err := a.GetImageList(volumePath, isZip)
	if err != nil {
		return nil, err
	}

	if startIndex < 0 {
		startIndex = 0
	}
	endIndex := startIndex + count
	if endIndex > len(imageList) {
		endIndex = len(imageList)
	}

	var images []*ImageInfo
	for i := startIndex; i < endIndex; i++ {
		img, err := a.GetImageData(volumePath, isZip, i)
		if err != nil {
			continue
		}
		images = append(images, img)
	}

	return images, nil
}

func getImageListFromDir(dirPath string) ([]string, error) {
	var images []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isImageFile(path) {
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			images = append(images, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortImagesNatural(images)
	return images, nil
}

func getImageListFromArchive(archivePath string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".zip", ".cbz":
		return getImageListFromZip(archivePath)
	case ".rar", ".cbr":
		return getImageListFromRar(archivePath)
	case ".7z", ".cb7":
		return getImageListFrom7z(archivePath)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", ext)
	}
}

func getImageListFromZip(zipPath string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var images []string
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && isImageFile(f.Name) {
			images = append(images, f.Name)
		}
	}

	sortImagesNatural(images)
	return images, nil
}

func getImageListFromRar(rarPath string) ([]string, error) {
	f, err := os.Open(rarPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := rardecode.NewReader(f)
	if err != nil {
		return nil, err
	}

	var images []string
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !header.IsDir && isImageFile(header.Name) {
			images = append(images, header.Name)
		}
	}

	sortImagesNatural(images)
	return images, nil
}

func getImageListFrom7z(sevenZipPath string) ([]string, error) {
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var images []string
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && isImageFile(f.Name) {
			images = append(images, f.Name)
		}
	}

	sortImagesNatural(images)
	return images, nil
}

func getImageListFromArchiveSubdir(archivePath, subdir string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".zip", ".cbz":
		return getImageListFromZipSubdir(archivePath, subdir)
	case ".rar", ".cbr":
		return getImageListFromRarSubdir(archivePath, subdir)
	case ".7z", ".cb7":
		return getImageListFrom7zSubdir(archivePath, subdir)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", ext)
	}
}

func getImageListFromZipSubdir(zipPath, subdir string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	prefix := strings.TrimPrefix(subdir, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var images []string
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && strings.HasPrefix(f.Name, prefix) && isImageFile(f.Name) {
			// Store relative path within the subdirectory
			relPath := strings.TrimPrefix(f.Name, prefix)
			images = append(images, relPath)
		}
	}

	sortImagesNatural(images)
	return images, nil
}

func getImageListFromRarSubdir(rarPath, subdir string) ([]string, error) {
	f, err := os.Open(rarPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := rardecode.NewReader(f)
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimPrefix(subdir, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var images []string
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !header.IsDir && strings.HasPrefix(header.Name, prefix) && isImageFile(header.Name) {
			relPath := strings.TrimPrefix(header.Name, prefix)
			images = append(images, relPath)
		}
	}

	sortImagesNatural(images)
	return images, nil
}

func getImageListFrom7zSubdir(sevenZipPath, subdir string) ([]string, error) {
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	prefix := strings.TrimPrefix(subdir, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var images []string
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && strings.HasPrefix(f.Name, prefix) && isImageFile(f.Name) {
			relPath := strings.TrimPrefix(f.Name, prefix)
			images = append(images, relPath)
		}
	}

	sortImagesNatural(images)
	return images, nil
}

func readImageFromArchive(archivePath, imagePath string) ([]byte, string, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".zip", ".cbz":
		return readImageFromZip(archivePath, imagePath)
	case ".rar", ".cbr":
		return readImageFromRar(archivePath, imagePath)
	case ".7z", ".cb7":
		return readImageFrom7z(archivePath, imagePath)
	default:
		return nil, "", fmt.Errorf("unsupported archive format: %s", ext)
	}
}

func readImageFromZip(zipPath, imagePath string) ([]byte, string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == imagePath {
			rc, err := f.Open()
			if err != nil {
				return nil, "", err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, "", err
			}

			return data, filepath.Base(f.Name), nil
		}
	}

	return nil, "", fmt.Errorf("image not found in zip: %s", imagePath)
}

func readImageFromRar(rarPath, imagePath string) ([]byte, string, error) {
	f, err := os.Open(rarPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	r, err := rardecode.NewReader(f)
	if err != nil {
		return nil, "", err
	}

	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}

		if header.Name == imagePath {
			data, err := io.ReadAll(r)
			if err != nil {
				return nil, "", err
			}
			return data, filepath.Base(header.Name), nil
		}
	}

	return nil, "", fmt.Errorf("image not found in rar: %s", imagePath)
}

func readImageFrom7z(sevenZipPath, imagePath string) ([]byte, string, error) {
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == imagePath {
			rc, err := f.Open()
			if err != nil {
				return nil, "", err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, "", err
			}

			return data, filepath.Base(f.Name), nil
		}
	}

	return nil, "", fmt.Errorf("image not found in 7z: %s", imagePath)
}

func naturalLess(a, b string) bool {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		ca, cb := a[i], b[j]
		if isDigit(ca) && isDigit(cb) {
			si, sj := i, j
			for i < len(a) && isDigit(a[i]) {
				i++
			}
			for j < len(b) && isDigit(b[j]) {
				j++
			}
			na, _ := strconv.Atoi(a[si:i])
			nb, _ := strconv.Atoi(b[sj:j])
			if na != nb {
				return na < nb
			}
		} else {
			if ca != cb {
				if ca == ' ' {
					return false
				}
				if cb == ' ' {
					return true
				}
				return ca < cb
			}
			i++
			j++
		}
	}
	return len(a) < len(b)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func sortImagesNatural(images []string) {
	sort.Slice(images, func(i, j int) bool {
		return naturalLess(images[i], images[j])
	})
}

func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}

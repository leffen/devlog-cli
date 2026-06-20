package cmd

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/leffen/devlog-cli/internal/api"
)

const maxEntryImageBytes = 3 * 1024 * 1024

var imageIDPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func loadEntryImages(paths []string) ([]api.Image, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if len(paths) > 8 {
		return nil, fmt.Errorf("a maximum of 8 images can be attached to one entry")
	}

	images := make([]api.Image, 0, len(paths))
	seen := map[string]int{}
	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading image %q: %w", path, err)
		}
		if len(data) > maxEntryImageBytes {
			return nil, fmt.Errorf("image %q is larger than 3 MB", path)
		}

		mimeType := detectImageMime(path, data)
		if !strings.HasPrefix(mimeType, "image/") {
			return nil, fmt.Errorf("unsupported image type for %q", path)
		}

		id := imageID(filepath.Base(path))
		if seen[id] > 0 {
			seen[id]++
			id = fmt.Sprintf("%s-%d", id, seen[id])
		} else {
			seen[id] = 1
		}

		images = append(images, api.Image{
			ID:       id,
			Alt:      strings.NewReplacer("-", " ", "_", " ").Replace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
			Filename: filepath.Base(path),
			MimeType: mimeType,
			Src:      fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)),
		})
	}

	return images, nil
}

func detectImageMime(path string, data []byte) string {
	mimeType := http.DetectContentType(data)
	if strings.HasPrefix(mimeType, "image/") {
		return mimeType
	}
	if fromExt := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); strings.HasPrefix(fromExt, "image/") {
		return fromExt
	}
	return mimeType
}

func imageID(filename string) string {
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	id := imageIDPattern.ReplaceAllString(strings.TrimSpace(stem), "-")
	id = strings.Trim(id, "-")
	if len(id) > 80 {
		id = id[:80]
	}
	if id == "" {
		return "image"
	}
	return id
}

func appendMissingImagePlaceholders(content string, images []api.Image) string {
	if len(images) == 0 {
		return content
	}

	missing := make([]string, 0, len(images))
	for _, image := range images {
		placeholder := fmt.Sprintf("{{image:%s}}", image.ID)
		if !strings.Contains(content, placeholder) {
			missing = append(missing, placeholder)
		}
	}
	if len(missing) == 0 {
		return content
	}

	return strings.TrimSpace(content) + "\n\n" + strings.Join(missing, "\n\n")
}

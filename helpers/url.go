package helpers

import (
	"net/url"
	"strings"
)

func ExtractUrlPath(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://dummy/" + strings.TrimLeft(rawURL, "/")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "/"
	}

	path := parsed.Path

	// Всегда начинаем со слеша
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Удаляем конечный слэш, если путь не просто "/"
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	if path == "/" || path == "" {
		return "/"
	}

	return path
}

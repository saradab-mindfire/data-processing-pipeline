package dataio

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const localSourceRoot = "uploads"

var httpSourceClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func openSource(path string) (io.ReadCloser, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return openRemoteSource(path)
	}
	return openLocalSource(path)
}

func openLocalSource(path string) (io.ReadCloser, error) {
	root, err := filepath.Abs(localSourceRoot)
	if err != nil {
		return nil, err
	}

	full, err := filepath.Abs(filepath.Join(root, path))
	if err != nil {
		return nil, err
	}
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return nil, fmt.Errorf("path %q is outside the allowed uploads directory", path)
	}

	return os.Open(full)
}

func openRemoteSource(rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %q", u.Scheme)
	}

	resp, err := httpSourceClient.Get(rawURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return resp.Body, nil
}


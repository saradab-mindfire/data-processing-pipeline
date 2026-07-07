package pipelines

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func openSource(path string) (io.ReadCloser, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= http.StatusBadRequest {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status %s", resp.Status)
		}
		return resp.Body, nil
	}
	return os.Open(path)
}
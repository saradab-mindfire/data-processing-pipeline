package dataio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenLocalSourceReadsFile(t *testing.T) {
	name := writeUploadFixture(t, "source_local.txt", "hello world")

	rc, err := openSource(name)
	if err != nil {
		t.Fatalf("openSource returned error: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read source: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", string(data), "hello world")
	}
}

func TestOpenLocalSourceRejectsPathTraversal(t *testing.T) {
	_, err := openSource("../source_test.go")
	if err == nil {
		t.Fatal("expected an error for a path escaping the uploads directory")
	}
}

func TestOpenLocalSourceMissingFile(t *testing.T) {
	_, err := openSource("does-not-exist.csv")
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestOpenRemoteSourceSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("remote content"))
	}))
	defer server.Close()

	rc, err := openSource(server.URL)
	if err != nil {
		t.Fatalf("openSource returned error: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read remote source: %v", err)
	}
	if string(data) != "remote content" {
		t.Errorf("content = %q, want %q", string(data), "remote content")
	}
}

func TestOpenRemoteSourceErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := openSource(server.URL)
	if err == nil {
		t.Fatal("expected an error for a non-2xx remote response")
	}
}

func TestOpenSourceRejectsUnsupportedScheme(t *testing.T) {
	_, err := openRemoteSource("ftp://example.com/file.csv")
	if err == nil {
		t.Fatal("expected an error for an unsupported URL scheme")
	}
}

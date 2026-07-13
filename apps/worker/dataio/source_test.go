package dataio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func unrestrictedHTTPClient(t *testing.T) {
	t.Helper()
	previous := httpSourceClient
	httpSourceClient = &http.Client{
		Timeout:       previous.Timeout,
		CheckRedirect: previous.CheckRedirect,
	}
	t.Cleanup(func() { httpSourceClient = previous })
}

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
	unrestrictedHTTPClient(t)

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
	unrestrictedHTTPClient(t)

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

func TestOpenRemoteSourceBlocksLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := openSource(server.URL)
	if err == nil {
		t.Fatal("expected an error for a request targeting a loopback address")
	}
}

func TestDenyPrivateAddresses(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"loopback", "127.0.0.1:80", true},
		{"loopback ipv6", "[::1]:80", true},
		{"unspecified", "0.0.0.0:80", true},
		{"link-local unicast", "169.254.1.1:80", true},
		{"link-local multicast", "[ff02::1]:80", true},
		{"multicast", "224.0.0.1:80", true},
		{"private class A", "10.0.0.1:80", true},
		{"private class B", "172.16.0.1:80", true},
		{"private class C", "192.168.1.1:80", true},
		{"public", "93.184.216.34:80", false},
		{"malformed address", "not-an-address", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := denyPrivateAddresses("tcp", tt.address, nil)
			if tt.wantErr && err == nil {
				t.Errorf("denyPrivateAddresses(%q) = nil, want error", tt.address)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("denyPrivateAddresses(%q) = %v, want nil", tt.address, err)
			}
		})
	}
}

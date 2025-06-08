package radiko

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Errorf("min failed")
	}
	if min(10, -1) != -1 {
		t.Errorf("min failed")
	}
}

func TestGeneratePartialKey(t *testing.T) {
	c := &Client{logger: NewLogger(false)}
	got, err := c.generatePartialKey("5", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "YmNkMTU=" // base64 of "bcd15"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
	if _, err := c.generatePartialKey("-1", "0"); err == nil {
		t.Errorf("expected error for invalid length")
	}
}

func TestFetchSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			io.WriteString(w, "#EXTM3U\nsub1.m3u8\nsegment1.ts\n")
		case "/sub1.m3u8":
			io.WriteString(w, "#EXTM3U\nsegment2.ts\nsegment3.ts\n")
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client(), logger: NewLogger(false)}
	segments, err := c.fetchSegments(server.URL + "/main.m3u8")
	if err != nil {
		t.Fatalf("fetchSegments error: %v", err)
	}
	expected := []string{
		server.URL + "/segment2.ts",
		server.URL + "/segment3.ts",
		server.URL + "/segment1.ts",
	}
	if !reflect.DeepEqual(segments, expected) {
		t.Errorf("segments mismatch:\nexpected=%v\n got=%v", expected, segments)
	}
}

func TestGetAvailableStations(t *testing.T) {
	stations := GetAvailableStations()
	if stations["TBS"] == "" {
		t.Errorf("expected TBS station")
	}
}

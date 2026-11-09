package tikdownloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Regression test for the \K panic: Go RE2 must compile these patterns and
// the escaped JSON-HTML response from tikdownloader.io must be parsed.
func TestExtractDownloadURL(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string
	}{
		{
			name: "mp4 hd raw",
			html: `<a href="https://dl.snapcdn.app/video/abc.mp4" class="x"><i class="i"></i>Download MP4 HD</a>`,
			want: "https://dl.snapcdn.app/video/abc.mp4",
		},
		{
			name: "mp4 hd escaped json",
			html: `{\"status\":\"ok\",\"data\":\"<a href=\"https:\/\/dl.snapcdn.app\/video\/abc.mp4\" ><i></i>Download MP4 HD</a>\"}`,
			want: "https://dl.snapcdn.app/video/abc.mp4",
		},
		{
			name: "snapcdn fallback skips mp3",
			html: `<a href="https://dl.snapcdn.app/a.mp3">x</a><a href="https://dl.snapcdn.app/b.mp4">y</a>`,
			want: "https://dl.snapcdn.app/b.mp4",
		},
		{
			name: "v16 fallback",
			html: `<a href="https://v16m-default.tiktokcdn.com/v/xyz.mp4?sig=1&amp;exp=2">x</a>`,
			want: "https://v16m-default.tiktokcdn.com/v/xyz.mp4?sig=1&exp=2",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractDownloadURL(tc.html)
			if err != nil {
				t.Fatalf("extractDownloadURL: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	if _, err := extractDownloadURL("<html>no links</html>"); err == nil {
		t.Fatal("expected error for HTML with no download link")
	}
}

func TestDownloadFileParallelAndSingle(t *testing.T) {
	payload := []byte(strings.Repeat("0123456789abcdef", 4096)) // 64 KiB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "v.mp4", time.Now(), strings.NewReader(string(payload)))
	}))
	defer srv.Close()
	client := srv.Client()

	for _, conns := range []int{1, 5} {
		out := filepath.Join(t.TempDir(), fmt.Sprintf("out-%d.mp4", conns))
		if err := downloadFile(context.Background(), client, srv.URL, out, conns); err != nil {
			t.Fatalf("connections=%d: %v", conns, err)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(payload) {
			t.Fatalf("connections=%d: got %d bytes, want %d", conns, len(got), len(payload))
		}
	}
}

func TestApplyNamingTemplate(t *testing.T) {
	got := ApplyNamingTemplate("%(upload_date)s_%(id)s_%(title)s.%(ext)s",
		VideoMeta{ID: "123", Title: "Hello World! (2026)", Uploader: "someone", UploadDate: "20261109"}, "mp4")
	if got != "20261109_123_Hello_World_2026.mp4" {
		t.Fatalf("got %q", got)
	}
	if got := ApplyNamingTemplate("", VideoMeta{ID: "9"}, "mp4"); got != "tiktok_9.mp4" {
		t.Fatalf("empty template: got %q", got)
	}
	if got := ApplyNamingTemplate("%(uploader)s/%(id)s.%(ext)s", VideoMeta{ID: "1", Uploader: "a/b"}, "mp4"); strings.Contains(got, "/") {
		t.Fatalf("path separator not stripped: %q", got)
	}
}

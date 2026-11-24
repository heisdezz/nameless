package tikdownloader

import (
	"context"
	"encoding/json"
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

func TestEnginesDownloadVideo(t *testing.T) {
	videoPayload := []byte("fake-mp4-content-1234567890")
	audioPayload := []byte("fake-mp3-content-1234567890")

	// Mock CDN server serving the media bytes.
	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".mp3") {
			_, _ = w.Write(audioPayload)
			return
		}
		_, _ = w.Write(videoPayload)
	}))
	defer cdnSrv.Close()

	t.Run("tikwm hd and audio", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/" {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{
					"hdplay": cdnSrv.URL + "/hd.mp4",
					"play":   cdnSrv.URL + "/sd.mp4",
					"music":  cdnSrv.URL + "/audio.mp3",
				},
			})
		}))
		defer srv.Close()

		outDir := t.TempDir()
		dl := NewWithOptions(Options{
			Engine:      EngineTikWM,
			APIEndpoint: srv.URL,
			Format:      "hd",
			Connections: 1,
			HTTPClient:  srv.Client(),
		})
		res, err := dl.DownloadVideo(context.Background(), "https://www.tiktok.com/@u/video/111", outDir)
		if err != nil || !res.Success {
			t.Fatalf("tikwm hd failed: res=%+v err=%v", res, err)
		}
		if !strings.HasSuffix(res.Path, ".mp4") {
			t.Fatalf("expected .mp4, got %q", res.Path)
		}
		got, _ := os.ReadFile(res.Path)
		if string(got) != string(videoPayload) {
			t.Fatalf("unexpected payload %q", string(got))
		}

		// Audio mode
		dlAudio := NewWithOptions(Options{
			Engine:      EngineTikWM,
			APIEndpoint: srv.URL,
			Format:      "audio_only",
			Connections: 1,
			HTTPClient:  srv.Client(),
		})
		resA, err := dlAudio.DownloadVideo(context.Background(), "https://www.tiktok.com/@u/video/112", outDir)
		if err != nil || !resA.Success {
			t.Fatalf("tikwm audio failed: res=%+v err=%v", resA, err)
		}
		if !strings.HasSuffix(resA.Path, ".mp3") {
			t.Fatalf("expected .mp3, got %q", resA.Path)
		}
	})

	t.Run("lovetik", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/ajax/search" {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"links": []map[string]any{
					{"a": cdnSrv.URL + "/audio.mp3", "t": "<b>MP3</b> Audio"},
					{"a": cdnSrv.URL + "/video.mp4", "t": "<b>MP4</b> [1080p] (No Watermark)"},
				},
			})
		}))
		defer srv.Close()

		dl := NewWithOptions(Options{
			Engine:      EngineLoveTik,
			APIEndpoint: srv.URL,
			Connections: 1,
			HTTPClient:  srv.Client(),
		})
		res, err := dl.DownloadVideo(context.Background(), "https://www.tiktok.com/@u/video/222", t.TempDir())
		if err != nil || !res.Success {
			t.Fatalf("lovetik failed: res=%+v err=%v", res, err)
		}
		got, _ := os.ReadFile(res.Path)
		if string(got) != string(videoPayload) {
			t.Fatalf("unexpected payload %q", string(got))
		}
	})

	t.Run("ssstik", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/en":
				_, _ = fmt.Fprint(w, `<script>var s_tt = 'tok123';</script>`)
			case "/abc":
				if err := r.ParseForm(); err != nil || r.FormValue("tt") != "tok123" {
					http.Error(w, "bad token", http.StatusBadRequest)
					return
				}
				_, _ = fmt.Fprintf(w, `<a href="%s/sss.mp4" class="pure-button download_link without_watermark">Without watermark</a>`, cdnSrv.URL)
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		dl := NewWithOptions(Options{
			Engine:      EngineSSSTik,
			APIEndpoint: srv.URL,
			Connections: 1,
			HTTPClient:  srv.Client(),
		})
		res, err := dl.DownloadVideo(context.Background(), "https://www.tiktok.com/@u/video/333", t.TempDir())
		if err != nil || !res.Success {
			t.Fatalf("ssstik failed: res=%+v err=%v", res, err)
		}
	})

	t.Run("auto fallback when primary is rate-limited", func(t *testing.T) {
		rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
		}))
		defer rateLimited.Close()

		tikwmOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{"hdplay": cdnSrv.URL + "/ok.mp4"},
			})
		}))
		defer tikwmOK.Close()

		dl := NewWithOptions(Options{
			Engine:      EngineAuto,
			Connections: 1,
			HTTPClient:  tikwmOK.Client(),
			engineEndpoints: map[string]string{
				EngineTikDownloader: rateLimited.URL,
				EngineTikWM:         tikwmOK.URL,
				EngineLoveTik:       rateLimited.URL,
				EngineSSSTik:        rateLimited.URL,
			},
		})
		res, err := dl.DownloadVideo(context.Background(), "https://www.tiktok.com/@u/video/444", t.TempDir())
		if err != nil || !res.Success {
			t.Fatalf("auto fallback failed: res=%+v err=%v", res, err)
		}
	})

	t.Run("rate limit error surfaces clearly", func(t *testing.T) {
		rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "slow down", http.StatusTooManyRequests)
		}))
		defer rateLimited.Close()

		dl := NewWithOptions(Options{
			Engine:      EngineTikDownloader,
			APIEndpoint: rateLimited.URL,
			Connections: 1,
			HTTPClient:  rateLimited.Client(),
		})
		res, err := dl.DownloadVideo(context.Background(), "https://www.tiktok.com/@u/video/555", t.TempDir())
		if err == nil || res.Err == nil || !strings.Contains(err.Error(), "429") {
			t.Fatalf("expected 429 error, got res=%+v err=%v", res, err)
		}
	})
}

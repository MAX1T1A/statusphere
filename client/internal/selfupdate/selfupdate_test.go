package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"statusphere-client/internal/version"
)

func releaseJSON(tag string, assets ...string) string {
	var parts []string
	for _, a := range assets {
		parts = append(parts, fmt.Sprintf(`{"name":%q,"browser_download_url":"http://example/%s"}`, a, a))
	}
	return fmt.Sprintf(`{"tag_name":%q,"assets":[%s]}`, tag, strings.Join(parts, ","))
}

func withAPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := APIBase
	APIBase = srv.URL
	t.Cleanup(func() { APIBase = old })
}

func TestLatestPicksThisArch(t *testing.T) {
	withAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, releaseJSON("v9.9.9", "statusphere-linux-amd64", "statusphere-linux-arm64"))
	})

	rel, err := Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != "v9.9.9" {
		t.Fatalf("tag = %q", rel.Version)
	}
	if !strings.HasSuffix(rel.AssetURL, "statusphere-linux-"+runtime.GOARCH) {
		t.Fatalf("wrong asset for %s: %s", runtime.GOARCH, rel.AssetURL)
	}
}

func TestLatestErrorsWithoutMatchingAsset(t *testing.T) {
	withAPI(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, releaseJSON("v9.9.9", "statusphere-darwin-arm64"))
	})
	if _, err := Latest(context.Background()); err == nil {
		t.Fatal("expected an error when no asset matches this platform")
	}
}

func TestLatestErrorsOnBadStatus(t *testing.T) {
	withAPI(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
	if _, err := Latest(context.Background()); err == nil {
		t.Fatal("expected an error on non-200")
	}
}

func TestIsNewer(t *testing.T) {
	old := version.Version
	version.Version = "v0.3.0"
	t.Cleanup(func() { version.Version = old })

	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.3.1", "v0.3.0", true},
		{"v0.4.0", "v0.3.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.3.0", "v0.3.0", false},
		{"v0.2.9", "v0.3.0", false},
		{"0.3.1", "0.3.0", true},
		{"v0.3.1-rc1", "v0.3.0", true},
		{"", "v0.3.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestIsNewerAlwaysTrueForDevBuild(t *testing.T) {
	old := version.Version
	version.Version = "dev"
	t.Cleanup(func() { version.Version = old })

	if !IsNewer("v0.0.1", "dev") {
		t.Fatal("a dev build should always accept a real release")
	}
}

func TestApplyReplacesBinaryAtomically(t *testing.T) {
	payload := strings.Repeat("B", 2<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := applyTo(context.Background(), &Release{Version: "v1.2.3", AssetURL: srv.URL}, exe)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Fatalf("binary not replaced (%d bytes)", len(got))
	}
	info, _ := os.Stat(exe)
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("replacement is not executable: %v", info.Mode())
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("temp files left behind: %v", names)
	}
}

func TestApplyRejectsTruncatedDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "too small")
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	os.WriteFile(exe, []byte("old binary"), 0o755)

	if err := applyTo(context.Background(), &Release{AssetURL: srv.URL}, exe); err == nil {
		t.Fatal("expected truncation to be rejected")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("a failed update must leave the existing binary untouched")
	}
}

func TestApplyKeepsBinaryOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	os.WriteFile(exe, []byte("old binary"), 0o755)

	if err := applyTo(context.Background(), &Release{AssetURL: srv.URL}, exe); err == nil {
		t.Fatal("expected an error on 500")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("binary must survive a failed download")
	}
}

func TestApplyKeepsExistingPermissions(t *testing.T) {
	payload := strings.Repeat("B", 2<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	for _, mode := range []os.FileMode{0o700, 0o750, 0o755} {
		dir := t.TempDir()
		exe := filepath.Join(dir, "statusphere")
		os.WriteFile(exe, []byte("old binary"), 0o600)
		if err := os.Chmod(exe, mode); err != nil {
			t.Fatal(err)
		}

		if err := applyTo(context.Background(), &Release{AssetURL: srv.URL}, exe); err != nil {
			t.Fatal(err)
		}
		fi, _ := os.Stat(exe)
		if fi.Mode().Perm() != mode {
			t.Errorf("permissions widened: %04o -> %04o", mode, fi.Mode().Perm())
		}
	}
}

func TestApplyRejectsOversizeAssetInsteadOfTruncating(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.CopyN(w, zeros{}, maxAsset+4096)
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	os.WriteFile(exe, []byte("old binary"), 0o755)

	err := applyTo(context.Background(), &Release{AssetURL: srv.URL}, exe)
	if err == nil {
		t.Fatal("an oversize asset must be rejected, not truncated into place")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("binary must survive an oversize download")
	}
}

func TestApplyRejectsShortBodyAgainstContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(4<<20))
		io.CopyN(w, zeros{}, 2<<20)
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	os.WriteFile(exe, []byte("old binary"), 0o755)

	if err := applyTo(context.Background(), &Release{AssetURL: srv.URL}, exe); err == nil {
		t.Fatal("a body shorter than Content-Length must be rejected")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("binary must survive an incomplete download")
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestApplyWithoutReleaseIsAnError(t *testing.T) {
	if err := applyTo(context.Background(), nil, "/nonexistent"); err == nil {
		t.Fatal("expected an error for a nil release")
	}
}

func TestIsNewerHandlesOddTags(t *testing.T) {
	old := version.Version
	version.Version = "v0.3.0"
	t.Cleanup(func() { version.Version = old })

	cases := []struct {
		latest, current string
		want            bool
		why             string
	}{
		{"v0.4", "v0.3.0", true, "short tag must not read as older"},
		{"nightly", "v0.3.0", true, "unparseable tag differs -> offer it"},
		{"v0.3.0", "v0.3.0", false, "identical tags"},
		{"nightly", "nightly", false, "identical unparseable tags"},
		{"v1.2.3.4", "v1.2.3", true, "four-part tag differs"},
		{"v0.4.0", "v0.4.0-rc1", true, "final release beats its own rc"},
		{"v0.4.0-rc2", "v0.4.0", false, "rc must not replace the final"},
		{"v0.5.0-rc1", "v0.4.0", true, "newer rc still wins on numbers"},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q,%q)=%v want %v — %s", c.latest, c.current, got, c.want, c.why)
		}
	}
}

func TestSweepRemovesOnlyStaleStagedFiles(t *testing.T) {
	dir := t.TempDir()
	fresh := filepath.Join(dir, ".statusphere-update-fresh")
	stale := filepath.Join(dir, ".statusphere-update-stale")
	keep := filepath.Join(dir, "statusphere")
	for _, f := range []string{fresh, stale, keep} {
		os.WriteFile(f, []byte("x"), 0o600)
	}
	os.Chtimes(stale, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))

	sweepStaged(dir)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("a stale staged file should be swept")
	}
	for _, f := range []string{fresh, keep} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("%s must survive the sweep", filepath.Base(f))
		}
	}
}

func TestDownloadIsNotCappedByTheMetadataClientTimeout(t *testing.T) {
	payload := strings.Repeat("B", 2<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		half := len(payload) / 2
		w.Write([]byte(payload[:half]))
		w.(http.Flusher).Flush()
		time.Sleep(300 * time.Millisecond)
		w.Write([]byte(payload[half:]))
	}))
	defer srv.Close()

	restore := client.Timeout
	client.Timeout = 100 * time.Millisecond
	t.Cleanup(func() { client.Timeout = restore })

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	os.WriteFile(exe, []byte("old"), 0o755)

	if err := applyTo(context.Background(), &Release{AssetURL: srv.URL}, exe); err != nil {
		t.Fatalf("a slow download must not be killed by the metadata client timeout: %v", err)
	}
	if got, _ := os.ReadFile(exe); len(got) != len(payload) {
		t.Fatalf("installed %d bytes, want %d", len(got), len(payload))
	}
}

func TestDownloadStillObeysContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(4<<20))
		for i := 0; i < 40; i++ {
			w.Write(make([]byte, 100<<10))
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "statusphere")
	os.WriteFile(exe, []byte("old"), 0o755)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := applyTo(ctx, &Release{AssetURL: srv.URL}, exe); err == nil {
		t.Fatal("the context deadline must still abort a download")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Fatal("binary must survive an aborted download")
	}
}

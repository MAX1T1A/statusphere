package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"statusphere-client/internal/version"
)

const (
	repo        = "MAX1T1A/statusphere"
	assetPrefix = "statusphere-linux-"
	maxAsset    = 64 << 20
)

var (
	APIBase = "https://api.github.com"
	client  = &http.Client{Timeout: 30 * time.Second}
	// a real build is several MB; anything tiny means a truncated download
	minAssetSize int64 = 1 << 20
)

type Release struct {
	Version  string
	AssetURL string
}

// Latest reports the newest published release and the asset for this platform.
func Latest(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(APIBase, "/"), repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.TagName == "" {
		return nil, fmt.Errorf("github: no tag in latest release")
	}

	want := assetPrefix + runtime.GOARCH
	rel := &Release{Version: payload.TagName}
	for _, a := range payload.Assets {
		if a.Name == want {
			rel.AssetURL = a.URL
			break
		}
	}
	if rel.AssetURL == "" {
		return nil, fmt.Errorf("no build for %s/%s in %s", runtime.GOOS, runtime.GOARCH, payload.TagName)
	}
	return rel, nil
}

// Apply downloads the release binary and swaps it in place of the running one.
// The replacement takes effect on the next start.
func Apply(ctx context.Context, rel *Release) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return applyTo(ctx, rel, exe)
}

func applyTo(ctx context.Context, rel *Release, exe string) error {
	if rel == nil || rel.AssetURL == "" {
		return fmt.Errorf("nothing to install")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.AssetURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: status %d", resp.StatusCode)
	}

	// stage next to the target so the final swap is an atomic same-fs rename
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".statusphere-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w", filepath.Dir(exe), err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	// read one byte past the cap so an oversize asset is an error, never a
	// silently truncated binary
	written, err := io.Copy(tmp, io.LimitReader(resp.Body, maxAsset+1))
	if err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if written > maxAsset {
		return fmt.Errorf("release asset exceeds %d bytes", maxAsset)
	}
	if written < minAssetSize {
		return fmt.Errorf("downloaded file looks truncated (%d bytes)", written)
	}
	if n := resp.ContentLength; n > 0 && written != n {
		return fmt.Errorf("incomplete download: got %d of %d bytes", written, n)
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	return os.Rename(tmpName, exe)
}

// IsNewer reports whether latest is a higher version than current. A dev build
// is always considered outdated so a real release can replace it.
func IsNewer(latest, current string) bool {
	if latest == "" {
		return false
	}
	if version.IsDev() || current == "" {
		return true
	}
	l, c := parse(latest), parse(current)
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parse(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	for i, part := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return out
		}
		out[i] = n
	}
	return out
}

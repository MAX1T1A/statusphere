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
	APIBase              = "https://api.github.com"
	client               = &http.Client{Timeout: 30 * time.Second}
	downloadClient       = &http.Client{}
	minAssetSize   int64 = 1 << 20
	staleStaged          = time.Hour
)

type Release struct {
	Version  string
	AssetURL string
}

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
	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: status %d", resp.StatusCode)
	}

	sweepStaged(filepath.Dir(exe))

	tmp, err := os.CreateTemp(filepath.Dir(exe), ".statusphere-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w", filepath.Dir(exe), err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

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
	mode := os.FileMode(0o755)
	if fi, err := os.Stat(exe); err == nil {
		mode = fi.Mode().Perm() | 0o100
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, exe)
}

func sweepStaged(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, ".statusphere-update-*"))
	if err != nil {
		return
	}
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil || fi.IsDir() || time.Since(fi.ModTime()) < staleStaged {
			continue
		}
		os.Remove(m)
	}
}

func IsNewer(latest, current string) bool {
	if latest == "" {
		return false
	}
	if version.IsDev() || current == "" {
		return true
	}

	l, lPre, lOK := parse(latest)
	c, cPre, cOK := parse(current)
	if !lOK || !cOK {
		return strings.TrimSpace(latest) != strings.TrimSpace(current)
	}

	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return cPre != "" && lPre == ""
}

func parse(v string) (nums [3]int, pre string, ok bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre, v = v[i+1:], v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		return nums, pre, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nums, pre, false
		}
		nums[i] = n
	}
	return nums, pre, true
}

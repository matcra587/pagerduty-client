package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gechr/clog"
)

const (
	projectName     = "pagerduty-client"
	binaryName      = "pdc"
	downloadTimeout = 60 * time.Second
	maxBinarySize   = 100 << 20 // 100 MiB
)

// releaseAsset describes a single asset in a GitHub release.
type releaseAsset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// AssetName returns the archive filename matching the goreleaser template
// for the given version, OS and architecture.
func AssetName(version, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s.%s", projectName, version, goos, goarch, ext)
}

// VerifyChecksum computes the SHA-256 hash of data and compares it
// against the hash in checksumLine for the given assetName. The
// checksumLine format is "<hex>  <filename>" (two spaces, matching
// sha256sum output).
func VerifyChecksum(data []byte, checksumLine, assetName string) bool {
	parts := strings.Fields(checksumLine)
	if len(parts) != 2 {
		return false
	}

	if parts[1] != assetName {
		return false
	}

	got := sha256.Sum256(data)
	actual := []byte(hex.EncodeToString(got[:]))
	return subtle.ConstantTimeCompare(actual, []byte(parts[0])) == 1
}

// AtomicReplace writes data to a temporary file in the same directory
// as target, then renames it into place. This avoids partial writes.
func AtomicReplace(target string, data []byte) error {
	dir := filepath.Dir(target)

	tmp, err := os.CreateTemp(dir, ".pdc-update-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// fetchReleaseAssets returns the assets for the given tag from the
// GitHub API. Authenticates using go-gh's token resolution (GH_TOKEN,
// GITHUB_TOKEN or gh auth login).
func fetchReleaseAssets(ctx context.Context, tag string) ([]releaseAsset, error) {
	url := defaultAPIBase + "/repos/matcra587/pagerduty-client/releases/tags/" + tag
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching release %s: %w", tag, err)
	}

	var release struct {
		Assets []releaseAsset `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}
	return release.Assets, nil
}

// downloadAsset downloads a release asset by ID via the GitHub API.
// Uses Accept: application/octet-stream to stream the binary directly.
func downloadAsset(ctx context.Context, assetID int) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/matcra587/pagerduty-client/releases/assets/%d", defaultAPIBase, assetID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	setGitHubAuth(req)
	req.Header.Set("Accept", "application/octet-stream")

	client := &http.Client{
		CheckRedirect: stripAuthOnRedirect,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d downloading asset %d", resp.StatusCode, assetID)
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxBinarySize))
}

// selfReplace downloads the latest release archive, verifies its
// checksum, extracts the binary and atomically replaces the current
// executable.
func selfReplace(ctx context.Context, latest string) error {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	return clog.Spinner("Updating").
		Str("version", latest).
		Elapsed("elapsed").
		Progress(ctx, func(ctx context.Context, update *clog.Update) error {
			return doSelfReplace(ctx, latest, update)
		}).
		Msg("Updated")
}

func doSelfReplace(ctx context.Context, latest string, update *clog.Update) error {
	asset := AssetName(latest, runtime.GOOS, runtime.GOARCH)

	update.Msg("Fetching release").Send()

	assets, err := fetchReleaseAssets(ctx, "v"+latest)
	if err != nil {
		return err
	}

	// Find the archive and checksums asset IDs.
	var archiveID, checksumID int
	for _, a := range assets {
		switch a.Name {
		case asset:
			archiveID = a.ID
		case "checksums.txt":
			checksumID = a.ID
		}
	}

	if archiveID == 0 {
		return fmt.Errorf("asset %s not found in release v%s", asset, latest)
	}
	if checksumID == 0 {
		return fmt.Errorf("checksums.txt not found in release v%s", latest)
	}

	update.Msg("Downloading").Str("asset", asset).Send()

	archiveData, err := downloadAsset(ctx, archiveID)
	if err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}

	update.Msg("Verifying checksum").Send()

	checksumData, err := downloadAsset(ctx, checksumID)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	// Find the matching checksum line.
	var matched string
	for line := range strings.SplitSeq(string(checksumData), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), asset) {
			matched = strings.TrimSpace(line)
			break
		}
	}

	if matched == "" {
		return fmt.Errorf("no checksum found for %s", asset)
	}

	if !VerifyChecksum(archiveData, matched, asset) {
		return errors.New("checksum verification failed")
	}

	update.Msg("Installing").Send()

	// Extract the binary from the archive.
	binName := binaryName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	var binData []byte
	if runtime.GOOS == "windows" {
		binData, err = extractFromZip(archiveData, binName)
	} else {
		binData, err = extractFromTarGz(archiveData, binName)
	}

	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	return AtomicReplace(exe, binData)
}

// httpGet performs an authenticated HTTP GET via the GitHub API.
// Strips the Authorization header on cross-host redirects (GitHub
// redirects to CDN). Response capped at 100 MiB.
func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	setGitHubAuth(req)

	client := &http.Client{
		CheckRedirect: stripAuthOnRedirect,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxBinarySize))
}

// extractFromTarGz finds and returns the named file from a tar.gz
// archive.
func extractFromTarGz(data []byte, name string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("opening gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("reading tar entry: %w", err)
		}

		// Match on the base name to handle archives that include a
		// directory prefix.
		if filepath.Base(hdr.Name) == name && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(io.LimitReader(tr, maxBinarySize))
		}
	}

	return nil, fmt.Errorf("%s not found in archive", name)
}

// extractFromZip finds and returns the named file from a zip archive.
func extractFromZip(data []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening zip reader: %w", err)
	}

	for _, f := range zr.File {
		if filepath.Base(f.Name) == name {
			return readZipFile(f, name)
		}
	}

	return nil, fmt.Errorf("%s not found in archive", name)
}

func readZipFile(f *zip.File, name string) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("opening %s in zip: %w", name, err)
	}
	defer func() { _ = rc.Close() }()

	return io.ReadAll(io.LimitReader(rc, maxBinarySize))
}

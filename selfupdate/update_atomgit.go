package selfupdate

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/blang/semver"
)

// rewriteAtomGitDownloadURL converts the browser_download_url returned by
// AtomGit into the actual API download endpoint that returns a 302 to CDN.
//
//	in:  https://api.atomgit.com/{owner}/{repo}/releases/download/{tag}/{file}
//	out: https://api.atomgit.com/api/v5/repos/{owner}/{repo}/releases/{tag}/attach_files/{file}/download
var reAtomGitURL = regexp.MustCompile(`^(https?://[^/]+)/([^/]+)/([^/]+)/releases/download/([^/]+)/(.+)$`)

func rewriteAtomGitDownloadURL(rawURL string) string {
	m := reAtomGitURL.FindStringSubmatch(rawURL)
	if m == nil {
		return rawURL
	}
	return fmt.Sprintf("%s/api/v5/repos/%s/%s/releases/%s/attach_files/%s/download", m[1], m[2], m[3], m[4], m[5])
}

func (up *AtomGitUpdater) downloadDirectlyFromURL(assetURL string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", assetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create HTTP request to %s: %s", assetURL, err)
	}

	req.Header.Add("Accept", "application/octet-stream")
	req = req.WithContext(up.apiCtx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to download a release file from %s: %s", assetURL, err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to download a release file from %s: Not successful status %d", assetURL, res.StatusCode)
	}

	return res.Body, nil
}

func (up *AtomGitUpdater) UpdateTo(rel *Release, cmdPath string) error {
	src, err := up.downloadDirectlyFromURL(rel.AssetURL)
	if err != nil {
		return err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("Failed reading asset body: %v", err)
	}

	return uncompressAndUpdate(bytes.NewReader(data), rel.AssetName, cmdPath, up.binaryName)
}

func (up *AtomGitUpdater) UpdateCommand(cmdPath string, current semver.Version, slug string) (*Release, error) {
	if runtime.GOOS == "windows" && !strings.HasSuffix(cmdPath, ".exe") {
		cmdPath = cmdPath + ".exe"
	}

	stat, err := os.Lstat(cmdPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to stat '%s'. File may not exist: %s", cmdPath, err)
	}
	if stat.Mode()&os.ModeSymlink != 0 {
		p, err := filepath.EvalSymlinks(cmdPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to resolve symlink '%s' for executable: %s", cmdPath, err)
		}
		cmdPath = p
	}

	rel, ok, err := up.DetectLatest(slug)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Println("No release detected. Current version is considered up-to-date")
		return &Release{Version: current}, nil
	}
	if current.Equals(rel.Version) {
		log.Println("Current version", current, "is the latest. Update is not needed")
		return rel, nil
	}
	log.Println("Will update", cmdPath, "to the latest version", rel.Version)
	if err := up.UpdateTo(rel, cmdPath); err != nil {
		return nil, err
	}
	return rel, nil
}

func (up *AtomGitUpdater) UpdateSelf(current semver.Version, slug string) (*Release, error) {
	cmdPath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return up.UpdateCommand(cmdPath, current, slug)
}

func UpdateToAtomGit(assetURL, cmdPath string) error {
	up := DefaultAtomGitUpdater()
	src, err := up.downloadDirectlyFromURL(assetURL)
	if err != nil {
		return err
	}
	defer src.Close()
	return uncompressAndUpdate(src, assetURL, cmdPath, up.binaryName)
}

func UpdateCommandAtomGit(cmdPath string, current semver.Version, slug string) (*Release, error) {
	return DefaultAtomGitUpdater().UpdateCommand(cmdPath, current, slug)
}

func UpdateSelfAtomGit(current semver.Version, slug string) (*Release, error) {
	return DefaultAtomGitUpdater().UpdateSelf(current, slug)
}

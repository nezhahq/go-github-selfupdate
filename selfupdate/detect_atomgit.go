package selfupdate

import (
	"context"
	"fmt"
	"strings"

	"gitee.com/naibahq/go-gitee/gitee"
)

func (up *AtomGitUpdater) DetectLatest(slug string) (release *Release, found bool, err error) {
	return up.DetectVersion(slug, "")
}

func (up *AtomGitUpdater) DetectVersion(slug string, version string) (release *Release, found bool, err error) {
	repo := strings.Split(slug, "/")
	if len(repo) != 2 || repo[0] == "" || repo[1] == "" {
		return nil, false, fmt.Errorf("Invalid slug format. It should be 'owner/name': %s", slug)
	}

	rels, res, err := up.api.RepositoriesApi.GetV5ReposOwnerRepoReleases(context.Background(), repo[0], repo[1], &gitee.GetV5ReposOwnerRepoReleasesOpts{})
	if err != nil {
		log.Println("API returned an error response:", err)
		if res != nil && res.StatusCode == 404 {
			err = nil
			log.Println("API returned 404. Repository or release not found")
		}
		return nil, false, err
	}

	rel, asset, ver, found := findReleaseAndAssetGitee(rels, version, up.filters)
	if !found {
		return nil, false, nil
	}

	url := rewriteAtomGitDownloadURL(asset.BrowserDownloadUrl)
	log.Println("Successfully fetched the latest release. tag:", rel.TagName, ", name:", rel.Name, ", Asset:", url)

	publishedAt := rel.CreatedAt
	release = &Release{
		Version:           ver,
		AssetURL:          url,
		AssetByteSize:     -1,
		AssetID:           -1,
		ValidationAssetID: -1,
		URL:               "",
		ReleaseNotes:      rel.Body,
		Name:              rel.Name,
		PublishedAt:       &publishedAt,
		RepoOwner:         repo[0],
		RepoName:          repo[1],
		AssetName:         asset.Name,
	}

	if up.validator != nil {
		validationName := asset.Name + up.validator.Suffix()
		_, ok := findValidationAssetGitee(rel, validationName)
		if !ok {
			return nil, false, fmt.Errorf("Failed finding validation file %q", validationName)
		}
	}

	return release, true, nil
}

func DetectLatestAtomGit(slug string) (*Release, bool, error) {
	return DefaultAtomGitUpdater().DetectLatest(slug)
}

func DetectVersionAtomGit(slug string, version string) (*Release, bool, error) {
	return DefaultAtomGitUpdater().DetectVersion(slug, version)
}

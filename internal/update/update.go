package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	forgejoAPI = "https://git.laoker.cc/api/v1/repos/Laoke/EngineTools/releases/latest"
	githubAPI  = "https://api.github.com/repos/LaokeQwQ/EngineTools/releases/latest"
	timeout    = 5 * time.Second
)

type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

// CheckUpdate checks for updates from Forgejo first, falls back to GitHub if Forgejo fails.
func CheckUpdate(currentVersion string) (*Release, bool, error) {
	// Try Forgejo first
	release, err := fetchRelease(forgejoAPI)
	if err == nil {
		hasUpdate := release.TagName != currentVersion && release.TagName != "v"+currentVersion
		return release, hasUpdate, nil
	}

	// Fallback to GitHub
	release, err = fetchRelease(githubAPI)
	if err != nil {
		return nil, false, fmt.Errorf("both Forgejo and GitHub failed: %w", err)
	}

	hasUpdate := release.TagName != currentVersion && release.TagName != "v"+currentVersion
	return release, hasUpdate, nil
}

func fetchRelease(apiURL string) (*Release, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

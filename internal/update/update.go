package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

func CheckUpdate(currentVersion string) (*Release, bool, error) {
	release, err := fetchRelease(forgejoAPI)
	if err == nil {
		hasUpdate := isNewer(release.TagName, currentVersion)
		return release, hasUpdate, nil
	}

	release, err = fetchRelease(githubAPI)
	if err != nil {
		return nil, false, fmt.Errorf("both Forgejo and GitHub failed: %w", err)
	}

	hasUpdate := isNewer(release.TagName, currentVersion)
	return release, hasUpdate, nil
}

func isNewer(remote, local string) bool {
	rv := parseVersion(remote)
	lv := parseVersion(local)
	for i := 0; i < 3; i++ {
		if rv[i] > lv[i] {
			return true
		}
		if rv[i] < lv[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n, _ := strconv.Atoi(parts[i])
		result[i] = n
	}
	return result
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

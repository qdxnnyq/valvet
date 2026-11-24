package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// appVersion is only the update counter, so older builds can still see a newer one.
// appLabel is the version people see. A small change adds 0.0.1. A big change bumps the minor or major.
const appVersion = 199
const appLabel = "1.9.9"

const updateMetaURL = "https://raw.githubusercontent.com/qdxnnyq/valvet/main/version.json"

type updateOffer struct {
	Version int    `json:"version"`
	Name    string `json:"name,omitempty"`
	Label   string `json:"label,omitempty"`
	Notes   string `json:"notes"`
	URL     string `json:"url"`
	Current int    `json:"current"`
	Update  bool   `json:"update"`
	Error   string `json:"error,omitempty"`
}

func updateClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func allowedUpdateHost(host string) bool {
	host = strings.ToLower(host)
	switch host {
	case "github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com", "github-releases.githubusercontent.com":
		return true
	default:
		return strings.HasSuffix(host, ".githubusercontent.com")
	}
}

func fetchUpdate(ctx context.Context, fresh bool) (updateOffer, error) {
	fileCh := make(chan struct {
		offer updateOffer
		err   error
	}, 1)
	relCh := make(chan struct {
		ver        int
		url, label string
		err        error
	}, 1)
	go func() {
		offer, err := fetchVersionFile(ctx)
		fileCh <- struct {
			offer updateOffer
			err   error
		}{offer, err}
	}()
	go func() {
		ver, url, label, err := latestRelease(ctx, fresh)
		relCh <- struct {
			ver        int
			url, label string
			err        error
		}{ver, url, label, err}
	}()
	fileRes := <-fileCh
	relRes := <-relCh

	offer := updateOffer{Current: appVersion, Label: appLabel}
	if fileRes.err == nil && fileRes.offer.Version > 0 {
		offer.Version = fileRes.offer.Version
		offer.Name = fileRes.offer.Name
		offer.Notes = fileRes.offer.Notes
		offer.URL = fileRes.offer.URL
	}
	if relRes.err == nil && (relRes.ver > offer.Version || labelNewer(relRes.label, offer.Name)) {
		staleNotes := fileRes.err != nil || fileRes.offer.Version != relRes.ver
		if relRes.ver > 0 {
			offer.Version = relRes.ver
		}
		if relRes.label != "" {
			offer.Name = relRes.label
		}
		if relRes.url != "" {
			offer.URL = relRes.url
		}
		if staleNotes {
			offer.Notes = "A newer Valvet is on GitHub."
		}
	}
	if fileRes.err != nil && relRes.err != nil {
		return updateOffer{Current: appVersion, Label: appLabel}, fileRes.err
	}
	offer.Current = appVersion
	offer.Label = appLabel
	if offer.URL == "" && offer.Version > 0 {
		offer.URL = fmt.Sprintf("https://github.com/qdxnnyq/valvet/releases/download/v%d/Valvet.exe", offer.Version)
	}
	offer.Update = false
	newer := offer.Version > appVersion || labelNewer(offer.Name, appLabel)
	if newer {
		if parsed, err := http.NewRequest(http.MethodGet, offer.URL, nil); err == nil && allowedUpdateHost(parsed.URL.Hostname()) {
			offer.Update = true
		} else {
			offer.URL = ""
		}
	}
	return offer, nil
}

func fetchVersionFile(ctx context.Context) (updateOffer, error) {
	offer := updateOffer{Current: appVersion}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateMetaURL+"?t="+fmt.Sprint(time.Now().UnixNano()), nil)
	if err != nil {
		return offer, err
	}
	req.Header.Set("User-Agent", "Valvet")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	res, err := updateClient().Do(req)
	if err != nil {
		return offer, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return offer, fmt.Errorf("status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return offer, err
	}
	if json.Unmarshal(body, &offer) != nil || offer.Version <= 0 {
		return updateOffer{Current: appVersion}, errors.New("bad version file")
	}
	offer.Current = appVersion
	offer.Label = appLabel
	return offer, nil
}

var releaseMu sync.Mutex
var releaseAt time.Time
var releaseVer int
var releaseURL string
var releaseLabel string

func labelNewer(remote, local string) bool {
	parse := func(s string) (int, int, int, bool) {
		s = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), "v")
		var a, b, c int
		if _, err := fmt.Sscanf(s, "%d.%d.%d", &a, &b, &c); err != nil {
			return 0, 0, 0, false
		}
		return a, b, c, true
	}
	ra, rb, rc, rok := parse(remote)
	la, lb, lc, lok := parse(local)
	if !rok || !lok {
		return false
	}
	if ra != la {
		return ra > la
	}
	if rb != lb {
		return rb > lb
	}
	return rc > lc
}

func parseReleaseTag(tag string) (int, string) {
	tag = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tag)), "v")
	if n, err := strconv.Atoi(tag); err == nil {
		return n, ""
	}
	var a, b, c int
	if _, err := fmt.Sscanf(tag, "%d.%d.%d", &a, &b, &c); err == nil {
		return 0, fmt.Sprintf("%d.%d.%d", a, b, c)
	}
	return 0, ""
}

func latestOne(ctx context.Context) (int, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/qdxnnyq/valvet/releases/latest", nil)
	if err != nil {
		return 0, "", "", err
	}
	req.Header.Set("User-Agent", "Valvet")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Cache-Control", "no-cache")
	res, err := updateClient().Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return 0, "", "", err
	}
	if res.StatusCode >= 400 {
		return 0, "", "", fmt.Errorf("status %d", res.StatusCode)
	}
	var rel struct {
		Tag    string `json:"tag_name"`
		Name   string `json:"name"`
		Draft  bool   `json:"draft"`
		Pre    bool   `json:"prerelease"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.Unmarshal(body, &rel) != nil {
		return 0, "", "", errors.New("bad release")
	}
	if rel.Draft || rel.Pre {
		return 0, "", "", errors.New("no release")
	}
	tagVer, tagLabel := parseReleaseTag(rel.Tag)
	asset := ""
	for _, item := range rel.Assets {
		if strings.EqualFold(item.Name, "Valvet.exe") || strings.EqualFold(item.Name, "Velvet.exe") {
			if asset == "" || strings.EqualFold(item.Name, "Valvet.exe") {
				asset = item.URL
			}
		}
	}
	if asset == "" {
		return 0, "", "", errors.New("no build")
	}
	label := tagLabel
	if label == "" {
		label = semverName(rel.Name)
	}
	if tagVer <= 0 && label == "" {
		return 0, "", "", errors.New("no release")
	}
	return tagVer, asset, label, nil
}

func latestRelease(ctx context.Context, fresh bool) (int, string, string, error) {
	releaseMu.Lock()
	if !fresh && !releaseAt.IsZero() && time.Since(releaseAt) < 45*time.Second {
		ver, url, label := releaseVer, releaseURL, releaseLabel
		releaseMu.Unlock()
		if ver <= 0 && label == "" {
			return 0, "", "", errors.New("no release")
		}
		return ver, url, label, nil
	}
	releaseMu.Unlock()
	if ver, url, label, err := latestOne(ctx); err == nil && (ver > 0 || label != "") {
		releaseMu.Lock()
		releaseAt = time.Now()
		releaseVer = ver
		releaseURL = url
		releaseLabel = label
		releaseMu.Unlock()
		return ver, url, label, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/qdxnnyq/valvet/releases?per_page=30", nil)
	if err != nil {
		return 0, "", "", err
	}
	req.Header.Set("User-Agent", "Valvet")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Cache-Control", "no-cache")
	res, err := updateClient().Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return 0, "", "", err
	}
	if res.StatusCode >= 400 {
		return 0, "", "", fmt.Errorf("status %d", res.StatusCode)
	}
	var payload []struct {
		Tag    string `json:"tag_name"`
		Name   string `json:"name"`
		Draft  bool   `json:"draft"`
		Pre    bool   `json:"prerelease"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return 0, "", "", errors.New("bad release")
	}
	ver, url, label := 0, "", ""
	for _, rel := range payload {
		if rel.Draft || rel.Pre {
			continue
		}
		tagVer, tagLabel := parseReleaseTag(rel.Tag)
		if tagVer <= 0 || tagVer < ver {
			continue
		}
		asset := ""
		for _, item := range rel.Assets {
			if strings.EqualFold(item.Name, "Valvet.exe") || strings.EqualFold(item.Name, "Velvet.exe") {
				if asset == "" || strings.EqualFold(item.Name, "Valvet.exe") {
					asset = item.URL
				}
			}
		}
		if asset == "" {
			continue
		}
		nextLabel := tagLabel
		if nextLabel == "" {
			nextLabel = semverName(rel.Name)
		}
		if tagVer > ver || label == "" {
			ver, url, label = tagVer, asset, nextLabel
		}
	}
	releaseMu.Lock()
	releaseAt = time.Now()
	releaseVer = ver
	releaseURL = url
	releaseLabel = label
	releaseMu.Unlock()
	if ver <= 0 {
		return 0, "", "", errors.New("no release")
	}
	return ver, url, label, nil
}

func semverName(raw string) string {
	for _, part := range strings.Fields(raw) {
		part = strings.Trim(strings.TrimPrefix(strings.ToLower(part), "v"), ".,")
		var a, b, c int
		if _, err := fmt.Sscanf(part, "%d.%d.%d", &a, &b, &c); err == nil {
			return fmt.Sprintf("%d.%d.%d", a, b, c)
		}
	}
	return ""
}

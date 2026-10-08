package seo

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"investo/internal/repository"
)

// IndexNowService auto-submits new URLs to IndexNow search engines
// (Bing, Yandex, Seznam, Naver). Submitted URLs are cached to avoid re-submitting.
type IndexNowService struct {
	SettingRepo *repository.SettingRepository
	Host        string // e.g. "investo.whitelabel.co.id" (no scheme)
	client      *http.Client
}

func NewIndexNowService(settingRepo *repository.SettingRepository, host string) *IndexNowService {
	return &IndexNowService{
		SettingRepo: settingRepo,
		Host:        strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://"),
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

// Key returns the persisted IndexNow key, generating one on first use.
func (s *IndexNowService) Key() string {
	if s.SettingRepo == nil {
		return ""
	}
	key, _ := s.SettingRepo.Get("indexnow_key")
	if key == "" {
		key = generateKey()
		_ = s.SettingRepo.Set("indexnow_key", key)
	}
	return key
}

func generateKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Submit sends the given URLs to the IndexNow API for the configured host.
// It deduplicates against the local submitted-cache.
func (s *IndexNowService) Submit(urls []string) (int, error) {
	if s.Host == "" || len(urls) == 0 {
		return 0, nil
	}

	key := s.Key()
	if key == "" {
		return 0, fmt.Errorf("indexnow: no key available")
	}

	// Deduplicate against previously submitted URLs.
	cache := s.loadCache()
	var pending []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || cache[u] {
			continue
		}
		// Normalize to absolute URL on our host.
		if strings.HasPrefix(u, "/") {
			u = "https://" + s.Host + u
		}
		pending = append(pending, u)
		cache[u] = true
	}
	if len(pending) == 0 {
		return 0, nil
	}

	payload := map[string]interface{}{
		"host":        s.Host,
		"key":         key,
		"keyLocation": "https://" + s.Host + "/indexnow-key.txt",
		"urlList":     pending,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.indexnow.org/indexnow", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("indexnow: submit: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return 0, fmt.Errorf("indexnow: unexpected status %d", resp.StatusCode)
	}

	s.saveCache(cache)
	return len(pending), nil
}

func (s *IndexNowService) loadCache() map[string]bool {
	cache := map[string]bool{}
	if s.SettingRepo == nil {
		return cache
	}
	raw, _ := s.SettingRepo.Get("indexnow_submitted")
	_ = json.Unmarshal([]byte(raw), &cache)
	return cache
}

func (s *IndexNowService) saveCache(cache map[string]bool) {
	if s.SettingRepo == nil {
		return
	}
	// Cap the cache to avoid unbounded growth.
	if len(cache) > 10000 {
		for k := range cache {
			delete(cache, k)
			if len(cache) <= 5000 {
				break
			}
		}
	}
	raw, _ := json.Marshal(cache)
	_ = s.SettingRepo.Set("indexnow_submitted", string(raw))
}

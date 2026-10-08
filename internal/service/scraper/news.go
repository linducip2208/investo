package scraper

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"investo/internal/model"
)

type NewsScraper struct{}

type rssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Source      string `xml:"source"`
}

func slugify(title string) string {
	title = strings.ToLower(title)
	re := regexp.MustCompile(`[^a-z0-9\s-]`)
	title = re.ReplaceAllString(title, "")
	re = regexp.MustCompile(`\s+`)
	title = re.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-")
	re = regexp.MustCompile(`-+`)
	title = re.ReplaceAllString(title, "-")

	if len(title) > 480 {
		title = title[:480]
		title = strings.TrimRight(title, "-")
	}

	return title
}

func (s *NewsScraper) FetchMarketNews(limit int) ([]model.News, error) {
	if limit <= 0 {
		limit = 20
	}

	url := fmt.Sprintf("https://feeds.finance.yahoo.com/rss/2.0/headline?s=%%5EJKSE&region=ID&lang=en-ID")

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("news scraper: create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("news scraper: fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("news scraper: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("news scraper: read body: %w", err)
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("news scraper: parse XML: %w", err)
	}

	sourceName := "Yahoo Finance"
	if feed.Channel.Title != "" {
		sourceName = feed.Channel.Title
	}

	var news []model.News
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta == nil {
		jakarta = time.FixedZone("WIB", 7*3600)
	}

	dateFormats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 MST",
	}

	for i, item := range feed.Channel.Items {
		if i >= limit {
			break
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}

		slug := slugify(title)
		content := strings.TrimSpace(item.Description)
		sourceURL := strings.TrimSpace(item.Link)
		itemSource := strings.TrimSpace(item.Source)
		if itemSource == "" {
			itemSource = sourceName
		}

		var publishedAt time.Time
		for _, format := range dateFormats {
			if t, err := time.Parse(format, strings.TrimSpace(item.PubDate)); err == nil {
				publishedAt = t.In(jakarta)
				break
			}
		}
		if publishedAt.IsZero() {
			publishedAt = time.Now().In(jakarta)
		}

		news = append(news, model.News{
			Title:          title,
			Slug:           slug,
			Content:        content,
			Source:         itemSource,
			SourceURL:      sourceURL,
			ImageURL:       "",
			PublishedAt:    publishedAt,
			SentimentScore: 0,
		})
	}

	return news, nil
}

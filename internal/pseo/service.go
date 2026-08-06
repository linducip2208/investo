package pseo

import (
	"fmt"
	"investo/internal/model"
	"investo/internal/repository"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	StockRepo  *repository.StockRepository
	SectorRepo *repository.SectorRepository
	BlogRepo   *repository.BlogRepository
	NewsRepo   *repository.NewsRepository
	AppURL     string
}

type URLSet struct {
	XMLName string `xml:"urlset"`
	URLs    []URL  `xml:"url"`
}

type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func (s *Service) GenerateSitemap() URLSet {
	now := time.Now().Format("2006-01-02")

	urls := []URL{
		{Loc: s.AppURL + "/", LastMod: now, ChangeFreq: "daily", Priority: "1.0"},
		{Loc: s.AppURL + "/saham", LastMod: now, ChangeFreq: "daily", Priority: "0.9"},
		{Loc: s.AppURL + "/forex", LastMod: now, ChangeFreq: "daily", Priority: "0.9"},
		{Loc: s.AppURL + "/screener", LastMod: now, ChangeFreq: "daily", Priority: "0.8"},
		{Loc: s.AppURL + "/berita", LastMod: now, ChangeFreq: "daily", Priority: "0.8"},
		{Loc: s.AppURL + "/blog", LastMod: now, ChangeFreq: "weekly", Priority: "0.7"},
		{Loc: s.AppURL + "/docs", LastMod: now, ChangeFreq: "monthly", Priority: "0.6"},
		{Loc: s.AppURL + "/faq", LastMod: now, ChangeFreq: "monthly", Priority: "0.5"},
		{Loc: s.AppURL + "/kontak", LastMod: now, ChangeFreq: "monthly", Priority: "0.5"},
		{Loc: s.AppURL + "/beli-aplikasi-saham", LastMod: now, ChangeFreq: "monthly", Priority: "0.7"},
	}

	stocks, err := s.StockRepo.ListActive()
	if err == nil {
		for _, stock := range stocks {
			urls = append(urls, URL{
				Loc:        s.AppURL + "/saham/" + stock.Code,
				LastMod:    now,
				ChangeFreq: "daily",
				Priority:   "0.8",
			})
		}
	}

	sectors, err := s.SectorRepo.FindAll()
	if err == nil {
		for _, sector := range sectors {
			urls = append(urls, URL{
				Loc:        s.AppURL + "/best-saham-" + sector.Slug,
				LastMod:    now,
				ChangeFreq: "weekly",
				Priority:   "0.7",
			})

			currentYear := time.Now().Year()
			urls = append(urls, URL{
				Loc:        s.AppURL + "/best-saham-" + sector.Slug + "-" + strconv.Itoa(currentYear),
				LastMod:    now,
				ChangeFreq: "monthly",
				Priority:   "0.6",
			})

			for _, stock := range stocks {
				if stock.SectorID == sector.ID {
					urls = append(urls, URL{
						Loc:        s.AppURL + "/alternatives-to-" + stock.Code,
						LastMod:    now,
						ChangeFreq: "weekly",
						Priority:   "0.6",
					})
				}
			}
		}
	}

	if len(stocks) > 1 {
		for i := 0; i < len(stocks) && i < 20; i++ {
			for j := i + 1; j < len(stocks) && j < 20; j++ {
				urls = append(urls, URL{
					Loc:        fmt.Sprintf("%s/compare/%s-vs-%s", s.AppURL, stocks[i].Code, stocks[j].Code),
					LastMod:    now,
					ChangeFreq: "weekly",
					Priority:   "0.6",
				})
			}
		}
	}

	posts, _, err := s.BlogRepo.ListPublished(0, 500, 0)
	if err == nil {
		for _, post := range posts {
			urls = append(urls, URL{
				Loc:        s.AppURL + "/blog/" + post.Slug,
				LastMod:    post.UpdatedAt.Format("2006-01-02"),
				ChangeFreq: "monthly",
				Priority:   "0.6",
			})
		}
	}

	return URLSet{URLs: urls}
}

func (s *Service) GenerateRobotsTxt() string {
	return strings.Join([]string{
		"User-agent: *",
		"Allow: /$",
		"Allow: /saham",
		"Allow: /forex",
		"Allow: /screener",
		"Allow: /berita",
		"Allow: /blog",
		"Allow: /docs",
		"Allow: /faq",
		"Allow: /kontak",
		"Allow: /best-",
		"Allow: /alternatives-to-",
		"Allow: /compare/",
		"Allow: /beli-aplikasi-saham",
		"Allow: /sitemap.xml",
		"Disallow: /admin",
		"Disallow: /api",
		"Disallow: /dashboard",
		"Disallow: /webhooks",
		fmt.Sprintf("Sitemap: %s/sitemap.xml", s.AppURL),
	}, "\n")
}

func (s *Service) GenerateJSONLDStock(stock *model.Stock, price float64) string {
	return fmt.Sprintf(`{
  "@context": "https://schema.org",
  "@type": "Corporation",
  "name": "%s",
  "tickerSymbol": "%s",
  "url": "%s/saham/%s",
  "description": "%s"
}`, strings.ReplaceAll(stock.Name, "\"", "\\\""), stock.Code, s.AppURL, stock.Code, strings.ReplaceAll(stock.Description, "\"", "\\\""))
}

func (s *Service) GenerateJSONLDBlogPost(post *model.BlogPost, authorName string) string {
	return fmt.Sprintf(`{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "%s",
  "description": "%s",
  "author": {
    "@type": "Person",
    "name": "%s"
  },
  "datePublished": "%s",
  "dateModified": "%s",
  "url": "%s/blog/%s"
}`, strings.ReplaceAll(post.Title, "\"", "\\\""), strings.ReplaceAll(post.Excerpt, "\"", "\\\""), authorName, post.PublishedAt.Format("2006-01-02"), post.UpdatedAt.Format("2006-01-02"), s.AppURL, post.Slug)
}

func (s *Service) GenerateJSONLDCompare(stockA, stockB *model.Stock) string {
	return fmt.Sprintf(`{
  "@context": "https://schema.org",
  "@type": "WebPage",
  "name": "Perbandingan %s (%s) vs %s (%s)",
  "description": "Bandingkan saham %s dengan %s secara head-to-head: harga, fundamental, performa.",
  "url": "%s/compare/%s-vs-%s"
}`, stockA.Name, stockA.Code, stockB.Name, stockB.Code, stockA.Code, stockB.Code, s.AppURL, stockA.Code, stockB.Code)
}

func (s *Service) GenerateJSONLDSector(sector *model.Sector, year int) string {
	return fmt.Sprintf(`{
  "@context": "https://schema.org",
  "@type": "ItemList",
  "name": "Saham Terbaik Sektor %s %d",
  "description": "Daftar saham terbaik di sektor %s tahun %d berdasarkan fundamental dan performa.",
  "url": "%s/best-saham-%s-%d"
}`, sector.Name, year, sector.Name, year, s.AppURL, sector.Slug, year)
}

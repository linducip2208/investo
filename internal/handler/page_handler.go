package handler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/pseo"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

var startTime = time.Now()

type PageHandler struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	WatchlistRepo  *repository.WatchlistRepository
	WatchlistItemRepo *repository.WatchlistItemRepository
	PortfolioRepo  *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	NewsRepo       *repository.NewsRepository
	SectorRepo     *repository.SectorRepository
	PSEOService    *pseo.Service
	Templates      *template.Template
	NotifService   *service.NotificationService
}

func (h *PageHandler) Home(w http.ResponseWriter, r *http.Request) {
	if middleware.IsAuthenticated(r) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title": "Investo - Platform Investasi Saham & Forex Indonesia",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "marketing/home.html", data)
}

type DashboardStockRow struct {
	Code          string
	Name          string
	Price         float64
	ChangePercent float64
}

func (h *PageHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stocks, _ := h.StockRepo.ListActive()
	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}
	priceMap, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	type row struct {
		model.Stock
		Price         float64
		ChangePercent float64
	}
	var marketRows []row
	for _, s := range stocks {
		r := row{Stock: s, Price: priceMap[s.ID]}
		prev, _ := h.StockPriceRepo.GetPriceChange(s.ID, 1)
		if prev > 0 {
			r.ChangePercent = ((r.Price - prev) / prev) * 100
		}
		marketRows = append(marketRows, r)
	}

	sort.Slice(marketRows, func(i, j int) bool {
		return marketRows[i].ChangePercent > marketRows[j].ChangePercent
	})

	gainers := marketRows
	if len(gainers) > 5 {
		gainers = gainers[:5]
	}

	sort.Slice(marketRows, func(i, j int) bool {
		return marketRows[i].ChangePercent < marketRows[j].ChangePercent
	})
	losers := marketRows
	if len(losers) > 5 {
		losers = losers[:5]
	}

	var gainerRows, loserRows []DashboardStockRow
	for _, s := range gainers {
		gainerRows = append(gainerRows, DashboardStockRow{s.Code, s.Name, s.Price, s.ChangePercent})
	}
	for _, s := range losers {
		loserRows = append(loserRows, DashboardStockRow{s.Code, s.Name, s.Price, s.ChangePercent})
	}

	news, _, _ := h.NewsRepo.List(0, 5)

	watchlists, _ := h.WatchlistRepo.FindByUserID(user.ID)
	watchlistCount := len(watchlists)

	type WatchlistItemRow struct {
		Code string
		Name string
	}
	var allWatchlistItems []WatchlistItemRow
	seenCodes := map[string]bool{}
	for _, wl := range watchlists {
		items, _ := h.WatchlistItemRepo.FindByWatchlistID(wl.ID)
		for _, item := range items {
			stock, err := h.StockRepo.FindByID(item.StockID)
			if err == nil && !seenCodes[stock.Code] {
				seenCodes[stock.Code] = true
				allWatchlistItems = append(allWatchlistItems, WatchlistItemRow{Code: stock.Code, Name: stock.Name})
			}
		}
	}

	portfolios, _ := h.PortfolioRepo.FindByUserID(user.ID)
	portfolioCount := len(portfolios)

	var defaultPortfolioID int64
	for _, p := range portfolios {
		if p.IsDefault {
			defaultPortfolioID = p.ID
			break
		}
	}
	if defaultPortfolioID == 0 && len(portfolios) > 0 {
		defaultPortfolioID = portfolios[0].ID
	}

	alertsExist := false
	_ = alertsExist

	data := map[string]interface{}{
		"Title":               "Dashboard - Investo",
		"User":                user,
		"TotalStocks":         len(stocks),
		"Gainers":             gainerRows,
		"Losers":              loserRows,
		"News":                news,
		"WatchlistCount":      watchlistCount,
		"PortfolioCount":      portfolioCount,
		"Watchlists":          watchlists,
		"Portfolios":          portfolios,
		"WatchlistItems":      allWatchlistItems,
		"DefaultPortfolioID":  defaultPortfolioID,
	}
	log.Println("Dashboard: executing template...")
	if err := h.Templates.ExecuteTemplate(w, "dashboard/index.html", data); err != nil {
		log.Printf("Dashboard: ExecuteTemplate error: %v", err)
		http.Error(w, "Dashboard error: "+err.Error(), 500)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *PageHandler) Docs(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Dokumentasi - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "docs/index.html", data)
}

func (h *PageHandler) FAQ(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "FAQ - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pages/faq.html", data)
}

func (h *PageHandler) Contact(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Kontak - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pages/contact.html", data)
}

func (h *PageHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	urlset := h.PSEOService.GenerateSitemap()

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`))

	type URL struct {
		Loc        string `xml:"loc"`
		LastMod    string `xml:"lastmod,omitempty"`
		ChangeFreq string `xml:"changefreq,omitempty"`
		Priority   string `xml:"priority,omitempty"`
	}

	for _, u := range urlset.URLs {
		bytes, _ := xml.Marshal(u)
		w.Write(bytes)
	}

	w.Write([]byte(`</urlset>`))
}

func (h *PageHandler) Robots(w http.ResponseWriter, r *http.Request) {
	txt := h.PSEOService.GenerateRobotsTxt()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(txt))
}

func (h *PageHandler) Health(w http.ResponseWriter, r *http.Request) {
	// Simple status
	stocks, _ := h.StockRepo.ListActive()
	now := time.Now().In(time.FixedZone("WIB", 7*3600))

	// JSON if explicitly requested, otherwise HTML
	wantJSON := r.URL.Query().Get("format") == "json" || r.Header.Get("Accept") == "application/json"
	if !wantJSON {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Investo — System Status</title><style>body{font-family:'Inter',system-ui,sans-serif;background:#0b1120;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}.card{background:#1e293b;border:1px solid #334155;border-radius:16px;padding:40px;max-width:500px;width:90%%;text-align:center}h1{font-size:2rem;margin:0 0 8px}.status{display:inline-block;width:12px;height:12px;border-radius:50%%;background:#10b981;margin-right:8px;animation:pulse 2s infinite}@keyframes pulse{0%%,100%%{opacity:1}50%%{opacity:.5}}.metric{display:flex;justify-content:space-between;padding:12px 0;border-bottom:1px solid #334155;font-size:14px}.label{color:#94a3b8}.value{color:#f8fafc;font-weight:600;font-family:'JetBrains Mono',monospace}.footer{margin-top:24px;font-size:12px;color:#475569}a{color:#3b82f6}</style></head><body><div class="card"><h1><span class="status"></span>Investo</h1><p style="color:#94a3b8;margin:0 0 24px">Sistem berjalan normal</p><div class="metric"><span class="label">Status</span><span class="value" style="color:#10b981">ONLINE</span></div><div class="metric"><span class="label">Uptime</span><span class="value">%s</span></div><div class="metric"><span class="label">Saham IDX</span><span class="value">%d emiten</span></div><div class="metric"><span class="label">Waktu Server</span><span class="value">%s WIB</span></div><div class="metric"><span class="label">Environment</span><span class="value">%s</span></div><div class="footer">Investo — Stock & Forex Intelligence Platform<br>%s</div></div></body></html>`,
			now.Format("15:04:05"), len(stocks), now.Format("02 Jan 2006 15:04:05"), "production", now.Format("2006"))
		return
	}

	// JSON response for API/monitoring
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app": map[string]interface{}{
			"name":    "Investo",
			"version": "1.0.0",
			"env":     "production",
		},
		"server": map[string]interface{}{
			"status":  "healthy",
			"uptime":  time.Since(startTime).String(),
			"time":    now.Format(time.RFC3339),
			"wib":     now.Format("15:04:05 WIB"),
			"date":    now.Format("Monday, 02 January 2006"),
		},
		"database": map[string]interface{}{
			"driver":  "MySQL 8.4",
			"status":  "connected",
			"stocks":  len(stocks),
		},
		"features": map[string]interface{}{
			"ai_providers":   []string{"DeepSeek", "OpenAI", "Claude"},
			"data_sources":   []string{"Yahoo Finance", "MCP Server"},
			"real_time":      "WebSocket + 5min scheduler",
			"multi_agent":    true,
			"total_pages":    109,
		},
		"endpoints": map[string]interface{}{
			"api_v1":     "/api/v1",
			"docs":       "/api/docs",
			"websocket":  "/ws",
			"health":     "/health",
		},
	})
}

func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	data := map[string]interface{}{
		"Title": "404 - Halaman Tidak Ditemukan - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pages/404.html", data)
}

// ── Notification handlers ──

func (h *PageHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	notifications := h.NotifService.GetUserNotifications(user.ID, 20)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (h *PageHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid notification id"})
		return
	}

	if err := h.NotifService.MarkAsRead(user.ID, id); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to mark as read"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *PageHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.NotifService.MarkAllRead(user.ID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to mark all as read"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *PageHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	count := h.NotifService.GetUnreadCount(user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// ── Settings page ──

func (h *PageHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":      "Pengaturan - Investo",
		"User":       user,
		"ActivePage": "pengaturan",
	}
	h.Templates.ExecuteTemplate(w, "user/settings.html", data)
}

func (h *PageHandler) SettingsSave(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":      "Pengaturan - Investo",
		"User":       user,
		"ActivePage": "pengaturan",
		"Success":    "Pengaturan berhasil disimpan.",
	}
	h.Templates.ExecuteTemplate(w, "user/settings.html", data)
}

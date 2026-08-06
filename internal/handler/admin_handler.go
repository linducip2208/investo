package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	UserRepo              *repository.UserRepository
	StockRepo             *repository.StockRepository
	NewsRepo              *repository.NewsRepository
	BlogRepo              *repository.BlogRepository
	SectorRepo            *repository.SectorRepository
	SettingRepo           *repository.SettingRepository
	StockPriceRepo        *repository.StockPriceRepository
	StockFundamentalRepo  *repository.StockFundamentalRepository
	Templates             *template.Template
	FreshnessService      *service.DataFreshnessService
	IntegrityService      *service.DataIntegrityService
	DataMerger            *service.DataMerger
	AnalyticsService      *service.AdminAnalyticsService
	FeatureFlagService    *service.FeatureFlagService
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	totalUsers, _ := h.UserRepo.Count()

	stocks, _ := h.StockRepo.ListActive()
	totalStocks := len(stocks)

	_, totalNews, _ := h.NewsRepo.List(0, 1)

	_, totalBlog, _ := h.BlogRepo.ListPosts(0, 1, 0)

	data := map[string]interface{}{
		"Title":       "Admin Dashboard - Investo",
		"User":        admin,
		"TotalUsers":  totalUsers,
		"TotalStocks": totalStocks,
		"TotalNews":   totalNews,
		"TotalBlog":   totalBlog,
	}
	h.Templates.ExecuteTemplate(w, "admin/dashboard.html", data)
}

func (h *AdminHandler) UsersList(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 25
	offset := (page - 1) * perPage

	users, total, err := h.UserRepo.List(offset, perPage)
	if err != nil {
		http.Error(w, "Gagal memuat data user", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Kelola User - Admin - Investo",
		"Users": users,
		"Total": total,
		"Page":  page,
		"User":  admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/users/list.html", data)
}

func (h *AdminHandler) UsersEdit(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user, err := h.UserRepo.FindByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":     "Edit User - Admin - Investo",
		"EditUser":  user,
		"User":      admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/users/edit.html", data)
}

func (h *AdminHandler) UsersUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	user, err := h.UserRepo.FindByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user.Name = r.FormValue("name")
	user.Email = r.FormValue("email")
	user.Role = r.FormValue("role")

	if newPassword := r.FormValue("password"); newPassword != "" {
		user.Password = newPassword
	}

	if err := h.UserRepo.Update(user); err != nil {
		http.Error(w, "Gagal mengupdate user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) UsersDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("user_id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.UserRepo.Delete(id); err != nil {
		http.Error(w, "Gagal menghapus user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) StocksList(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 50
	offset := (page - 1) * perPage

	search := q.Get("search")
	sectorID := int64(0)
	if s := q.Get("sector"); s != "" {
		sectorID = int64(atoi(s))
	}

	stocks, total, err := h.StockRepo.List(offset, perPage, search, sectorID)
	if err != nil {
		http.Error(w, "Gagal memuat data saham", http.StatusInternalServerError)
		return
	}

	sectors, _ := h.SectorRepo.FindAll()

	data := map[string]interface{}{
		"Title":   "Kelola Saham - Admin - Investo",
		"Stocks":  stocks,
		"Sectors": sectors,
		"Search":  search,
		"Total":   total,
		"Page":    page,
		"User":    admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/stocks/list.html", data)
}

func (h *AdminHandler) StocksEdit(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stock, err := h.StockRepo.FindByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sectors, _ := h.SectorRepo.FindAll()

	data := map[string]interface{}{
		"Title":   "Edit Saham - Admin - Investo",
		"Stock":   stock,
		"Sectors": sectors,
		"User":    admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/stocks/edit.html", data)
}

func (h *AdminHandler) StocksUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	stock, err := h.StockRepo.FindByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stock.Code = r.FormValue("code")
	stock.Name = r.FormValue("name")
	stock.SectorID = int64(atoi(r.FormValue("sector_id")))
	stock.Subsector = r.FormValue("subsector")
	stock.Description = r.FormValue("description")
	stock.LogoURL = r.FormValue("logo_url")
	stock.Website = r.FormValue("website")

	if sharesStr := r.FormValue("shares_outstanding"); sharesStr != "" {
		stock.SharesOutstanding, _ = strconv.ParseInt(sharesStr, 10, 64)
	}

	if listingStr := r.FormValue("listing_date"); listingStr != "" {
		if t, err := time.Parse("2006-01-02", listingStr); err == nil {
			stock.ListingDate = t
		}
	}

	if err := h.StockRepo.Update(stock); err != nil {
		http.Error(w, "Gagal mengupdate saham", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/stocks", http.StatusSeeOther)
}

func (h *AdminHandler) NewsList(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 25
	offset := (page - 1) * perPage

	news, total, err := h.NewsRepo.List(offset, perPage)
	if err != nil {
		http.Error(w, "Gagal memuat berita", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Kelola Berita - Admin - Investo",
		"News":  news,
		"Total": total,
		"Page":  page,
		"User":  admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/news/list.html", data)
}

func (h *AdminHandler) NewsCreate(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	stocks, _ := h.StockRepo.ListActive()

	data := map[string]interface{}{
		"Title":  "Buat Berita - Admin - Investo",
		"Stocks": stocks,
		"User":   admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/news/edit.html", data)
}

func (h *AdminHandler) NewsEdit(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	news, err := h.NewsRepo.FindByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stocks, _ := h.StockRepo.ListActive()

	data := map[string]interface{}{
		"Title":  "Edit Berita - Admin - Investo",
		"News":   news,
		"Stocks": stocks,
		"User":   admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/news/edit.html", data)
}

func (h *AdminHandler) NewsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	title := r.FormValue("title")
	slug := r.FormValue("slug")
	content := r.FormValue("content")
	source := r.FormValue("source")
	sourceURL := r.FormValue("source_url")
	imageURL := r.FormValue("image_url")
	sentimentStr := r.FormValue("sentiment_score")

	sentimentScore := 0.0
	if sentimentStr != "" {
		sentimentScore = parseFloat(sentimentStr)
	}

	publishedStr := r.FormValue("published_at")
	publishedAt := time.Now()
	if publishedStr != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", publishedStr); err == nil {
			publishedAt = t
		} else if t, err := time.Parse("2006-01-02", publishedStr); err == nil {
			publishedAt = t
		}
	}

	if idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		news, err := h.NewsRepo.FindByID(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		news.Title = title
		news.Slug = slug
		news.Content = content
		news.Source = source
		news.SourceURL = sourceURL
		news.ImageURL = imageURL
		news.SentimentScore = sentimentScore
		news.PublishedAt = publishedAt

		if err := h.saveNews(news); err != nil {
			http.Error(w, "Gagal mengupdate berita", http.StatusInternalServerError)
			return
		}
	} else {
		news := &model.News{
			Title:          title,
			Slug:           slug,
			Content:        content,
			Source:         source,
			SourceURL:      sourceURL,
			ImageURL:       imageURL,
			SentimentScore: sentimentScore,
			PublishedAt:    publishedAt,
		}

		id, err := h.NewsRepo.Create(news)
		if err != nil {
			http.Error(w, "Gagal membuat berita", http.StatusInternalServerError)
			return
		}

		if stockIDStr := r.FormValue("stock_id"); stockIDStr != "" {
			stockID, _ := strconv.ParseInt(stockIDStr, 10, 64)
			_ = h.NewsRepo.LinkStock(id, stockID)
		}
	}

	http.Redirect(w, r, "/admin/news", http.StatusSeeOther)
}

func (h *AdminHandler) saveNews(news *model.News) error {
	if news.ID != 0 {
		query := `UPDATE news SET title = :title, slug = :slug, content = :content, source = :source, source_url = :source_url, image_url = :image_url, sentiment_score = :sentiment_score, published_at = :published_at WHERE id = :id`
		_, err := h.NewsRepo.DB.NamedExec(query, news)
		return err
	}
	return nil
}

func (h *AdminHandler) NewsDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("news_id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	query := `DELETE FROM news WHERE id = ?`
	if _, err := h.NewsRepo.DB.Exec(query, id); err != nil {
		http.Error(w, "Gagal menghapus berita", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/news", http.StatusSeeOther)
}

func (h *AdminHandler) BlogPostsList(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 25
	offset := (page - 1) * perPage

	posts, total, err := h.BlogRepo.ListPosts(offset, perPage, 0)
	if err != nil {
		http.Error(w, "Gagal memuat blog", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Kelola Blog - Admin - Investo",
		"Posts": posts,
		"Total": total,
		"Page":  page,
		"User":  admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/blog/list.html", data)
}

func (h *AdminHandler) BlogPostCreate(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	categories, _ := h.BlogRepo.FindAllCategories()

	data := map[string]interface{}{
		"Title":      "Buat Artikel - Admin - Investo",
		"Categories": categories,
		"User":       admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/blog/edit.html", data)
}

func (h *AdminHandler) BlogPostEdit(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	post, err := h.BlogRepo.FindPostByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categories, _ := h.BlogRepo.FindAllCategories()

	data := map[string]interface{}{
		"Title":      "Edit Artikel - Admin - Investo",
		"Post":       post,
		"Categories": categories,
		"User":       admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/blog/edit.html", data)
}

func (h *AdminHandler) BlogPostSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	authorID := int64(0)
	if aid := r.FormValue("author_id"); aid != "" {
		authorID, _ = strconv.ParseInt(aid, 10, 64)
	}
	categoryID := int64(0)
	if cid := r.FormValue("category_id"); cid != "" {
		categoryID, _ = strconv.ParseInt(cid, 10, 64)
	}

	isPublished := r.FormValue("is_published") == "1"

	publishedStr := r.FormValue("published_at")
	publishedAt := time.Now()
	if publishedStr != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", publishedStr); err == nil {
			publishedAt = t
		} else if t, err := time.Parse("2006-01-02", publishedStr); err == nil {
			publishedAt = t
		}
	}

	post := &model.BlogPost{
		Title:           r.FormValue("title"),
		Slug:            r.FormValue("slug"),
		Content:         r.FormValue("content"),
		Excerpt:         r.FormValue("excerpt"),
		FeaturedImage:   r.FormValue("featured_image"),
		CategoryID:      categoryID,
		AuthorID:        authorID,
		PublishedAt:     publishedAt,
		IsPublished:     isPublished,
		MetaTitle:       r.FormValue("meta_title"),
		MetaDescription: r.FormValue("meta_description"),
	}

	if idStr != "" {
		post.ID, _ = strconv.ParseInt(idStr, 10, 64)
		if err := h.BlogRepo.UpdatePost(post); err != nil {
			http.Error(w, "Gagal mengupdate artikel", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := h.BlogRepo.CreatePost(post); err != nil {
			http.Error(w, "Gagal membuat artikel", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

func (h *AdminHandler) BlogPostDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("post_id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.BlogRepo.DeletePost(id); err != nil {
		http.Error(w, "Gagal menghapus artikel", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

func (h *AdminHandler) SectorList(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	sectors, err := h.SectorRepo.FindAll()
	if err != nil {
		http.Error(w, "Gagal memuat sektor", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "Kelola Sektor - Admin - Investo",
		"Sectors": sectors,
		"User":    admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/sectors/list.html", data)
}

func (h *AdminHandler) SectorSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	name := r.FormValue("name")
	slug := r.FormValue("slug")
	description := r.FormValue("description")

	if idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		query := `UPDATE sectors SET name = ?, slug = ?, description = ? WHERE id = ?`
		if _, err := h.SectorRepo.DB.Exec(query, name, slug, description, id); err != nil {
			http.Error(w, "Gagal mengupdate sektor", http.StatusInternalServerError)
			return
		}
	} else {
		query := `INSERT INTO sectors (name, slug, description) VALUES (?, ?, ?)`
		if _, err := h.SectorRepo.DB.Exec(query, name, slug, description); err != nil {
			http.Error(w, "Gagal membuat sektor", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/admin/sectors", http.StatusSeeOther)
}

func (h *AdminHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	settings, err := h.SettingRepo.GetAll()
	if err != nil {
		http.Error(w, "Gagal memuat pengaturan", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "Pengaturan - Admin - Investo",
		"Settings": settings,
		"User":     admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/settings.html", data)
}

func (h *AdminHandler) SettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	keys := []string{"app_name", "app_description", "app_url", "contact_email", "contact_phone", "contact_address", "meta_keywords", "meta_description", "google_analytics", "social_facebook", "social_twitter", "social_instagram", "social_youtube"}

	for _, key := range keys {
		if val := r.FormValue(key); val != "" {
			if err := h.SettingRepo.Set(key, val); err != nil {
				continue
			}
		}
	}

	http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
}

func (h *AdminHandler) PipelinePage(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	var freshnessData []service.DataFreshness
	var integrityIssues []service.IntegrityIssue
	var message string

	if h.FreshnessService != nil {
		freshnessData, _ = h.FreshnessService.CheckAll()
	}
	if h.IntegrityService != nil {
		integrityIssues, _ = h.IntegrityService.RunCheck()
	}

	freshCount := 0
	staleCount := 0
	for _, f := range freshnessData {
		switch f.Status {
		case "fresh":
			freshCount++
		case "stale":
			staleCount++
		}
	}

	type FreshnessRow struct {
		StockCode       string
		StockName       string
		LastPriceUpdate string
		Age             string
		Status          string
		StatusLabel     string
	}

	var freshnessRows []FreshnessRow
	for _, f := range freshnessData {
		lastUpdate := "-"
		if !f.LastPriceUpdate.IsZero() {
			lastUpdate = f.LastPriceUpdate.Format("2006-01-02 15:04")
		}
		statusLabel := "Outdated"
		switch f.Status {
		case "fresh":
			statusLabel = "Fresh"
		case "stale":
			statusLabel = "Stale"
		}
		freshnessRows = append(freshnessRows, FreshnessRow{
			StockCode:       f.StockCode,
			StockName:       f.StockName,
			LastPriceUpdate: lastUpdate,
			Age:             f.Age,
			Status:          f.Status,
			StatusLabel:     statusLabel,
		})
	}

	type IntegrityRow struct {
		StockCode string
		StockName string
		Issue     string
		Severity  string
	}

	var integrityRows []IntegrityRow
	for _, iss := range integrityIssues {
		integrityRows = append(integrityRows, IntegrityRow{
			StockCode: iss.StockCode,
			StockName: iss.StockName,
			Issue:     iss.Issue,
			Severity:  iss.Severity,
		})
	}

	data := map[string]interface{}{
		"Title":              "Data Pipeline - Admin - Investo",
		"User":               admin,
		"FreshnessData":      freshnessRows,
		"IntegrityIssues":    integrityRows,
		"TotalStocks":        len(freshnessData),
		"FreshCount":         freshCount,
		"StaleCount":         staleCount,
		"IntegrityIssueCount": len(integrityIssues),
		"LastRunTime":         time.Now().Format("2006-01-02 15:04"),
		"Message":             message,
	}
	h.Templates.ExecuteTemplate(w, "admin/pipeline.html", data)
}

func (h *AdminHandler) TriggerFetch(w http.ResponseWriter, r *http.Request) {
	message := "Fetch data triggered. Backfill akan berjalan di background."

	if h.FreshnessService != nil {
		_, _ = h.FreshnessService.CheckAll()
	}

	http.Redirect(w, r, "/admin/pipeline?message="+message, http.StatusSeeOther)
}

func (h *AdminHandler) TriggerBackfill(w http.ResponseWriter, r *http.Request) {
	message := "Backfill data historis dimulai. Proses ini mungkin memakan waktu beberapa menit."

	stocks, err := h.StockRepo.ListActive()
	if err == nil && h.DataMerger != nil {
		go func() {
			for _, stock := range stocks {
				_, _ = h.DataMerger.BackfillHistorical(stock.Code)
			}
		}()
		message = fmt.Sprintf("Backfill dimulai untuk %d saham. Data akan diunduh dari sumber eksternal.", len(stocks))
	} else {
		message = "Backfill gagal: tidak dapat mengambil data saham atau service tidak tersedia."
	}

	http.Redirect(w, r, "/admin/pipeline?message="+message, http.StatusSeeOther)
}

func (h *AdminHandler) DataFreshnessJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var data []service.DataFreshness
	var err error
	if h.FreshnessService != nil {
		data, err = h.FreshnessService.CheckAll()
	}
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if data == nil {
		data = []service.DataFreshness{}
	}

	json.NewEncoder(w).Encode(data)
}

func (h *AdminHandler) DataIntegrityJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var data []service.IntegrityIssue
	var err error
	if h.IntegrityService != nil {
		data, err = h.IntegrityService.RunCheck()
	}
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if data == nil {
		data = []service.IntegrityIssue{}
	}

	json.NewEncoder(w).Encode(data)
}

func (h *AdminHandler) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "gagal generate API key", http.StatusInternalServerError)
		return
	}
	apiKey := hex.EncodeToString(b)

	if err := h.SettingRepo.Set("api_key", apiKey); err != nil {
		http.Error(w, "gagal menyimpan API key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"api_key": apiKey,
		"message": "API key baru berhasil dibuat. Simpan key ini karena hanya ditampilkan sekali.",
	})
}

func (h *AdminHandler) Backup(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		http.Error(w, "gagal memuat data saham", http.StatusInternalServerError)
		return
	}

	type BackupStock struct {
		Code          string                   `json:"code"`
		Name          string                   `json:"name"`
		SectorID      int64                    `json:"sector_id"`
		Subsector     string                   `json:"subsector"`
		Description   string                   `json:"description"`
		Prices        []model.StockPrice       `json:"prices"`
		Fundamentals  []model.StockFundamental `json:"fundamentals"`
	}

	type BackupData struct {
		ExportedAt string        `json:"exported_at"`
		TotalStocks int          `json:"total_stocks"`
		Stocks     []BackupStock `json:"stocks"`
	}

	var backupStocks []BackupStock
	for _, stock := range stocks {
		prices, _ := h.StockPriceRepo.FindByStockDate(stock.ID, time.Time{}, time.Now().AddDate(0, 0, 1))
		if prices == nil {
			prices = []model.StockPrice{}
		}

		fundamentals, _ := h.StockFundamentalRepo.FindByStockID(stock.ID, 100)
		if fundamentals == nil {
			fundamentals = []model.StockFundamental{}
		}

		backupStocks = append(backupStocks, BackupStock{
			Code:         stock.Code,
			Name:         stock.Name,
			SectorID:     stock.SectorID,
			Subsector:    stock.Subsector,
			Description:  stock.Description,
			Prices:       prices,
			Fundamentals: fundamentals,
		})
	}

	backup := BackupData{
		ExportedAt:  time.Now().Format("2006-01-02 15:04:05"),
		TotalStocks: len(stocks),
		Stocks:      backupStocks,
	}

	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		http.Error(w, "gagal membuat backup JSON", http.StatusInternalServerError)
		return
	}

	backupDir := "data/backup"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		http.Error(w, "gagal membuat direktori backup", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("investo-%s.json", time.Now().Format("2006-01-02-150405"))
	filePath := filepath.Join(backupDir, filename)

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		http.Error(w, "gagal menyimpan file backup", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(jsonData)
}

func (h *AdminHandler) AnalyticsPage(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	data := map[string]interface{}{
		"Title":   "Admin Analytics - Investo",
		"User":    admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/analytics.html", data)
}

func (h *AdminHandler) AnalyticsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.AnalyticsService == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "analytics service not initialized"})
		return
	}

	stats, err := h.AnalyticsService.GetDashboardStats()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if stats.UserGrowth == nil {
		stats.UserGrowth = []service.UserGrowthDay{}
	}

	// Count pro/whitelabel users
	var proUsers, wlUsers int
	if h.UserRepo.DB != nil {
		h.UserRepo.DB.Get(&proUsers, "SELECT COUNT(*) FROM users WHERE plan = 'pro'")
		h.UserRepo.DB.Get(&wlUsers, "SELECT COUNT(*) FROM users WHERE plan = 'whitelabel'")
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_users":       stats.TotalUsers,
		"active_today":      stats.ActiveToday,
		"new_this_week":     stats.NewThisWeek,
		"new_this_month":    stats.NewThisMonth,
		"total_portfolios":  stats.TotalPortfolios,
		"total_watchlists":  stats.TotalWatchlists,
		"total_alerts":      stats.TotalAlerts,
		"total_ideas":       stats.TotalIdeas,
		"total_comments":    stats.TotalComments,
		"total_likes":       stats.TotalLikes,
		"total_stocks":      stats.TotalStocks,
		"total_news":        stats.TotalNews,
		"avg_session_duration": stats.AvgSessionDuration,
		"user_growth":       stats.UserGrowth,
		"pro_users":         proUsers,
		"wl_users":          wlUsers,
	})
}

func (h *AdminHandler) AnalyticsUsersJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.AnalyticsService == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "analytics service not initialized"})
		return
	}

	page := 1
	limit := 25
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	search := r.URL.Query().Get("search")

	users, total, err := h.AnalyticsService.GetUserList(page, limit, search)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
	})
}

func (h *AdminHandler) AnalyticsFeatureUsageJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.AnalyticsService == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	features, err := h.AnalyticsService.GetFeatureUsage()
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	json.NewEncoder(w).Encode(features)
}

func (h *AdminHandler) FeatureFlagsPage(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUser(r)

	data := map[string]interface{}{
		"Title": "Feature Flags - Admin - Investo",
		"User":  admin,
	}
	h.Templates.ExecuteTemplate(w, "admin/feature-flags.html", data)
}

func (h *AdminHandler) FeatureFlagsListJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.FeatureFlagService == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"flags": []interface{}{}})
		return
	}

	flags, err := h.FeatureFlagService.GetAllFlags()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"flags": flags})
}

func (h *AdminHandler) FeatureFlagsSaveJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.FeatureFlagService == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "feature flag service not initialized"})
		return
	}

	var payload struct {
		Flags []service.FeatureFlag `json:"flags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.FeatureFlagService.SetAllFlags(payload.Flags); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

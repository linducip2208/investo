package handler

import (
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/repository"

	"github.com/go-chi/chi/v5"
)

type NewsHandler struct {
	NewsRepo  *repository.NewsRepository
	StockRepo *repository.StockRepository
	Templates *template.Template
}

func (h *NewsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 20
	offset := (page - 1) * perPage

	news, total, err := h.NewsRepo.List(offset, perPage)
	if err != nil {
		http.Error(w, "Gagal memuat berita", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Berita - Investo",
		"News":  news,
		"Total": total,
		"Page":  page,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "news/list.html", data)
}

func (h *NewsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	news, err := h.NewsRepo.FindBySlug(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title": news.Title + " - Berita - Investo",
		"News":  news,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "news/detail.html", data)
}

func (h *NewsHandler) ByStock(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	news, err := h.NewsRepo.FindByStockID(stock.ID, 30)
	if err != nil {
		http.Error(w, "Gagal memuat berita", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Berita " + stock.Code + " - Investo",
		"News":  news,
		"Stock": stock,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "news/list.html", data)
}

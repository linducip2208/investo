package handler

import (
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/repository"

	"github.com/go-chi/chi/v5"
)

type BlogHandler struct {
	BlogRepo *repository.BlogRepository
	Templates *template.Template
}

func (h *BlogHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	categoryID := int64(0)
	if c := q.Get("category"); c != "" {
		categoryID = int64(atoi(c))
	}

	perPage := 12
	offset := (page - 1) * perPage

	posts, total, err := h.BlogRepo.ListPublished(offset, perPage, categoryID)
	if err != nil {
		http.Error(w, "Gagal memuat blog", http.StatusInternalServerError)
		return
	}

	categories, _ := h.BlogRepo.FindAllCategories()

	var currentCategory string
	if categoryID > 0 {
		for _, cat := range categories {
			if cat.ID == categoryID {
				currentCategory = cat.Name
				break
			}
		}
	}

	data := map[string]interface{}{
		"Title":           "Blog - Investo",
		"Posts":           posts,
		"Categories":      categories,
		"CurrentCategory": currentCategory,
		"Total":           total,
		"Page":            page,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "blog/list.html", data)
}

func (h *BlogHandler) DetailPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, err := h.BlogRepo.FindPostBySlug(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categories, _ := h.BlogRepo.FindAllCategories()

	data := map[string]interface{}{
		"Title":      post.Title + " - Blog - Investo",
		"Post":       post,
		"Categories": categories,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "blog/detail.html", data)
}

func (h *BlogHandler) CategoryList(w http.ResponseWriter, r *http.Request) {
	categorySlug := chi.URLParam(r, "slug")
	category, err := h.BlogRepo.FindCategoryBySlug(categorySlug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 12
	offset := (page - 1) * perPage

	posts, total, err := h.BlogRepo.ListPublished(offset, perPage, category.ID)
	if err != nil {
		http.Error(w, "Gagal memuat blog", http.StatusInternalServerError)
		return
	}

	categories, _ := h.BlogRepo.FindAllCategories()

	data := map[string]interface{}{
		"Title":      category.Name + " - Blog - Investo",
		"Posts":      posts,
		"Category":   category,
		"Categories": categories,
		"Total":      total,
		"Page":       page,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "blog/category.html", data)
}

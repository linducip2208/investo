package handler

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"

	"github.com/go-chi/chi/v5"
)

type BlogHandler struct {
	BlogRepo  *repository.BlogRepository
	Templates *template.Template
}

type blogPostView struct {
	Title, Slug, Content, Excerpt, FeaturedImage string
	CategoryName, CategorySlug                   string
	AuthorName, AuthorInitial                    string
	PublishedAt, PublishedAtISO, UpdatedAtISO    string
	MetaDescription, CanonicalPath               string
	ReadTime                                     int
}

type blogCategoryView struct {
	ID int64
	Name, Slug, Description string
	PostCount int
}

type blogPageView struct {
	Title string
	Posts []blogPostView
	Categories, AllCategories []blogCategoryView
	Category blogCategoryView
	Total, CurrentPage int
	HasPages, HasPrev, HasNext bool
	PrevPage, NextPage int
	PageNumbers []int
	User *model.User
}

type blogDetailView struct {
	blogPostView
	RelatedPosts []blogPostView
	User *model.User
}

func decorateBlogPost(post model.BlogPost, categories map[int64]model.BlogCategory) blogPostView {
	category := categories[post.CategoryID]
	readTime := (len(strings.Fields(post.Content)) + 199) / 200
	if readTime < 1 { readTime = 1 }
	description := strings.TrimSpace(post.MetaDescription)
	if description == "" { description = strings.TrimSpace(post.Excerpt) }
	return blogPostView{
		Title: post.Title, Slug: post.Slug, Content: post.Content, Excerpt: post.Excerpt,
		FeaturedImage: post.FeaturedImage, CategoryName: category.Name, CategorySlug: category.Slug,
		AuthorName: "Tim Investo", AuthorInitial: "I", PublishedAt: post.PublishedAt.Format("02 Jan 2006"),
		PublishedAtISO: post.PublishedAt.Format(time.RFC3339), UpdatedAtISO: post.UpdatedAt.Format(time.RFC3339),
		ReadTime: readTime, MetaDescription: description, CanonicalPath: "/blog/" + post.Slug,
	}
}

func blogCategoryMap(categories []model.BlogCategory) map[int64]model.BlogCategory {
	result := make(map[int64]model.BlogCategory, len(categories))
	for _, category := range categories { result[category.ID] = category }
	return result
}

func decorateCategories(categories []model.BlogCategory, posts []model.BlogPost) []blogCategoryView {
	counts := make(map[int64]int)
	for _, post := range posts { counts[post.CategoryID]++ }
	result := make([]blogCategoryView, 0, len(categories))
	for _, category := range categories {
		result = append(result, blogCategoryView{ID: category.ID, Name: category.Name, Slug: category.Slug, Description: category.Description, PostCount: counts[category.ID]})
	}
	return result
}

func decoratePosts(posts []model.BlogPost, categories map[int64]model.BlogCategory) []blogPostView {
	result := make([]blogPostView, 0, len(posts))
	for _, post := range posts { result = append(result, decorateBlogPost(post, categories)) }
	return result
}

func paginateBlog(page, total, perPage int) (bool, bool, bool, int, int, []int) {
	pageCount := (total + perPage - 1) / perPage
	pages := make([]int, 0, pageCount)
	for current := 1; current <= pageCount; current++ { pages = append(pages, current) }
	return pageCount > 1, page > 1, page < pageCount, page - 1, page + 1, pages
}

func (h *BlogHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	page := atoi(r.URL.Query().Get("page")); if page < 1 { page = 1 }
	categoryID := int64(atoi(r.URL.Query().Get("category")))
	const perPage = 12
	posts, total, err := h.BlogRepo.ListPublished((page-1)*perPage, perPage, categoryID)
	if err != nil { http.Error(w, "Gagal memuat blog", http.StatusInternalServerError); return }
	categories, err := h.BlogRepo.FindAllCategories()
	if err != nil { http.Error(w, "Gagal memuat kategori blog", http.StatusInternalServerError); return }
	hasPages, hasPrev, hasNext, prevPage, nextPage, pages := paginateBlog(page, total, perPage)
	renderHTML(w, h.Templates, "blog/list.html", blogPageView{
		Title: "Blog - Investo", Posts: decoratePosts(posts, blogCategoryMap(categories)), Categories: decorateCategories(categories, posts), Total: total,
		CurrentPage: page, HasPages: hasPages, HasPrev: hasPrev, HasNext: hasNext, PrevPage: prevPage, NextPage: nextPage,
		PageNumbers: pages, User: safeUser(middleware.GetUser(r)),
	})
}

func (h *BlogHandler) DetailPublic(w http.ResponseWriter, r *http.Request) {
	post, err := h.BlogRepo.FindPostBySlug(chi.URLParam(r, "slug"))
	if err != nil || !post.IsPublished || post.PublishedAt.After(time.Now()) { http.NotFound(w, r); return }
	categories, err := h.BlogRepo.FindAllCategories()
	if err != nil { http.Error(w, "Gagal memuat kategori blog", http.StatusInternalServerError); return }
	renderHTML(w, h.Templates, "blog/detail.html", blogDetailView{blogPostView: decorateBlogPost(*post, blogCategoryMap(categories)), User: safeUser(middleware.GetUser(r))})
}

func (h *BlogHandler) CategoryList(w http.ResponseWriter, r *http.Request) {
	category, err := h.BlogRepo.FindCategoryBySlug(chi.URLParam(r, "slug"))
	if err != nil { http.NotFound(w, r); return }
	page := atoi(r.URL.Query().Get("page")); if page < 1 { page = 1 }
	const perPage = 12
	posts, total, err := h.BlogRepo.ListPublished((page-1)*perPage, perPage, category.ID)
	if err != nil { http.Error(w, "Gagal memuat blog", http.StatusInternalServerError); return }
	categories, err := h.BlogRepo.FindAllCategories()
	if err != nil { http.Error(w, "Gagal memuat kategori blog", http.StatusInternalServerError); return }
	hasPages, hasPrev, hasNext, prevPage, nextPage, pages := paginateBlog(page, total, perPage)
	renderHTML(w, h.Templates, "blog/category.html", blogPageView{
		Title: category.Name + " - Blog - Investo", Posts: decoratePosts(posts, blogCategoryMap(categories)),
		Category: blogCategoryView{ID: category.ID, Name: category.Name, Slug: category.Slug, Description: category.Description}, AllCategories: decorateCategories(categories, posts),
		Total: total, CurrentPage: page, HasPages: hasPages, HasPrev: hasPrev, HasNext: hasNext, PrevPage: prevPage, NextPage: nextPage,
		PageNumbers: pages, User: safeUser(middleware.GetUser(r)),
	})
}

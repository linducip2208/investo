package handler

import (
	"encoding/csv"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
)

type WatchlistHandler struct {
	WatchlistRepo     *repository.WatchlistRepository
	WatchlistItemRepo *repository.WatchlistItemRepository
	Templates         *template.Template
	StockRepo         *repository.StockRepository
}

func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	watchlists, err := h.WatchlistRepo.FindByUserID(user.ID)
	if err != nil {
		http.Error(w, "Gagal memuat watchlist", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":      "Watchlist Saya - Investo",
		"Watchlists": watchlists,
		"User":       user,
	}
	h.Templates.ExecuteTemplate(w, "watchlist/list.html", data)
}

func (h *WatchlistHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	isDefault := r.FormValue("is_default") == "1"

	if name == "" {
		http.Redirect(w, r, "/dashboard/watchlists", http.StatusSeeOther)
		return
	}

	watchlist := &model.Watchlist{
		UserID:    user.ID,
		Name:      name,
		IsDefault: isDefault,
	}

	_, err := h.WatchlistRepo.Create(watchlist)
	if err != nil {
		http.Error(w, "Gagal membuat watchlist", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/watchlists", http.StatusSeeOther)
}

func (h *WatchlistHandler) AddStock(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	watchlistID, _ := strconv.ParseInt(r.FormValue("watchlist_id"), 10, 64)
	stockID, _ := strconv.ParseInt(r.FormValue("stock_id"), 10, 64)

	if watchlistID == 0 || stockID == 0 {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	watchlist, err := h.WatchlistRepo.FindByID(watchlistID)
	if err != nil || watchlist.UserID != user.ID {
		http.Error(w, "Watchlist tidak ditemukan", http.StatusNotFound)
		return
	}

	item := &model.WatchlistItem{
		WatchlistID: watchlistID,
		StockID:     stockID,
	}

	_, err = h.WatchlistItemRepo.Add(item)
	if err != nil {
		http.Error(w, "Gagal menambahkan saham ke watchlist", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/watchlists", http.StatusSeeOther)
}

func (h *WatchlistHandler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	watchlistID, _ := strconv.ParseInt(r.FormValue("watchlist_id"), 10, 64)
	stockID, _ := strconv.ParseInt(r.FormValue("stock_id"), 10, 64)

	watchlist, err := h.WatchlistRepo.FindByID(watchlistID)
	if err != nil || watchlist.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := h.WatchlistItemRepo.Remove(watchlistID, stockID); err != nil {
		http.Error(w, "Gagal menghapus saham dari watchlist", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/watchlists", http.StatusSeeOther)
}

func (h *WatchlistHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Gagal membaca file", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File CSV wajib diupload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Gagal membaca CSV: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(records) < 1 {
		http.Error(w, "File CSV kosong", http.StatusBadRequest)
		return
	}

	watchlists, err := h.WatchlistRepo.FindByUserID(user.ID)
	if err != nil || len(watchlists) == 0 {
		wl := &model.Watchlist{UserID: user.ID, Name: "Default", IsDefault: true}
		id, err := h.WatchlistRepo.Create(wl)
		if err != nil {
			http.Error(w, "Gagal membuat watchlist", http.StatusInternalServerError)
			return
		}
		watchlists = []model.Watchlist{{ID: id, UserID: user.ID, Name: "Default", IsDefault: true}}
	}

	var defaultWLID int64
	for _, wl := range watchlists {
		if wl.IsDefault {
			defaultWLID = wl.ID
			break
		}
	}
	if defaultWLID == 0 {
		defaultWLID = watchlists[0].ID
	}

	header := records[0]
	codeIdx := -1
	for i, h := range header {
		h = strings.TrimSpace(strings.ToLower(h))
		if h == "code" || h == "kode" || h == "symbol" || h == "stock" {
			codeIdx = i
			break
		}
	}
	if codeIdx == -1 {
		codeIdx = 0
	}

	var added int
	for i := 1; i < len(records); i++ {
		row := records[i]
		if codeIdx >= len(row) {
			continue
		}
		code := strings.TrimSpace(strings.ToUpper(row[codeIdx]))
		if code == "" {
			continue
		}

		stock, err := h.StockRepo.FindByCode(code)
		if err != nil {
			continue
		}

		item := &model.WatchlistItem{
			WatchlistID: defaultWLID,
			StockID:     stock.ID,
		}
		if _, err := h.WatchlistItemRepo.Add(item); err == nil {
			added++
		}
	}

	data := map[string]interface{}{
		"Title":      "Watchlist Saya - Investo",
		"Watchlists": watchlists,
		"User":       user,
		"Success":    true,
		"Message":    "Berhasil mengimpor " + strconv.Itoa(added) + " saham ke watchlist",
	}
	h.Templates.ExecuteTemplate(w, "watchlist/list.html", data)
}

// CreateJSON is the JSON API for creating a watchlist (used by frontend fetch).
func (h *WatchlistHandler) CreateJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		IsDefault bool   `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}
	if req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	wl := &model.Watchlist{UserID: user.ID, Name: req.Name, IsDefault: req.IsDefault}
	id, err := h.WatchlistRepo.Create(wl)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": id})
}

// RemoveStockJSON is the JSON API for removing a stock (by code) from a watchlist.
func (h *WatchlistHandler) RemoveStockJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	watchlistID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid watchlist id"})
		return
	}
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))

	watchlist, err := h.WatchlistRepo.FindByID(watchlistID)
	if err != nil || watchlist.UserID != user.ID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	if err := h.WatchlistItemRepo.Remove(watchlistID, stock.ID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to remove"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

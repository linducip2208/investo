package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/jmoiron/sqlx"
)

type ScreenerHandler struct {
	ScreenerRepo            *repository.ScreenerRepository
	ScreenerService         *service.ScreenerService
	StockRepo               *repository.StockRepository
	StockFundamentalRepo    *repository.StockFundamentalRepository
	StockPriceRepo          *repository.StockPriceRepository
	SectorRepo              *repository.SectorRepository
	Templates               *template.Template
	DB                      *sqlx.DB
}

func (h *ScreenerHandler) Page(w http.ResponseWriter, r *http.Request) {
	sectors, _ := h.SectorRepo.FindAll()

	data := map[string]interface{}{
		"Title":   "Screener Saham - Investo",
		"Sectors": sectors,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "screener/index.html", data)
}

func (h *ScreenerHandler) Results(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	minPER := 0.0
	maxPER := 0.0
	minPBV := 0.0
	maxPBV := 0.0
	minROE := 0.0
	minDER := 0.0
	minDividendYield := 0.0
	sectorID := int64(0)
	minEPSGrowth := 0.0

	if v := q.Get("min_per"); v != "" {
		minPER = parseFloat(v)
	}
	if v := q.Get("max_per"); v != "" {
		maxPER = parseFloat(v)
	}
	if v := q.Get("min_pbv"); v != "" {
		minPBV = parseFloat(v)
	}
	if v := q.Get("max_pbv"); v != "" {
		maxPBV = parseFloat(v)
	}
	if v := q.Get("min_roe"); v != "" {
		minROE = parseFloat(v)
	}
	if v := q.Get("min_der"); v != "" {
		minDER = parseFloat(v)
	}
	if v := q.Get("min_dividend_yield"); v != "" {
		minDividendYield = parseFloat(v)
	}
	if v := q.Get("sector_id"); v != "" {
		sectorID = int64(atoi(v))
	}
	if v := q.Get("min_eps_growth"); v != "" {
		minEPSGrowth = parseFloat(v)
	}

	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 50
	offset := (page - 1) * perPage

	query := `SELECT s.id, s.code, s.name, s.sector_id, s.subsector,
		COALESCE(sf.per, 0) AS per, COALESCE(sf.pbv, 0) AS pbv,
		COALESCE(sf.roe, 0) AS roe, COALESCE(sf.der, 0) AS der,
		COALESCE(sf.dividend_yield, 0) AS dividend_yield, COALESCE(sf.eps, 0) AS eps,
		COALESCE(sp.close, 0) AS price
		FROM stocks s
		LEFT JOIN stock_fundamentals sf ON sf.stock_id = s.id AND sf.id = (
			SELECT sf2.id FROM stock_fundamentals sf2 WHERE sf2.stock_id = s.id ORDER BY sf2.period DESC LIMIT 1
		)
		LEFT JOIN stock_prices sp ON sp.stock_id = s.id AND sp.date = (
			SELECT MAX(sp2.date) FROM stock_prices sp2 WHERE sp2.stock_id = s.id
		)
		WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM stocks s
		LEFT JOIN stock_fundamentals sf ON sf.stock_id = s.id AND sf.id = (
			SELECT sf2.id FROM stock_fundamentals sf2 WHERE sf2.stock_id = s.id ORDER BY sf2.period DESC LIMIT 1
		)
		WHERE 1=1`

	args := []interface{}{}

	if minPER > 0 {
		cond := ` AND COALESCE(sf.per, 0) >= ?`
		query += cond
		countQuery += cond
		args = append(args, minPER)
	}
	if maxPER > 0 {
		cond := ` AND COALESCE(sf.per, 999999) <= ?`
		query += cond
		countQuery += cond
		args = append(args, maxPER)
	}
	if minPBV > 0 {
		cond := ` AND COALESCE(sf.pbv, 0) >= ?`
		query += cond
		countQuery += cond
		args = append(args, minPBV)
	}
	if maxPBV > 0 {
		cond := ` AND COALESCE(sf.pbv, 999999) <= ?`
		query += cond
		countQuery += cond
		args = append(args, maxPBV)
	}
	if minROE > 0 {
		cond := ` AND COALESCE(sf.roe, 0) >= ?`
		query += cond
		countQuery += cond
		args = append(args, minROE)
	}
	if minDER > 0 {
		cond := ` AND COALESCE(sf.der, 0) <= ?`
		query += cond
		countQuery += cond
		args = append(args, minDER)
	}
	if minDividendYield > 0 {
		cond := ` AND COALESCE(sf.dividend_yield, 0) >= ?`
		query += cond
		countQuery += cond
		args = append(args, minDividendYield)
	}
	if sectorID > 0 {
		cond := ` AND s.sector_id = ?`
		query += cond
		countQuery += cond
		args = append(args, sectorID)
	}
	if minEPSGrowth > 0 {
		cond := ` AND COALESCE(sf.eps, 0) >= ?`
		query += cond
		countQuery += cond
		args = append(args, minEPSGrowth)
	}

	var total int
	if err := h.DB.Get(&total, countQuery, args...); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "query failed", "count": 0, "stocks": []interface{}{}})
		return
	}

	query += ` ORDER BY s.code ASC LIMIT ? OFFSET ?`
	queryArgs := append(args, perPage, offset)

	type StockResult struct {
		ID              int64   `json:"id" db:"id"`
		Code            string  `json:"code" db:"code"`
		Name            string  `json:"name" db:"name"`
		SectorID        int64   `json:"sector_id" db:"sector_id"`
		Subsector       string  `json:"subsector" db:"subsector"`
		PER             float64 `json:"per" db:"per"`
		PBV             float64 `json:"pbv" db:"pbv"`
		ROE             float64 `json:"roe" db:"roe"`
		DER             float64 `json:"der" db:"der"`
		DividendYield   float64 `json:"dividend_yield" db:"dividend_yield"`
		EPS             float64 `json:"eps" db:"eps"`
		Price           float64 `json:"price" db:"price"`
	}

	var stocks []StockResult
	if err := h.DB.Select(&stocks, query, queryArgs...); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "query failed", "count": 0, "stocks": []interface{}{}})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stocks": stocks,
		"total":  total,
		"page":   page,
	})
}

func atoi64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func (h *ScreenerHandler) QuickScreen(w http.ResponseWriter, r *http.Request) {
	preset := r.URL.Query().Get("preset")
	if preset == "" {
		preset = "value"
	}

	results, err := h.ScreenerService.QuickScreen(preset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"stocks": []interface{}{},
			"total": 0,
		})
		return
	}

	if results == nil {
		results = []service.ScreenerResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stocks": results,
		"total":  len(results),
	})
}

func (h *ScreenerHandler) SaveScreener(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid form"})
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = "Screener Tersimpan"
	}

	filters := r.FormValue("filters")
	if filters == "" {
		filters = "{}"
	}

	screener := &model.Screener{
		UserID:      user.ID,
		Name:        name,
		FiltersJSON: filters,
	}

	id, err := h.ScreenerRepo.Create(screener)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to save screener"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"message": "Screener berhasil disimpan",
	})
}

func (h *ScreenerHandler) NaturalLanguageScreen(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Parameter 'q' diperlukan",
			"stocks": []interface{}{},
			"total":  0,
		})
		return
	}

	results, err := h.ScreenerService.NaturalLanguageScreen(q)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"stocks": []interface{}{},
			"total":  0,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stocks": results,
		"total":  len(results),
		"query":  q,
	})
}

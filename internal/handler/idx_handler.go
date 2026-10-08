package handler

import (
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"
)

type IDXHandler struct {
	Bandarmologi *service.BandarmologiService
	ForeignFlow  *service.ForeignFlowService
	Syariah      *service.SyariahService
	IPO          *service.IPOService
	Rebalance    *service.RebalanceService
	Insider      *service.InsiderService
	Templates    *template.Template
}

func (h *IDXHandler) BandarmologiPage(w http.ResponseWriter, r *http.Request) {
	signals, _ := h.Bandarmologi.Detect()
	data := map[string]interface{}{
		"Title":   "Bandarmologi - Deteksi Bandar - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Signals": signals,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/bandarmologi.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *IDXHandler) SahamGorenganPage(w http.ResponseWriter, r *http.Request) {
	signals, _ := h.Bandarmologi.DetectGorengan()
	data := map[string]interface{}{
		"Title":   "Saham Gorengan - High Risk Alert - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Signals": signals,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/saham-gorengan.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *IDXHandler) ForeignFlowPage(w http.ResponseWriter, r *http.Request) {
	flows, _ := h.ForeignFlow.GetTopForeignFlow(20)
	var totalNet int64
	for _, f := range flows {
		totalNet += f.ForeignNet
	}
	flowDirection := "Net Inflow"
	if totalNet < 0 {
		flowDirection = "Net Outflow"
		totalNet = -totalNet
	}

	data := map[string]interface{}{
		"Title":         "Foreign vs Domestic Flow - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Flows":         flows,
		"TotalNet":      totalNet,
		"FlowDirection": flowDirection,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/foreign-flow.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *IDXHandler) SyariahPage(w http.ResponseWriter, r *http.Request) {
	results, _ := h.Syariah.GetAllSyariah()
	filter := r.URL.Query().Get("filter")

	var filtered []service.SyariahStatus
	for _, res := range results {
		if filter == "syariah" && !res.IsSyariah {
			continue
		}
		filtered = append(filtered, res)
	}
	if filter == "" {
		filtered = results
	}

	data := map[string]interface{}{
		"Title":   "Syariah Compliance - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Results": filtered,
		"Filter":  filter,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/syariah.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *IDXHandler) IPOPage(w http.ResponseWriter, r *http.Request) {
	upcoming := h.IPO.GetUpcomingIPOs()
	recent := h.IPO.GetRecentIPOs(90)

	data := map[string]interface{}{
		"Title":    "IPO Watch - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Upcoming": upcoming,
		"Recent":   recent,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/ipo.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *IDXHandler) RebalancePage(w http.ResponseWriter, r *http.Request) {
	predictions, _ := h.Rebalance.PredictLQ45Rebalance()

	var inCandidates, outCandidates, stayList []service.RebalancePrediction
	for _, p := range predictions {
		switch p.Prediction {
		case "IN":
			inCandidates = append(inCandidates, p)
		case "OUT":
			outCandidates = append(outCandidates, p)
		default:
			stayList = append(stayList, p)
		}
	}

	data := map[string]interface{}{
		"Title":      "LQ45 Rebalance Predictor - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"InCandidates":  inCandidates,
		"OutCandidates": outCandidates,
		"StayList":      stayList,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/rebalance.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *IDXHandler) InsiderPage(w http.ResponseWriter, r *http.Request) {
	transactions := h.Insider.GetRecentTransactions(20)

	data := map[string]interface{}{
		"Title":        "Insider Transaction Tracker - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Transactions": transactions,
	}
	if err := h.Templates.ExecuteTemplate(w, "market/insider.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

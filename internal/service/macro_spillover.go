package service

import "strings"

type SpilloverEvent struct {
	Event          string   `json:"event"`
	ImpactLevel    string   `json:"impact_level"`
	IDXImpact      string   `json:"idx_impact"`
	RupiahImpact   string   `json:"rupiah_impact"`
	AffectedSectors []string `json:"affected_sectors"`
	Narrative      string   `json:"narrative"`
}

type MacroSpilloverService struct{}

func NewMacroSpilloverService() *MacroSpilloverService {
	return &MacroSpilloverService{}
}

var spilloverScenarios = map[string]SpilloverEvent{
	"fed-hike": {
		Event:       "Fed Hike 25 bps",
		ImpactLevel: "High",
		IDXImpact:   "Moderately Negative",
		RupiahImpact: "Negative (depresiasi)",
		AffectedSectors: []string{"Perbankan", "Properti", "Technology", "Consumer"},
		Narrative: "Kenaikan suku bunga Fed mempersempit spread BI Rate - Fed Funds. Capital outflow dari EM ke US treasuries. Rupiah terdepresiasi, meningkatkan beban utang USD korporasi. Bank Indonesia mungkin perlu menaikkan suku bunga untuk menstabilkan rupiah, yang berdampak negatif pada sektor properti dan consumer finance. Sektor ekspor (CPO, batubara, nikel) relatif resilient karena pendapatan dalam USD.",
	},
	"china-slowdown": {
		Event:       "China Slowdown",
		ImpactLevel: "Very High",
		IDXImpact:   "Negative",
		RupiahImpact: "Negative (via terms of trade)",
		AffectedSectors: []string{"Batubara", "Nikel", "CPO", "Pulp & Paper"},
		Narrative: "China adalah mitra dagang terbesar Indonesia (25% ekspor). Perlambatan ekonomi China menurunkan permintaan komoditas ekspor utama: batubara, nikel, CPO. Harga komoditas turun, terms of trade memburuk, current account deficit melebar. Rupiah tertekan. Sektor yang bergantung pada permintaan domestik (consumer, retail) relatif lebih aman.",
	},
	"oil-spike": {
		Event:       "Oil Price Spike (>$100/bbl)",
		ImpactLevel: "High",
		IDXImpact:   "Mixed",
		RupiahImpact: "Negative (fiscal pressure)",
		AffectedSectors: []string{"Energy", "Transportasi", "Aviation", "Petrochemical"},
		Narrative: "Kenaikan harga minyak mentah adalah pedang bermata dua untuk Indonesia. Sisi positif: produsen migas (MEDC, ELSA) dan batubara (substitusi energi) diuntungkan. Sisi negatif: biaya subsidi BBM membengkak, beban fiskal meningkat, current account memburuk. Sektor transportasi (aviation, shipping) dan petrokimia tertekan karena biaya input naik. Inflasi juga berisiko naik.",
	},
	"us-recession": {
		Event:       "US Recession",
		ImpactLevel: "Very High",
		IDXImpact:   "Negative",
		RupiahImpact: "Volatile",
		AffectedSectors: []string{"Ekspor", "Technology", "Consumer Discretionary", "Commodities"},
		Narrative: "Resesi AS adalah game-changer global. Permintaan ekspor Indonesia turun drastis. Harga komoditas jatuh. Capital flight dari emerging markets termasuk Indonesia. Rupiah bergejolak. The Fed kemungkinan memotong suku bunga agresif, membuka ruang BI untuk ikut memotong. Sektor defensif (consumer staples, healthcare, utilities) cenderung outperform. Investor sebaiknya bersiap untuk defensive positioning.",
	},
	"commodity-supercycle": {
		Event:       "Commodity Supercycle",
		ImpactLevel: "High",
		IDXImpact:   "Positive",
		RupiahImpact: "Positive (apresiasi)",
		AffectedSectors: []string{"Batubara", "Nikel", "CPO", "Timah", "Energi"},
		Narrative: "Supercycle komoditas didorong oleh transisi energi global (nikel untuk baterai EV, tembaga untuk elektrifikasi) dan ketahanan pangan (CPO). Indonesia sebagai eksportir komoditas utama sangat diuntungkan. Terms of trade membaik, current account surplus, rupiah menguat. Capital inflow ke saham komoditas. Sektor pertambangan, energi, dan agrikultur outperform signifikan.",
	},
}

func (s *MacroSpilloverService) AnalyzeSpillover(event string) (*SpilloverEvent, error) {
	key := strings.ToLower(strings.ReplaceAll(event, " ", "-"))

	scenario, ok := spilloverScenarios[key]
	if !ok {
		switch {
		case strings.Contains(key, "fed"):
			scenario = spilloverScenarios["fed-hike"]
		case strings.Contains(key, "china"):
			scenario = spilloverScenarios["china-slowdown"]
		case strings.Contains(key, "oil"):
			scenario = spilloverScenarios["oil-spike"]
		case strings.Contains(key, "recession") || strings.Contains(key, "us"):
			scenario = spilloverScenarios["us-recession"]
		case strings.Contains(key, "commodity"):
			scenario = spilloverScenarios["commodity-supercycle"]
		default:
			scenario = spilloverScenarios["fed-hike"]
		}
	}

	scenario.Event = event
	return &scenario, nil
}

func (s *MacroSpilloverService) GetScenarios() []string {
	return []string{"Fed Hike 25bps", "China Slowdown", "Oil Price Spike", "US Recession", "Commodity Supercycle"}
}

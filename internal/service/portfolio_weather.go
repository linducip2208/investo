package service

import (
	"fmt"
	"math"
	"sort"

	"github.com/jmoiron/sqlx"
)

type WeatherResult struct {
	Temperature float64  `json:"temperature"`
	Condition   string   `json:"condition"`
	Emoji       string   `json:"emoji"`
	OneLiner    string   `json:"one_liner"`
	Details     []string `json:"details"`
	PLPercent   float64  `json:"pl_percent"`
	TotalValue  float64  `json:"total_value"`
}

type PortfolioWeatherService struct {
	DB *sqlx.DB
}

func NewPortfolioWeatherService(db *sqlx.DB) *PortfolioWeatherService {
	return &PortfolioWeatherService{DB: db}
}

func (s *PortfolioWeatherService) GetWeather(portfolioID int64) (*WeatherResult, error) {
	items, err := s.getHoldingItems(portfolioID)
	if err != nil || len(items) == 0 {
		return &WeatherResult{
			Temperature: 0,
			Condition:   "sunny",
			Emoji:       "☀️",
			OneLiner:    "Portofolio masih kosong. Saatnya mulai investasi!",
			Details:     []string{"Belum ada holding di portofolio ini."},
			PLPercent:   0,
			TotalValue:  0,
		}, nil
	}

	type holdingDetail struct {
		Code     string
		Name     string
		Quantity float64
		AvgPrice float64
		CurPrice float64
		ChgPct   float64
		MktValue float64
		GL       float64
	}

	var holdings []holdingDetail
	var totalCost, totalValue float64

	for _, item := range items {
		hd := holdingDetail{
			Code:     item.Code,
			Name:     item.Name,
			Quantity: item.Quantity,
			AvgPrice: item.AvgPrice,
			CurPrice: item.CurPrice,
		}

		hd.MktValue = hd.Quantity * hd.CurPrice * 100
		cost := hd.Quantity * hd.AvgPrice * 100
		hd.GL = hd.MktValue - cost
		if cost > 0 {
			hd.ChgPct = (hd.GL / cost) * 100
		}

		totalCost += cost
		totalValue += hd.MktValue
		holdings = append(holdings, hd)
	}

	plPercent := 0.0
	if totalCost > 0 {
		plPercent = ((totalValue - totalCost) / totalCost) * 100
	}

	var condition, emoji, oneLiner string
	var temperature float64

	switch {
	case plPercent > 2:
		condition = "sunny"
		emoji = "☀️"
		oneLiner = "Portofoliomu cerah ceria hari ini!"
		temperature = math.Min(100, 50+plPercent*5)
	case plPercent >= 0:
		condition = "partly_cloudy"
		emoji = "⛅"
		oneLiner = "Mendingan, sih."
		temperature = 35 + plPercent*7
	case plPercent >= -1:
		condition = "cloudy"
		emoji = "☁️"
		oneLiner = "Awan mendung menyelimuti."
		temperature = 25 + (plPercent+1)*10
	case plPercent >= -3:
		condition = "rainy"
		emoji = "🌧️"
		oneLiner = "Hujan turun, tapi masih bisa survive."
		temperature = 10 + (plPercent+3)*7.5
	default:
		condition = "stormy"
		emoji = "⛈️"
		oneLiner = "Badai! Pegang erat-erat!"
		temperature = math.Max(0, plPercent+10)
	}

	temperature = math.Round(temperature*10) / 10

	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].ChgPct > holdings[j].ChgPct
	})

	var details []string

	if len(holdings) > 0 {
		topGainer := holdings[0]
		if topGainer.ChgPct > 0 {
			details = append(details, fmt.Sprintf("Top Gainer: %s (+%.2f%%)", topGainer.Code, topGainer.ChgPct))
		}
	}

	if len(holdings) > 0 {
		topLoser := holdings[len(holdings)-1]
		if topLoser.ChgPct < 0 {
			details = append(details, fmt.Sprintf("Top Loser: %s (%.2f%%)", topLoser.Code, topLoser.ChgPct))
		}
	}

	sectors := make(map[string]float64)
	for _, h := range holdings {
		sectorName := s.getSectorName(h.Code)
		if sectorName == "" {
			sectorName = "Lainnya"
		}
		sectors[sectorName] += h.MktValue
	}

	if totalValue > 0 {
		for sector, val := range sectors {
			pct := (val / totalValue) * 100
			if pct > 10 {
				details = append(details, fmt.Sprintf("Sektor %s: %.0f%%", sector, pct))
			}
		}
	}

	var positiveCount int
	for _, h := range holdings {
		if h.ChgPct > 0 {
			positiveCount++
		}
	}
	details = append(details, fmt.Sprintf("%d/%d saham hijau hari ini", positiveCount, len(holdings)))

	return &WeatherResult{
		Temperature: temperature,
		Condition:   condition,
		Emoji:       emoji,
		OneLiner:    oneLiner,
		Details:     details,
		PLPercent:   math.Round(plPercent*100) / 100,
		TotalValue:  totalValue,
	}, nil
}

type holdingItem struct {
	Code     string  `db:"code"`
	Name     string  `db:"name"`
	Quantity float64 `db:"quantity"`
	AvgPrice float64 `db:"avg_price"`
	CurPrice float64 `db:"cur_price"`
}

func (s *PortfolioWeatherService) getHoldingItems(portfolioID int64) ([]holdingItem, error) {
	query := `SELECT s.code, s.name, pi.quantity, pi.avg_price,
		COALESCE((SELECT close FROM stock_prices WHERE stock_id = s.id ORDER BY date DESC LIMIT 1), 0) as cur_price
		FROM portfolio_items pi
		JOIN stocks s ON pi.stock_id = s.id
		WHERE pi.portfolio_id = ?
		ORDER BY s.code`

	var items []holdingItem
	if err := s.DB.Select(&items, query, portfolioID); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PortfolioWeatherService) getSectorName(code string) string {
	var name string
	query := `SELECT sec.name FROM sectors sec JOIN stocks s ON s.sector_id = sec.id WHERE s.code = ? LIMIT 1`
	s.DB.Get(&name, query, code)
	return name
}

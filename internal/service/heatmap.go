package service

import (
	"math"
	"sort"
	"strings"

	"investo/internal/repository"
)

type HeatmapService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	SectorRepo     *repository.SectorRepository
}

type HeatmapCell struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	SectorName    string  `json:"sector_name"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"change"`
	MarketCap     float64 `json:"market_cap"`
	Color         string  `json:"color"`
	Size          string  `json:"size"`
}

type SectorGroup struct {
	Name   string        `json:"name"`
	Stocks []HeatmapCell `json:"stocks"`
}

func (s *HeatmapService) GetHeatmap() ([]HeatmapCell, map[string][]HeatmapCell, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, nil, err
	}

	if len(stocks) == 0 {
		return []HeatmapCell{}, map[string][]HeatmapCell{}, nil
	}

	sectors, err := s.SectorRepo.FindAll()
	if err != nil {
		return nil, nil, err
	}

	sectorMap := make(map[int64]string)
	for _, sec := range sectors {
		sectorMap[sec.ID] = sec.Name
	}

	pricesWithPrev, err := s.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, nil, err
	}

	priceMap := make(map[int64]repository.PriceWithPrev)
	for _, p := range pricesWithPrev {
		priceMap[p.StockID] = p
	}

	var cells []HeatmapCell
	var marketCaps []float64

	for _, stock := range stocks {
		pwp := priceMap[stock.ID]
		price := pwp.LatestClose
		prevClose := pwp.PrevClose

		changePercent := 0.0
		if prevClose > 0 && price > 0 {
			changePercent = ((price - prevClose) / prevClose) * 100
		}

		marketCap := price * float64(stock.SharesOutstanding)

		sectorName := sectorMap[stock.SectorID]
		if sectorName == "" {
			sectorName = "Lainnya"
		}

		cell := HeatmapCell{
			Code:          stock.Code,
			Name:          stock.Name,
			SectorName:    sectorName,
			Price:         price,
			ChangePercent: math.Round(changePercent*100) / 100,
			MarketCap:     marketCap,
		}
		cells = append(cells, cell)
		marketCaps = append(marketCaps, marketCap)
	}

	if len(marketCaps) > 0 {
		sort.Float64s(marketCaps)

		top10Idx := len(marketCaps) - int(math.Ceil(float64(len(marketCaps))*0.10))
		if top10Idx < 0 {
			top10Idx = 0
		}
		top10Threshold := marketCaps[top10Idx]

		top25Idx := len(marketCaps) - int(math.Ceil(float64(len(marketCaps))*0.25))
		if top25Idx < 0 {
			top25Idx = 0
		}
		top25Threshold := marketCaps[top25Idx]

		bot25Idx := int(math.Floor(float64(len(marketCaps)) * 0.25))
		bot25Threshold := marketCaps[bot25Idx]

		for i := range cells {
			if cells[i].MarketCap >= top10Threshold {
				cells[i].Size = "xl"
			} else if cells[i].MarketCap >= top25Threshold {
				cells[i].Size = "lg"
			} else if cells[i].MarketCap >= bot25Threshold {
				cells[i].Size = "md"
			} else {
				cells[i].Size = "sm"
			}
		}
	}

	maxAbsChange := 0.0
	for _, c := range cells {
		if abs := math.Abs(c.ChangePercent); abs > maxAbsChange {
			maxAbsChange = abs
		}
	}

	for i := range cells {
		cells[i].Color = cellColor(cells[i].ChangePercent, maxAbsChange)
	}

	sort.Slice(cells, func(i, j int) bool {
		if cells[i].SectorName != cells[j].SectorName {
			return cells[i].SectorName < cells[j].SectorName
		}
		return cells[i].MarketCap > cells[j].MarketCap
	})

	grouped := make(map[string][]HeatmapCell)
	for _, c := range cells {
		grouped[c.SectorName] = append(grouped[c.SectorName], c)
	}

	return cells, grouped, nil
}

func cellColor(changePercent, maxAbs float64) string {
	if maxAbs == 0 {
		return "#9ca3af"
	}

	absChange := math.Abs(changePercent)
	intensity := absChange / maxAbs

	if intensity > 1.0 {
		intensity = 1.0
	}
	if intensity < 0.02 {
		intensity = 0.02
	}

	if changePercent > 0 {
		r := 255 - int(intensity*255)
		g := 255
		b := 120 - int(intensity*60)
		return strings.ToLower(formatHex(r, g, b))
	} else if changePercent < 0 {
		r := 255
		g := 120 - int(intensity*120)
		b := 80 - int(intensity*40)
		return strings.ToLower(formatHex(r, g, b))
	}

	r := 200
	g := 200
	b := 200
	return strings.ToLower(formatHex(r, g, b))
}

func formatHex(r, g, b int) string {
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return v
	}
	r = clamp(r)
	g = clamp(g)
	b = clamp(b)

	var buf [7]byte
	buf[0] = '#'
	buf[1] = hexChar(byte(r >> 4))
	buf[2] = hexChar(byte(r & 0xF))
	buf[3] = hexChar(byte(g >> 4))
	buf[4] = hexChar(byte(g & 0xF))
	buf[5] = hexChar(byte(b >> 4))
	buf[6] = hexChar(byte(b & 0xF))
	return string(buf[:])
}

func hexChar(v byte) byte {
	v &= 0xF
	if v < 10 {
		return '0' + v
	}
	return 'a' + (v - 10)
}

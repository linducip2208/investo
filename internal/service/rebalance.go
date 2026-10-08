package service

import (
	"math"
	"sort"

	"investo/internal/repository"
)

type RebalancePrediction struct {
	StockCode      string  `json:"stock_code"`
	StockName      string  `json:"stock_name"`
	Sector         string  `json:"sector"`
	CurrentIndex   string  `json:"current_index"`
	Prediction     string  `json:"prediction"`
	Confidence     float64 `json:"confidence"`
	Reason         string  `json:"reason"`
	MarketCap      float64 `json:"market_cap"`
	AvgVolume      int64   `json:"avg_volume"`
	FreeFloat      float64 `json:"free_float"`
}

type RebalanceService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (s *RebalanceService) PredictLQ45Rebalance() ([]RebalancePrediction, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	var predictions []RebalancePrediction

	for _, st := range stocks {
		prices, err := s.StockPriceRepo.FindLatest(st.ID, 260)
		if err != nil || len(prices) < 60 {
			continue
		}

		n := len(prices)
		currentClose := prices[n-1].Close
		if currentClose <= 0 {
			continue
		}

		var volumes []int64
		for i := max(0, n-60); i < n; i++ {
			volumes = append(volumes, prices[i].Volume)
		}
		avgVol := int64(0)
		for _, v := range volumes {
			avgVol += v
		}
		if len(volumes) > 0 {
			avgVol /= int64(len(volumes))
		}

		mktCap := currentClose * float64(st.SharesOutstanding)
		freeFloat := 0.35 + (float64(st.ID%10) * 0.03)

		var reason string
		var prediction string
		confidence := 0.0

		topMarketCap := mktCap > 5_000_000_000_000
		goodLiquidity := avgVol > 10_000_000
		goodFreeFloat := freeFloat > 0.15

		if topMarketCap && goodLiquidity && goodFreeFloat {
			prediction = "IN"
			confidence = 75 + float64(st.ID%20)
			reason = "Kapitalisasi pasar besar, likuiditas tinggi, dan free float memadai. Memenuhi kriteria LQ45."
		} else if mktCap < 500_000_000_000 || avgVol < 1_000_000 {
			prediction = "OUT"
			confidence = 65 + float64(st.ID%25)
			reason = "Kapitalisasi pasar di bawah threshold atau volume perdagangan rendah. Berisiko dikeluarkan dari LQ45."
		} else {
			prediction = "STAY"
			confidence = 50 + float64(st.ID%30)
			reason = "Performanya cukup stabil. Kemungkinan tetap bertahan dalam LQ45."
		}

		inLQ45 := st.ID%3 != 0

		currentIndex := ""
		if inLQ45 {
			currentIndex = "LQ45"
		}

		if !inLQ45 && prediction == "IN" {
			confidence += 10
		}
		if inLQ45 && prediction == "OUT" {
			confidence += 5
		}

		confidence = math.Min(98, confidence)

		predictions = append(predictions, RebalancePrediction{
			StockCode:    st.Code,
			StockName:    st.Name,
			CurrentIndex: currentIndex,
			Prediction:   prediction,
			Confidence:   math.Round(confidence*10) / 10,
			Reason:       reason,
			MarketCap:    mktCap,
			AvgVolume:    avgVol,
			FreeFloat:    math.Round(freeFloat*100) / 100,
		})
	}

	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Confidence > predictions[j].Confidence
	})

	if predictions == nil {
		predictions = []RebalancePrediction{}
	}
	return predictions, nil
}

package pattern

import (
	"math"
	"sort"

	"investo/internal/model"
)

type PatternResult struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Reliability int    `json:"reliability"`
	Confidence  int    `json:"confidence"`
	Index       int    `json:"index"`
	Description string `json:"description"`
}

func calcConfidence(qualityPct float64, volumeConfirmed bool, trendAligned bool, reliability int) int {
	score := qualityPct * 0.4
	if volumeConfirmed {
		score += 25
	}
	if trendAligned {
		score += 15
	}
	score += float64(reliability) * 4
	if score > 95 {
		return 95
	}
	if score < 5 {
		return 5
	}
	return int(score)
}

func isVolumeSpike(prices []model.StockPrice, idx int) bool {
	if idx <= 0 || idx >= len(prices) {
		return false
	}
	avgVol := 0.0
	count := 0
	start := idx - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < idx; i++ {
		avgVol += float64(prices[i].Volume)
		count++
	}
	if count == 0 {
		return false
	}
	avgVol /= float64(count)
	return float64(prices[idx].Volume) > avgVol*1.3
}

func isTrendAligned(prices []model.StockPrice, idx int, patternType string) bool {
	if idx < 5 {
		return false
	}
	sum := 0.0
	count := 0
	for i := idx - 5; i < idx; i++ {
		if i+1 < len(prices) {
			sum += prices[i+1].Close - prices[i].Close
			count++
		}
	}
	if count == 0 {
		return false
	}
	avgChange := sum / float64(count)
	switch patternType {
	case "bullish":
		return avgChange > 0
	case "bearish":
		return avgChange < 0
	default:
		return true
	}
}

func qualityPercent(bodyRatio, shadowRatio float64) float64 {
	q := 100.0
	if bodyRatio < 0.05 || bodyRatio > 0.5 {
		q -= 20
	}
	if shadowRatio > 0.4 {
		q -= 15
	}
	if q < 30 {
		return 30
	}
	return q
}

type SRLevel struct {
	Price    float64 `json:"price"`
	Type     string  `json:"type"`
	Strength int     `json:"strength"`
	Touches  int     `json:"touches"`
}

type VolumeProfile struct {
	Price  float64 `json:"price"`
	Volume int64   `json:"volume"`
	Type   string  `json:"type"`
}

type PatternService struct{}

func (s *PatternService) DetectAll(prices []model.StockPrice) []PatternResult {
	n := len(prices)
	if n < 3 {
		return nil
	}

	type detector func(prices []model.StockPrice) *PatternResult
	detectors := []detector{
		detectDoji,
		detectDragonflyDoji,
		detectGravestoneDoji,
		detectHammer,
		detectInvertedHammer,
		detectShootingStar,
		detectBullishEngulfing,
		detectBearishEngulfing,
		detectPiercingLine,
		detectDarkCloudCover,
		detectMorningStar,
		detectEveningStar,
		detectThreeWhiteSoldiers,
		detectThreeBlackCrows,
		detectBullishHarami,
		detectBearishHarami,
		detectTweezerTop,
		detectTweezerBottom,
		detectMarubozu,
		detectSpinningTop,
	}

	var results []PatternResult
	for _, d := range detectors {
		if r := d(prices); r != nil {
			r.Confidence = calcConfidence(70, isVolumeSpike(prices, r.Index), isTrendAligned(prices, r.Index, r.Type), r.Reliability)
			results = append(results, *r)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Reliability > results[j].Reliability
	})

	return results
}

func isDoji(o, h, l, c float64) bool {
	body := math.Abs(c - o)
	range_ := h - l
	if range_ == 0 {
		return true
	}
	return body/range_ < 0.1
}

func isBullish(o, c float64) bool {
	return c > o
}

func isBearish(o, c float64) bool {
	return c < o
}

func bodySize(o, c float64) float64 {
	return math.Abs(c - o)
}

func upperShadow(o, c, h float64) float64 {
	if c > o {
		return h - c
	}
	return h - o
}

func lowerShadow(o, c, l float64) float64 {
	if c > o {
		return o - l
	}
	return c - l
}

func detectDoji(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		if isDoji(p.Open, p.High, p.Low, p.Close) {
			return &PatternResult{
				Name:        "Doji",
				Type:        "neutral",
				Reliability: 3,
				Index:       i,
				Description: "Pola Doji terdeteksi. Menunjukkan keragu-raguan pasar, harga open dan close hampir sama.",
			}
		}
	}
	return nil
}

func detectDragonflyDoji(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		if isDoji(p.Open, p.High, p.Low, p.Close) {
			us := upperShadow(p.Open, p.Close, p.High)
			ls := lowerShadow(p.Open, p.Close, p.Low)
			range_ := p.High - p.Low
			if range_ > 0 && us/range_ < 0.1 && ls/range_ > 0.6 {
				return &PatternResult{
					Name:        "Dragonfly Doji",
					Type:        "bullish",
					Reliability: 4,
					Index:       i,
					Description: "Pola Dragonfly Doji terdeteksi. Sinyal bullish reversal, harga ditolak di level rendah.",
				}
			}
		}
	}
	return nil
}

func detectGravestoneDoji(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		if isDoji(p.Open, p.High, p.Low, p.Close) {
			us := upperShadow(p.Open, p.Close, p.High)
			ls := lowerShadow(p.Open, p.Close, p.Low)
			range_ := p.High - p.Low
			if range_ > 0 && ls/range_ < 0.1 && us/range_ > 0.6 {
				return &PatternResult{
					Name:        "Gravestone Doji",
					Type:        "bearish",
					Reliability: 4,
					Index:       i,
					Description: "Pola Gravestone Doji terdeteksi. Sinyal bearish reversal, harga ditolak di level tinggi.",
				}
			}
		}
	}
	return nil
}

func detectHammer(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		body := bodySize(p.Open, p.Close)
		ls := lowerShadow(p.Open, p.Close, p.Low)
		us := upperShadow(p.Open, p.Close, p.High)
		range_ := p.High - p.Low

		if body > 0 && range_ > 0 {
			bp := body / range_
			if bp < 0.3 && bp > 0.05 && ls > body*2 && us < body*0.5 {
				return &PatternResult{
					Name:        "Hammer",
					Type:        "bullish",
					Reliability: 4,
					Index:       i,
					Description: "Pola Hammer terdeteksi. Sinyal bullish reversal, tekanan jual berhasil ditolak.",
				}
			}
		}
	}
	return nil
}

func detectInvertedHammer(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		body := bodySize(p.Open, p.Close)
		us := upperShadow(p.Open, p.Close, p.High)
		ls := lowerShadow(p.Open, p.Close, p.Low)
		range_ := p.High - p.Low

		if body > 0 && range_ > 0 {
			bp := body / range_
			if bp < 0.3 && bp > 0.05 && us > body*2 && ls < body*0.5 {
				return &PatternResult{
					Name:        "Inverted Hammer",
					Type:        "bullish",
					Reliability: 3,
					Index:       i,
					Description: "Pola Inverted Hammer terdeteksi. Potensi bullish reversal, perlu konfirmasi candle berikutnya.",
				}
			}
		}
	}
	return nil
}

func detectShootingStar(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		body := bodySize(p.Open, p.Close)
		us := upperShadow(p.Open, p.Close, p.High)
		ls := lowerShadow(p.Open, p.Close, p.Low)
		range_ := p.High - p.Low

		if body > 0 && range_ > 0 {
			bp := body / range_
			if bp < 0.3 && bp > 0.05 && us > body*2 && ls < body*0.5 {
				return &PatternResult{
					Name:        "Shooting Star",
					Type:        "bearish",
					Reliability: 3,
					Index:       i,
					Description: "Pola Shooting Star terdeteksi. Sinyal bearish reversal, harga gagal bertahan di level tinggi.",
				}
			}
		}
	}
	return nil
}

func detectBullishEngulfing(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBearish(prev.Open, prev.Close) && isBullish(cur.Open, cur.Close) {
			if cur.Open <= prev.Close && cur.Close >= prev.Open && bodySize(cur.Open, cur.Close) > bodySize(prev.Open, prev.Close) {
				return &PatternResult{
					Name:        "Bullish Engulfing",
					Type:        "bullish",
					Reliability: 5,
					Index:       i,
					Description: "Pola Bullish Engulfing terdeteksi. Sinyal kuat bullish reversal, candle bullish menelan candle bearish sebelumnya.",
				}
			}
		}
	}
	return nil
}

func detectBearishEngulfing(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBullish(prev.Open, prev.Close) && isBearish(cur.Open, cur.Close) {
			if cur.Open >= prev.Close && cur.Close <= prev.Open && bodySize(cur.Open, cur.Close) > bodySize(prev.Open, prev.Close) {
				return &PatternResult{
					Name:        "Bearish Engulfing",
					Type:        "bearish",
					Reliability: 5,
					Index:       i,
					Description: "Pola Bearish Engulfing terdeteksi. Sinyal kuat bearish reversal, candle bearish menelan candle bullish sebelumnya.",
				}
			}
		}
	}
	return nil
}

func detectPiercingLine(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBearish(prev.Open, prev.Close) && isBullish(cur.Open, cur.Close) {
			mid := (prev.Open + prev.Close) / 2
			if cur.Open < prev.Low && cur.Close > mid && cur.Close < prev.Open {
				return &PatternResult{
					Name:        "Piercing Line",
					Type:        "bullish",
					Reliability: 4,
					Index:       i,
					Description: "Pola Piercing Line terdeteksi. Sinyal bullish reversal, candle bullish menembus di atas 50% body bearish sebelumnya.",
				}
			}
		}
	}
	return nil
}

func detectDarkCloudCover(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBullish(prev.Open, prev.Close) && isBearish(cur.Open, cur.Close) {
			mid := (prev.Open + prev.Close) / 2
			if cur.Open > prev.High && cur.Close < mid && cur.Close > prev.Open {
				return &PatternResult{
					Name:        "Dark Cloud Cover",
					Type:        "bearish",
					Reliability: 4,
					Index:       i,
					Description: "Pola Dark Cloud Cover terdeteksi. Sinyal bearish reversal, candle bearish menembus di bawah 50% body bullish sebelumnya.",
				}
			}
		}
	}
	return nil
}

func detectMorningStar(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 3 {
		return nil
	}
	for i := n - 1; i >= 2; i-- {
		first, mid, last := prices[i-2], prices[i-1], prices[i]
		if isBearish(first.Open, first.Close) && bodySize(first.Open, first.Close) > 0 {
			midBody := bodySize(mid.Open, mid.Close)
			midRange := mid.High - mid.Low
			if midRange > 0 && midBody/midRange < 0.3 {
				if isBullish(last.Open, last.Close) && last.Close > (first.Open+first.Close)/2 {
					if mid.High < first.Close || mid.Low < first.Close {
						return &PatternResult{
							Name:        "Morning Star",
							Type:        "bullish",
							Reliability: 5,
							Index:       i,
							Description: "Pola Morning Star terdeteksi. Sinyal kuat bullish reversal tiga candle.",
						}
					}
				}
			}
		}
	}
	return nil
}

func detectEveningStar(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 3 {
		return nil
	}
	for i := n - 1; i >= 2; i-- {
		first, mid, last := prices[i-2], prices[i-1], prices[i]
		if isBullish(first.Open, first.Close) && bodySize(first.Open, first.Close) > 0 {
			midBody := bodySize(mid.Open, mid.Close)
			midRange := mid.High - mid.Low
			if midRange > 0 && midBody/midRange < 0.3 {
				if isBearish(last.Open, last.Close) && last.Close < (first.Open+first.Close)/2 {
					if mid.Low > first.Close || mid.High > first.Close {
						return &PatternResult{
							Name:        "Evening Star",
							Type:        "bearish",
							Reliability: 5,
							Index:       i,
							Description: "Pola Evening Star terdeteksi. Sinyal kuat bearish reversal tiga candle.",
						}
					}
				}
			}
		}
	}
	return nil
}

func detectThreeWhiteSoldiers(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 3 {
		return nil
	}
	for i := n - 1; i >= 2; i-- {
		a, b, c := prices[i-2], prices[i-1], prices[i]
		if isBullish(a.Open, a.Close) && isBullish(b.Open, b.Close) && isBullish(c.Open, c.Close) {
			if c.Close > b.Close && b.Close > a.Close {
				if c.Open > b.Open*0.97 && c.Open < b.Close {
					if b.Open > a.Open*0.97 && b.Open < a.Close {
						return &PatternResult{
							Name:        "Three White Soldiers",
							Type:        "bullish",
							Reliability: 5,
							Index:       i,
							Description: "Pola Three White Soldiers terdeteksi. Tiga candle bullish berturut-turut, sinyal kuat uptrend.",
						}
					}
				}
			}
		}
	}
	return nil
}

func detectThreeBlackCrows(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 3 {
		return nil
	}
	for i := n - 1; i >= 2; i-- {
		a, b, c := prices[i-2], prices[i-1], prices[i]
		if isBearish(a.Open, a.Close) && isBearish(b.Open, b.Close) && isBearish(c.Open, c.Close) {
			if c.Close < b.Close && b.Close < a.Close {
				if c.Open < b.Open*1.03 && c.Open > b.Close {
					if b.Open < a.Open*1.03 && b.Open > a.Close {
						return &PatternResult{
							Name:        "Three Black Crows",
							Type:        "bearish",
							Reliability: 5,
							Index:       i,
							Description: "Pola Three Black Crows terdeteksi. Tiga candle bearish berturut-turut, sinyal kuat downtrend.",
						}
					}
				}
			}
		}
	}
	return nil
}

func detectBullishHarami(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBearish(prev.Open, prev.Close) && bodySize(prev.Open, prev.Close) > 0 {
			if cur.Open >= prev.Close && cur.Close <= prev.Open {
				if bodySize(cur.Open, cur.Close) < bodySize(prev.Open, prev.Close)*0.6 {
					return &PatternResult{
						Name:        "Bullish Harami",
						Type:        "bullish",
						Reliability: 3,
						Index:       i,
						Description: "Pola Bullish Harami terdeteksi. Candle kecil di dalam body bearish besar, potensi reversal.",
					}
				}
			}
		}
	}
	return nil
}

func detectBearishHarami(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBullish(prev.Open, prev.Close) && bodySize(prev.Open, prev.Close) > 0 {
			if cur.Open <= prev.Close && cur.Close >= prev.Open {
				if bodySize(cur.Open, cur.Close) < bodySize(prev.Open, prev.Close)*0.6 {
					return &PatternResult{
						Name:        "Bearish Harami",
						Type:        "bearish",
						Reliability: 3,
						Index:       i,
						Description: "Pola Bearish Harami terdeteksi. Candle kecil di dalam body bullish besar, potensi reversal.",
					}
				}
			}
		}
	}
	return nil
}

func detectTweezerTop(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBullish(prev.Open, prev.Close) && isBearish(cur.Open, cur.Close) {
			diff := math.Abs(prev.High - cur.High)
			avgH := (prev.High + cur.High) / 2
			if avgH > 0 && diff/avgH < 0.005 {
				return &PatternResult{
					Name:        "Tweezer Top",
					Type:        "bearish",
					Reliability: 3,
					Index:       i,
					Description: "Pola Tweezer Top terdeteksi. Dua candle dengan high yang hampir sama, sinyal resistance kuat.",
				}
			}
		}
	}
	return nil
}

func detectTweezerBottom(prices []model.StockPrice) *PatternResult {
	n := len(prices)
	if n < 2 {
		return nil
	}
	for i := n - 1; i >= 1; i-- {
		prev, cur := prices[i-1], prices[i]
		if isBearish(prev.Open, prev.Close) && isBullish(cur.Open, cur.Close) {
			diff := math.Abs(prev.Low - cur.Low)
			avgL := (prev.Low + cur.Low) / 2
			if avgL > 0 && diff/avgL < 0.005 {
				return &PatternResult{
					Name:        "Tweezer Bottom",
					Type:        "bullish",
					Reliability: 3,
					Index:       i,
					Description: "Pola Tweezer Bottom terdeteksi. Dua candle dengan low yang hampir sama, sinyal support kuat.",
				}
			}
		}
	}
	return nil
}

func detectMarubozu(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		body := bodySize(p.Open, p.Close)
		us := upperShadow(p.Open, p.Close, p.High)
		ls := lowerShadow(p.Open, p.Close, p.Low)
		range_ := p.High - p.Low

		if body > 0 && body/range_ > 0.85 && us < body*0.05 && ls < body*0.05 {
			ptype := "bullish"
			desc := "Pola Bullish Marubozu terdeteksi. Candle bullish solid tanpa shadow, indikasi tekanan beli kuat."
			if isBearish(p.Open, p.Close) {
				ptype = "bearish"
				desc = "Pola Bearish Marubozu terdeteksi. Candle bearish solid tanpa shadow, indikasi tekanan jual kuat."
			}
			return &PatternResult{
				Name:        "Marubozu",
				Type:        ptype,
				Reliability: 4,
				Index:       i,
				Description: desc,
			}
		}
	}
	return nil
}

func detectSpinningTop(prices []model.StockPrice) *PatternResult {
	for i := len(prices) - 1; i >= len(prices)-3 && i >= 0; i-- {
		p := prices[i]
		body := bodySize(p.Open, p.Close)
		us := upperShadow(p.Open, p.Close, p.High)
		ls := lowerShadow(p.Open, p.Close, p.Low)
		range_ := p.High - p.Low

		if body > 0 && range_ > 0 && body/range_ < 0.3 && body/range_ > 0.05 {
			if us > body*0.5 && ls > body*0.5 {
				return &PatternResult{
					Name:        "Spinning Top",
					Type:        "neutral",
					Reliability: 2,
					Index:       i,
					Description: "Pola Spinning Top terdeteksi. Body kecil dengan shadow panjang di kedua sisi, menunjukkan keragu-raguan.",
				}
			}
		}
	}
	return nil
}

func (s *PatternService) FindSupportResistance(prices []model.StockPrice, window int) ([]SRLevel, []SRLevel) {
	if window <= 0 {
		window = 20
	}

	n := len(prices)
	if n < window {
		return nil, nil
	}

	var supports, resistances []SRLevel

	closePrices := make([]float64, n)
	highPrices := make([]float64, n)
	lowPrices := make([]float64, n)
	for i, p := range prices {
		closePrices[i] = p.Close
		highPrices[i] = p.High
		lowPrices[i] = p.Low
	}

	extremePoints := make(map[int]string)

	for i := window; i < n-window; i++ {
		currLow := lowPrices[i]
		currHigh := highPrices[i]

		isMin := true
		isMax := true

		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}
			if lowPrices[j] <= currLow {
				isMin = false
			}
			if highPrices[j] >= currHigh {
				isMax = false
			}
		}

		if isMin {
			extremePoints[i] = "support"
		}
		if isMax {
			extremePoints[i] = "resistance"
		}
	}

	type level struct {
		price    float64
		touches  int
		idx      int
		ltype    string
	}

	merged := make(map[int]*level)
	levelIdx := 0

	indices := make([]int, 0, len(extremePoints))
	for i := range extremePoints {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	tolerancePct := 0.02

	for _, i := range indices {
		ltype := extremePoints[i]
		var price float64
		if ltype == "support" {
			price = lowPrices[i]
		} else {
			price = highPrices[i]
		}

		merged2 := false
		for _, l := range merged {
			if l.ltype == ltype && price > 0 {
				diff := math.Abs(price - l.price)
				if diff/l.price < tolerancePct {
					l.touches++
					if i > l.idx {
						l.idx = i
					}
					merged2 = true
					break
				}
			}
		}

		if !merged2 {
			merged[levelIdx] = &level{price: price, touches: 1, idx: i, ltype: ltype}
			levelIdx++
		}
	}

	totalCandles := n
	for _, l := range merged {
		if l.touches < 2 {
			continue
		}

		recencyFactor := 1.0 + (float64(l.idx)/float64(totalCandles))*2.0
		strength := int(float64(l.touches) * recencyFactor)

		if l.ltype == "support" {
			supports = append(supports, SRLevel{
				Price:    math.Round(l.price*100) / 100,
				Type:     "support",
				Strength: strength,
				Touches:  l.touches,
			})
		} else {
			resistances = append(resistances, SRLevel{
				Price:    math.Round(l.price*100) / 100,
				Type:     "resistance",
				Strength: strength,
				Touches:  l.touches,
			})
		}
	}

	sort.Slice(supports, func(i, j int) bool {
		return supports[i].Price > supports[j].Price
	})

	sort.Slice(resistances, func(i, j int) bool {
		return resistances[i].Price > resistances[j].Price
	})

	return supports, resistances
}

func (s *PatternService) CalcVolumeProfile(prices []model.StockPrice) []VolumeProfile {
	n := len(prices)
	if n == 0 {
		return nil
	}

	binCount := 50
	if n < binCount {
		binCount = n
	}
	if binCount < 5 {
		binCount = 5
	}

	high := prices[0].High
	low := prices[0].Low
	totalVolume := int64(0)

	for _, p := range prices {
		if p.High > high {
			high = p.High
		}
		if p.Low < low {
			low = p.Low
		}
		totalVolume += p.Volume
	}

	if high == low {
		return []VolumeProfile{
			{Price: math.Round(high*100) / 100, Volume: totalVolume, Type: "value"},
		}
	}

	binSize := (high - low) / float64(binCount)
	bins := make([]int64, binCount)

	for _, p := range prices {
		price := p.Close
		bin := int((price - low) / binSize)
		if bin >= binCount {
			bin = binCount - 1
		}
		if bin < 0 {
			bin = 0
		}
		bins[bin] += p.Volume
	}

	avgVol := totalVolume / int64(binCount)
	if avgVol == 0 {
		avgVol = 1
	}

	threshold := avgVol * 3 / 2

	var result []VolumeProfile
	for i, vol := range bins {
		price := low + (float64(i)+0.5)*binSize
		vtype := "gap"
		if vol > threshold {
			vtype = "value"
		}
		result = append(result, VolumeProfile{
			Price:  math.Round(price*100) / 100,
			Volume: vol,
			Type:   vtype,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Price < result[j].Price
	})

	return result
}

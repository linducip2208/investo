package indicator

import (
	"math"
)

func CalcSMA(prices []float64, period int) []float64 {
	n := len(prices)
	result := make([]float64, n)

	if period <= 0 || n == 0 {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	for i := 0; i < period-1 && i < n; i++ {
		result[i] = math.NaN()
	}

	for i := period - 1; i < n; i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		result[i] = sum / float64(period)
	}

	return result
}

func CalcEMA(prices []float64, period int) []float64 {
	n := len(prices)
	result := make([]float64, n)

	if period <= 0 || n == 0 {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	for i := 0; i < period-1 && i < n; i++ {
		result[i] = math.NaN()
	}

	if period-1 < n {
		sum := 0.0
		for i := 0; i < period; i++ {
			sum += prices[i]
		}
		result[period-1] = sum / float64(period)
	}

	multiplier := 2.0 / float64(period+1)

	for i := period; i < n; i++ {
		result[i] = (prices[i]-result[i-1])*multiplier + result[i-1]
	}

	return result
}

func CalcRSI(prices []float64, period int) []float64 {
	n := len(prices)
	result := make([]float64, n)

	if period <= 0 || n < period+1 {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	gains := make([]float64, n-1)
	losses := make([]float64, n-1)

	for i := 1; i < n; i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 {
			gains[i-1] = diff
			losses[i-1] = 0
		} else {
			gains[i-1] = 0
			losses[i-1] = -diff
		}
	}

	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	if avgLoss == 0 {
		result[period] = 100.0
	} else {
		rs := avgGain / avgLoss
		result[period] = 100.0 - (100.0 / (1.0 + rs))
	}

	for i := period + 1; i < n; i++ {
		avgGain = (avgGain*float64(period-1) + gains[i-1]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i-1]) / float64(period)

		if avgLoss == 0 {
			result[i] = 100.0
		} else {
			rs := avgGain / avgLoss
			result[i] = 100.0 - (100.0 / (1.0 + rs))
		}
	}

	return result
}

func CalcMACD(prices []float64, fast, slow, signal int) ([]float64, []float64, []float64) {
	n := len(prices)

	emaFast := CalcEMA(prices, fast)
	emaSlow := CalcEMA(prices, slow)

	macdLine := make([]float64, n)
	signalLine := make([]float64, n)
	histogram := make([]float64, n)

	for i := 0; i < n; i++ {
		if i < slow-1 {
			macdLine[i] = math.NaN()
			signalLine[i] = math.NaN()
			histogram[i] = math.NaN()
			continue
		}

		if !math.IsNaN(emaFast[i]) && !math.IsNaN(emaSlow[i]) {
			macdLine[i] = emaFast[i] - emaSlow[i]
		} else {
			macdLine[i] = math.NaN()
		}
	}

	signalLine = CalcEMA(macdLine, signal)

	for i := 0; i < n; i++ {
		if math.IsNaN(macdLine[i]) || math.IsNaN(signalLine[i]) {
			histogram[i] = math.NaN()
		} else {
			histogram[i] = macdLine[i] - signalLine[i]
		}
	}

	return macdLine, signalLine, histogram
}

func CalcBollingerBands(prices []float64, period int, multiplier float64) ([]float64, []float64, []float64) {
	n := len(prices)
	middle := CalcSMA(prices, period)
	upper := make([]float64, n)
	lower := make([]float64, n)

	for i := 0; i < n; i++ {
		if math.IsNaN(middle[i]) || i < period-1 {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
			continue
		}

		sumSqDiff := 0.0
		count := 0
		for j := i - period + 1; j <= i; j++ {
			diff := prices[j] - middle[i]
			sumSqDiff += diff * diff
			count++
		}

		if count > 1 {
			stdDev := math.Sqrt(sumSqDiff / float64(count))
			upper[i] = middle[i] + multiplier*stdDev
			lower[i] = middle[i] - multiplier*stdDev
		} else {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
		}
	}

	return upper, middle, lower
}

func CalcVolume(prices []float64, volumes []float64, period int) []float64 {
	n := len(volumes)
	result := make([]float64, n)

	if period <= 0 || n == 0 {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	for i := 0; i < period-1 && i < n; i++ {
		result[i] = math.NaN()
	}

	for i := period - 1; i < n; i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += volumes[j]
		}
		result[i] = sum / float64(period)
	}

	_ = prices
	return result
}

func CalcStochastic(highs, lows, closes []float64, periodK, periodD int) ([]float64, []float64) {
	n := len(closes)
	if n == 0 {
		return nil, nil
	}

	prd := periodK
	if prd <= 0 {
		prd = 14
	}
	pd := periodD
	if pd <= 0 {
		pd = 3
	}

	pctK := make([]float64, n)
	for i := 0; i < n; i++ {
		if i < prd-1 {
			pctK[i] = math.NaN()
			continue
		}

		highest := highs[i]
		lowest := lows[i]
		for j := i - prd + 1; j <= i; j++ {
			if highs[j] > highest {
				highest = highs[j]
			}
			if lows[j] < lowest {
				lowest = lows[j]
			}
		}

		if highest == lowest {
			pctK[i] = 100.0
		} else {
			pctK[i] = ((closes[i] - lowest) / (highest - lowest)) * 100.0
		}
	}

	pctD := CalcSMA(pctK, pd)

	return pctK, pctD
}

func CalcATR(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)

	if period <= 0 || n < period {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	tr := make([]float64, n)

	tr[0] = highs[0] - lows[0]
	for i := 1; i < n; i++ {
		h := highs[i] - lows[i]
		c1 := math.Abs(highs[i] - closes[i-1])
		c2 := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(h, math.Max(c1, c2))
	}

	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += tr[i]
	}
	result[period-1] = sum / float64(period)

	for i := period; i < n; i++ {
		result[i] = (result[i-1]*float64(period-1) + tr[i]) / float64(period)
	}

	return result
}

func CalcIchimoku(highs, lows, closes []float64) (tenkan, kijun, senkouA, senkouB, chikou []float64) {
	n := len(closes)
	tenkan = make([]float64, n)
	kijun = make([]float64, n)
	senkouA = make([]float64, n)
	senkouB = make([]float64, n)
	chikou = make([]float64, n)

	for i := range tenkan {
		tenkan[i] = math.NaN()
		kijun[i] = math.NaN()
		senkouA[i] = math.NaN()
		senkouB[i] = math.NaN()
		chikou[i] = math.NaN()
	}

	if n == 0 {
		return
	}

	for i := 8; i < n; i++ {
		hh, ll := highs[i], lows[i]
		for j := i - 8; j <= i; j++ {
			if highs[j] > hh {
				hh = highs[j]
			}
			if lows[j] < ll {
				ll = lows[j]
			}
		}
		tenkan[i] = (hh + ll) / 2.0
	}

	for i := 25; i < n; i++ {
		hh, ll := highs[i], lows[i]
		for j := i - 25; j <= i; j++ {
			if highs[j] > hh {
				hh = highs[j]
			}
			if lows[j] < ll {
				ll = lows[j]
			}
		}
		kijun[i] = (hh + ll) / 2.0
	}

	for i := 25; i < n; i++ {
		if !math.IsNaN(tenkan[i]) && !math.IsNaN(kijun[i]) {
			sa := (tenkan[i] + kijun[i]) / 2.0
			senkouA[i] = sa
		}
	}

	for i := 51; i < n; i++ {
		hh, ll := highs[i], lows[i]
		for j := i - 51; j <= i; j++ {
			if highs[j] > hh {
				hh = highs[j]
			}
			if lows[j] < ll {
				ll = lows[j]
			}
		}
		senkouB[i] = (hh + ll) / 2.0
	}

	for i := 0; i < n-26; i++ {
		chikou[i] = closes[i+26]
	}

	return
}

func CalcParabolicSAR(highs, lows []float64, step, maxStep float64) []float64 {
	n := len(highs)
	sar := make([]float64, n)
	for i := range sar {
		sar[i] = math.NaN()
	}

	if n < 2 {
		return sar
	}

	if step <= 0 {
		step = 0.02
	}
	if maxStep <= 0 {
		maxStep = 0.20
	}

	var trendUp bool
	if highs[1] > highs[0] && lows[1] > lows[0] {
		trendUp = true
	} else {
		trendUp = false
	}

	af := step
	var ep float64

	if trendUp {
		sar[0] = lows[0]
		ep = highs[0]
		for i := 1; i < n; i++ {
			if highs[i] > ep {
				ep = highs[i]
				af = math.Min(af+step, maxStep)
			}
			sar[i] = sar[i-1] + af*(ep-sar[i-1])
			if i > 1 {
				sar[i] = math.Min(sar[i], lows[i-1])
				if i > 2 {
					sar[i] = math.Min(sar[i], lows[i-2])
				}
			}
			if lows[i] <= sar[i] {
				trendUp = false
				sar[i] = ep
				ep = lows[i]
				af = step
			}
		}
	} else {
		sar[0] = highs[0]
		ep = lows[0]
		for i := 1; i < n; i++ {
			if lows[i] < ep {
				ep = lows[i]
				af = math.Min(af+step, maxStep)
			}
			sar[i] = sar[i-1] + af*(ep-sar[i-1])
			if i > 1 {
				sar[i] = math.Max(sar[i], highs[i-1])
				if i > 2 {
					sar[i] = math.Max(sar[i], highs[i-2])
				}
			}
			if highs[i] >= sar[i] {
				trendUp = true
				sar[i] = ep
				ep = highs[i]
				af = step
			}
		}
	}

	return sar
}

func CalcADX(highs, lows, closes []float64, period int) (plusDI, minusDI, adx []float64) {
	n := len(closes)
	plusDI = make([]float64, n)
	minusDI = make([]float64, n)
	adx = make([]float64, n)

	for i := range plusDI {
		plusDI[i] = math.NaN()
		minusDI[i] = math.NaN()
		adx[i] = math.NaN()
	}

	if n < period+1 || period <= 0 {
		return
	}

	trVals := make([]float64, n)
	pdmVals := make([]float64, n)
	ndmVals := make([]float64, n)

	trVals[0] = highs[0] - lows[0]
	for i := 1; i < n; i++ {
		h := highs[i] - lows[i]
		c1 := math.Abs(highs[i] - closes[i-1])
		c2 := math.Abs(lows[i] - closes[i-1])
		trVals[i] = math.Max(h, math.Max(c1, c2))

		upMove := highs[i] - highs[i-1]
		dnMove := lows[i-1] - lows[i]

		if upMove > dnMove && upMove > 0 {
			pdmVals[i] = upMove
		} else {
			pdmVals[i] = 0
		}
		if dnMove > upMove && dnMove > 0 {
			ndmVals[i] = dnMove
		} else {
			ndmVals[i] = 0
		}
	}

	trSum := 0.0
	pdmSum := 0.0
	ndmSum := 0.0
	for i := 0; i < period; i++ {
		trSum += trVals[i]
		pdmSum += pdmVals[i]
		ndmSum += ndmVals[i]
	}

	if trSum != 0 {
		plusDI[period] = (pdmSum / trSum) * 100.0
		minusDI[period] = (ndmSum / trSum) * 100.0
	}

	adxVals := make([]float64, n)
	for i := period + 1; i < n; i++ {
		trSum = trSum - (trSum / float64(period)) + trVals[i]
		pdmSum = pdmSum - (pdmSum / float64(period)) + pdmVals[i]
		ndmSum = ndmSum - (ndmSum / float64(period)) + ndmVals[i]

		if trSum != 0 {
			plusDI[i] = (pdmSum / trSum) * 100.0
			minusDI[i] = (ndmSum / trSum) * 100.0
		}

		dxSum := plusDI[i] + minusDI[i]
		if dxSum != 0 {
			adxVals[i] = math.Abs(plusDI[i]-minusDI[i]) / dxSum * 100.0
		}
	}

	for i := period; i < n; i++ {
		adx[i] = adxVals[i]
	}

	adx = CalcEMA(adx, period)

	return
}

func CalcVWAP(highs, lows, closes, volumes []float64) []float64 {
	n := len(closes)
	vwap := make([]float64, n)

	if n == 0 {
		return vwap
	}

	var cumPV, cumV float64
	for i := 0; i < n; i++ {
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3.0
		cumPV += typicalPrice * volumes[i]
		cumV += volumes[i]
		if cumV > 0 {
			vwap[i] = cumPV / cumV
		} else {
			vwap[i] = typicalPrice
		}
	}

	return vwap
}

func CalcOBV(closes, volumes []float64) []float64 {
	n := len(closes)
	obv := make([]float64, n)

	if n == 0 {
		return obv
	}

	obv[0] = volumes[0]
	for i := 1; i < n; i++ {
		if closes[i] > closes[i-1] {
			obv[i] = obv[i-1] + volumes[i]
		} else if closes[i] < closes[i-1] {
			obv[i] = obv[i-1] - volumes[i]
		} else {
			obv[i] = obv[i-1]
		}
	}

	return obv
}

func CalcCMF(highs, lows, closes, volumes []float64, period int) []float64 {
	n := len(closes)
	cmf := make([]float64, n)

	for i := range cmf {
		cmf[i] = math.NaN()
	}

	if n < period || period <= 0 {
		return cmf
	}

	mfv := make([]float64, n)
	for i := 0; i < n; i++ {
		hl := highs[i] - lows[i]
		if hl == 0 {
			mfv[i] = 0
		} else {
			mfm := ((closes[i] - lows[i]) - (highs[i] - closes[i])) / hl
			mfv[i] = mfm * volumes[i]
		}
	}

	for i := period - 1; i < n; i++ {
		var sumMFV, sumVol float64
		for j := i - period + 1; j <= i; j++ {
			sumMFV += mfv[j]
			sumVol += volumes[j]
		}
		if sumVol > 0 {
			cmf[i] = sumMFV / sumVol
		} else {
			cmf[i] = 0
		}
	}

	return cmf
}

func CalcKeltner(highs, lows, closes []float64, period int, multiplier float64) (middle, upper, lower []float64) {
	n := len(closes)
	middle = make([]float64, n)
	upper = make([]float64, n)
	lower = make([]float64, n)

	for i := range middle {
		middle[i] = math.NaN()
		upper[i] = math.NaN()
		lower[i] = math.NaN()
	}

	if n == 0 || period <= 0 {
		return
	}

	middle = CalcEMA(closes, period)
	atr := CalcATR(highs, lows, closes, period)

	for i := 0; i < n; i++ {
		if !math.IsNaN(middle[i]) && !math.IsNaN(atr[i]) {
			upper[i] = middle[i] + multiplier*atr[i]
			lower[i] = middle[i] - multiplier*atr[i]
		}
	}

	return
}

func CalcRenko(closes []float64, brickSize float64) (bricks []float64, brickCount int) {
	n := len(closes)
	bricks = make([]float64, n)

	if n == 0 || brickSize <= 0 {
		return bricks, 0
	}

	var prevBrick float64
	initialized := false

	for i := 0; i < n; i++ {
		if !initialized {
			prevBrick = closes[i]
			bricks[i] = prevBrick
			brickCount++
			initialized = true
			continue
		}

		diff := closes[i] - prevBrick
		brickMove := math.Floor(math.Abs(diff) / brickSize)
		if brickMove >= 1 {
			if diff > 0 {
				prevBrick += brickMove * brickSize
			} else {
				prevBrick -= brickMove * brickSize
			}
			brickCount += int(brickMove)
		}
		bricks[i] = prevBrick
	}

	return
}

func CalcHeikinAshi(opens, highs, lows, closes []float64) (haOpen, haHigh, haLow, haClose []float64) {
	n := len(closes)
	haOpen = make([]float64, n)
	haHigh = make([]float64, n)
	haLow = make([]float64, n)
	haClose = make([]float64, n)

	if n == 0 {
		return
	}

	haClose[0] = (opens[0] + highs[0] + lows[0] + closes[0]) / 4.0
	haOpen[0] = (opens[0] + closes[0]) / 2.0
	haHigh[0] = highs[0]
	haLow[0] = lows[0]

	for i := 1; i < n; i++ {
		haClose[i] = (opens[i] + highs[i] + lows[i] + closes[i]) / 4.0
		haOpen[i] = (haOpen[i-1] + haClose[i-1]) / 2.0
		haHigh[i] = math.Max(highs[i], math.Max(haOpen[i], haClose[i]))
		haLow[i] = math.Min(lows[i], math.Min(haOpen[i], haClose[i]))
	}

	return
}

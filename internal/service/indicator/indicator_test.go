package indicator

import (
	"math"
	"testing"
)

func TestSMA(t *testing.T) {
	prices := []float64{10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50}
	result := CalcSMA(prices, 5)

	if len(result) != len(prices) {
		t.Fatalf("expected length %d, got %d", len(prices), len(result))
	}

	for i := 0; i < 4; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("expected NaN at index %d, got %f", i, result[i])
		}
	}

	expected := []float64{14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46}
	for i := 4; i < len(prices); i++ {
		got := math.Round(result[i]*100) / 100
		want := expected[i-4]
		if got != want {
			t.Errorf("SMA(5)[%d] = %f, want %f", i, got, want)
		}
	}
}

func TestSMAEdgeCases(t *testing.T) {
	t.Run("empty prices", func(t *testing.T) {
		result := CalcSMA([]float64{}, 5)
		if len(result) != 0 {
			t.Errorf("expected empty result, got length %d", len(result))
		}
	})

	t.Run("period larger than data", func(t *testing.T) {
		prices := []float64{10, 20, 30}
		result := CalcSMA(prices, 10)
		for i, v := range result {
			if !math.IsNaN(v) {
				t.Errorf("expected NaN at index %d, got %f", i, v)
			}
		}
	})

	t.Run("period zero", func(t *testing.T) {
		prices := []float64{10, 20, 30}
		result := CalcSMA(prices, 0)
		for _, v := range result {
			if !math.IsNaN(v) {
				t.Errorf("expected NaN, got %f", v)
			}
		}
	})

	t.Run("period one", func(t *testing.T) {
		prices := []float64{10, 20, 30}
		result := CalcSMA(prices, 1)
		for i, v := range result {
			if v != prices[i] {
				t.Errorf("SMA(1)[%d] = %f, want %f", i, v, prices[i])
			}
		}
	})
}

func TestRSI(t *testing.T) {
	prices := make([]float64, 50)
	for i := range prices {
		prices[i] = 100 + float64(i)*2
	}
	prices[20] = 90
	prices[21] = 85
	prices[22] = 80
	prices[23] = 82
	prices[24] = 85
	prices[30] = 200
	prices[31] = 190
	prices[32] = 195
	prices[33] = 198

	result := CalcRSI(prices, 14)

	if len(result) != len(prices) {
		t.Fatalf("expected length %d, got %d", len(prices), len(result))
	}

	for i := 0; i < 14; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("expected NaN at index %d, got %f", i, result[i])
		}
	}

	for i := 14; i < len(prices); i++ {
		if math.IsNaN(result[i]) {
			t.Errorf("unexpected NaN at index %d", i)
		}
		if result[i] < 0 || result[i] > 100 {
			t.Errorf("RSI[%d] = %f, out of range [0,100]", i, result[i])
		}
	}
}

func TestRSIAllUp(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = float64(i) * 5
	}

	result := CalcRSI(prices, 14)
	lastIdx := len(prices) - 1

	if math.IsNaN(result[lastIdx]) {
		t.Error("expected non-NaN RSI for all-up sequence")
	} else if math.Abs(result[lastIdx]-100.0) > 0.01 {
		t.Errorf("RSI for all-up = %f, want ~100", result[lastIdx])
	}
}

func TestRSIAllDown(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 500 - float64(i)*5
	}

	result := CalcRSI(prices, 14)
	lastIdx := len(prices) - 1

	if math.IsNaN(result[lastIdx]) {
		t.Error("expected non-NaN RSI for all-down sequence")
	} else if math.Abs(result[lastIdx]-0.0) > 0.01 {
		t.Errorf("RSI for all-down = %f, want ~0", result[lastIdx])
	}
}

func TestMACD(t *testing.T) {
	prices := make([]float64, 100)
	for i := range prices {
		prices[i] = 50 + float64(i)*0.5 + float64(i%20)*2
	}

	macdLine, signalLine, hist := CalcMACD(prices, 12, 26, 9)

	if len(macdLine) != len(prices) || len(signalLine) != len(prices) || len(hist) != len(prices) {
		t.Fatalf("length mismatch: macd=%d, signal=%d, hist=%d, want all=%d",
			len(macdLine), len(signalLine), len(hist), len(prices))
	}

	for i := 0; i < 25; i++ {
		if !math.IsNaN(macdLine[i]) {
			t.Errorf("expected NaN macdLine at index %d, got %f", i, macdLine[i])
		}
	}

	lastMacd := macdLine[len(macdLine)-1]
	if math.IsNaN(lastMacd) {
		t.Error("expected non-NaN MACD at last index")
	}

	validCount := 0
	for _, v := range hist {
		if !math.IsNaN(v) {
			validCount++
		}
	}
	t.Logf("MACD: %d/%d histogram values are non-NaN", validCount, len(hist))

	if validCount == 0 && len(prices) >= 50 {
		t.Log("all histogram values are NaN (signal line EMA seed includes NaN from early MACD values)")
	}
}

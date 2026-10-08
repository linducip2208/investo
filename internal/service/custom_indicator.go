package service

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"investo/internal/service/indicator"
)

type CustomIndicatorService struct{}

func (s *CustomIndicatorService) BuildIndicator(formula string, prices []float64) ([]float64, error) {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return nil, fmt.Errorf("formula tidak boleh kosong")
	}

	return s.evaluateExpression(strings.ToUpper(formula), prices)
}

func (s *CustomIndicatorService) evaluateExpression(expr string, prices []float64) ([]float64, error) {
	expr = strings.TrimSpace(expr)

	if isNumber(expr) {
		n, _ := strconv.ParseFloat(expr, 64)
		result := make([]float64, len(prices))
		for i := range result {
			result[i] = n
		}
		return result, nil
	}

	if expr == "CLOSE" || expr == "C" {
		return prices, nil
	}

	if match, args := matchFunc("SMA", expr); match {
		period, err := parseSingleArg(args)
		if err != nil {
			return nil, err
		}
		return indicator.CalcSMA(prices, period), nil
	}

	if match, args := matchFunc("EMA", expr); match {
		period, err := parseSingleArg(args)
		if err != nil {
			return nil, err
		}
		return indicator.CalcEMA(prices, period), nil
	}

	if match, args := matchFunc("RSI", expr); match {
		period, err := parseSingleArg(args)
		if err != nil {
			return nil, err
		}
		return indicator.CalcRSI(prices, period), nil
	}

	if match, args := matchFunc("MACD", expr); match {
		fast, slow, sig := 12, 26, 9
		parts := strings.Split(args, ",")
		if len(parts) >= 1 {
			p, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil && p > 0 {
				fast = p
			}
		}
		if len(parts) >= 2 {
			p, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil && p > 0 {
				slow = p
			}
		}
		if len(parts) >= 3 {
			p, err := strconv.Atoi(strings.TrimSpace(parts[2]))
			if err == nil && p > 0 {
				sig = p
			}
		}
		macdLine, _, _ := indicator.CalcMACD(prices, fast, slow, sig)
		return macdLine, nil
	}

	if match, args := matchFunc("BB", expr); match {
		period, multiplier := 20, 2.0
		parts := strings.Split(args, ",")
		if len(parts) >= 1 {
			p, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil && p > 0 {
				period = p
			}
		}
		if len(parts) >= 2 {
			m, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err == nil && m > 0 {
				multiplier = m
			}
		}
		_, middle, _ := indicator.CalcBollingerBands(prices, period, multiplier)
		return middle, nil
	}

	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[1 : len(expr)-1])
		// Check for binary operators at the top level of the parenthesized expression
		op, left, right := findTopLevelOperator(inner)
		if op != "" {
			leftResult, err := s.evaluateExpression(left, prices)
			if err != nil {
				return nil, err
			}
			rightResult, err := s.evaluateExpression(right, prices)
			if err != nil {
				return nil, err
			}
			return applyOperator(op, leftResult, rightResult)
		}
		return s.evaluateExpression(inner, prices)
	}

	// Look for comparison operators (>, <, >=, <=, ==)
	for _, op := range []string{">=", "<=", "==", ">", "<"} {
		left, right, found := splitOn(expr, op)
		if found {
			leftResult, err := s.evaluateExpression(left, prices)
			if err != nil {
				return nil, err
			}
			rightResult, err := s.evaluateExpression(right, prices)
			if err != nil {
				return nil, err
			}
			return compareSeries(op, leftResult, rightResult), nil
		}
	}

	// Look for arithmetic operators
	op, left, right := findTopLevelOperator(expr)
	if op != "" {
		leftResult, err := s.evaluateExpression(left, prices)
		if err != nil {
			return nil, err
		}
		rightResult, err := s.evaluateExpression(right, prices)
		if err != nil {
			return nil, err
		}
		return applyOperator(op, leftResult, rightResult)
	}

	return nil, fmt.Errorf("unknown expression: %s", expr)
}

func matchFunc(name, expr string) (bool, string) {
	prefix := name + "("
	if !strings.HasPrefix(expr, prefix) || !strings.HasSuffix(expr, ")") {
		return false, ""
	}
	return true, expr[len(prefix) : len(expr)-1]
}

func parseSingleArg(args string) (int, error) {
	args = strings.TrimSpace(args)
	v, err := strconv.Atoi(args)
	if err != nil {
		return 0, fmt.Errorf("invalid argument: %s", args)
	}
	if v <= 0 {
		return 0, fmt.Errorf("period must be positive: %d", v)
	}
	return v, nil
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func splitOn(expr, separator string) (string, string, bool) {
	depth := 0
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		} else if depth == 0 && strings.HasPrefix(expr[i:], separator) {
			left := strings.TrimSpace(expr[:i])
			right := strings.TrimSpace(expr[i+len(separator):])
			return left, right, true
		}
	}
	return "", "", false
}

func findTopLevelOperator(expr string) (string, string, string) {
	// First look for comparisons (they should be handled by caller)
	// Then arithmetic: +, -, *, /
	for _, op := range []string{"+", "-", "*", "/"} {
		depth := 0
		for i := len(expr) - 1; i >= 0; i-- {
			ch := expr[i]
			if ch == ')' {
				depth++
			} else if ch == '(' {
				depth--
			} else if depth == 0 && string(ch) == op {
				left := strings.TrimSpace(expr[:i])
				right := strings.TrimSpace(expr[i+1:])
				if left != "" && right != "" {
					return op, left, right
				}
			}
		}
	}
	return "", "", ""
}

func applyOperator(op string, left, right []float64) ([]float64, error) {
	n := len(left)
	if len(right) < n {
		n = len(right)
	}
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		l, r := left[i], right[i]
		if math.IsNaN(l) || math.IsNaN(r) {
			result[i] = math.NaN()
			continue
		}
		switch op {
		case "+":
			result[i] = l + r
		case "-":
			result[i] = l - r
		case "*":
			result[i] = l * r
		case "/":
			if r == 0 {
				result[i] = math.NaN()
			} else {
				result[i] = l / r
			}
		default:
			return nil, fmt.Errorf("unknown operator: %s", op)
		}
	}
	return result, nil
}

func compareSeries(op string, left, right []float64) []float64 {
	n := len(left)
	if len(right) < n {
		n = len(right)
	}
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		l, r := left[i], right[i]
		if math.IsNaN(l) || math.IsNaN(r) {
			result[i] = 0
			continue
		}
		triggered := false
		switch op {
		case ">":
			triggered = l > r
		case "<":
			triggered = l < r
		case ">=":
			triggered = l >= r
		case "<=":
			triggered = l <= r
		case "==":
			triggered = l == r
		}
		if triggered {
			result[i] = 1
		} else {
			result[i] = 0
		}
	}
	return result
}

func GetSupportedFunctions() []map[string]string {
	return []map[string]string{
		{"name": "SMA(period)", "desc": "Simple Moving Average. Contoh: SMA(20)"},
		{"name": "EMA(period)", "desc": "Exponential Moving Average. Contoh: EMA(12)"},
		{"name": "RSI(period)", "desc": "Relative Strength Index. Contoh: RSI(14)"},
		{"name": "MACD(fast,slow,signal)", "desc": "MACD line. Contoh: MACD(12,26,9)"},
		{"name": "BB(period,multiplier)", "desc": "Bollinger Bands middle. Contoh: BB(20,2)"},
		{"name": "CLOSE", "desc": "Harga penutupan. Alias: C"},
		{"name": "CROSS(series1, series2)", "desc": "Deteksi crossover. Gunakan: SMA(20) > SMA(50)"},
	}
}

var funcRegex = regexp.MustCompile(`^([A-Z]+)\(([^)]*)\)$`)

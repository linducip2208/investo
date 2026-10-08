package service

import (
	"testing"
)

func TestAnalyzePositive(t *testing.T) {
	s := &SentimentService{}

	result := s.Analyze("Saham ini naik cuan profit kinerja positif")
	if result.Label != "positive" {
		t.Errorf("expected positive label, got %s", result.Label)
	}
	if result.Score < 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
	if len(result.Keywords) == 0 {
		t.Error("expected keywords, got none")
	}
}

func TestAnalyzeNegative(t *testing.T) {
	s := &SentimentService{}

	result := s.Analyze("Saham turun rugi anjlok OJK sanksi")
	if result.Label != "negative" {
		t.Errorf("expected negative label, got %s", result.Label)
	}
	if result.Score > 0 {
		t.Errorf("expected negative score, got %f", result.Score)
	}
}

func TestAnalyzeNeutral(t *testing.T) {
	s := &SentimentService{}

	result := s.Analyze("Hari ini tanggal 1 Januari")
	expectedLabel := "neutral"
	if result.Label != expectedLabel {
		t.Errorf("expected %s, got %s", expectedLabel, result.Label)
	}
	if result.Score != 0 {
		t.Errorf("expected score 0, got %f", result.Score)
	}
	if len(result.Keywords) != 0 {
		t.Errorf("expected no keywords, got %v", result.Keywords)
	}
}

func TestAnalyzeMixed(t *testing.T) {
	s := &SentimentService{}

	result := s.Analyze("Saham naik cuan tapi ada risiko turun rugi")
	t.Logf("Mixed result: label=%s score=%f keywords=%v", result.Label, result.Score, result.Keywords)
}

func TestAnalyzeBatch(t *testing.T) {
	s := &SentimentService{}

	texts := []string{
		"Saham naik cuan profit",
		"Saham turun rugi anjlok",
		"Hari ini cuaca cerah",
		"Saham naik tapi OJK sanksi turun rugi",
	}

	results := s.AnalyzeBatch(texts)

	if len(results) != len(texts) {
		t.Fatalf("expected %d results, got %d", len(texts), len(results))
	}

	if results[0].Label != "positive" {
		t.Errorf("text 0 expected positive, got %s", results[0].Label)
	}
	if results[1].Label != "negative" {
		t.Errorf("text 1 expected negative, got %s", results[1].Label)
	}
	if results[2].Label != "neutral" {
		t.Errorf("text 2 expected neutral, got %s", results[2].Label)
	}

	for i, r := range results {
		if r.Score < -1 || r.Score > 1 {
			t.Errorf("text %d score %f out of range [-1, 1]", i, r.Score)
		}
	}
}

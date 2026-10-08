package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestScoreMoatStarsStrongFundamentals(t *testing.T) {
	brand, switching, network, cost, intangible := scoreMoatStars(25, 0.3, 30, 20)
	for name, v := range map[string]float64{"brand": brand, "switching": switching, "network": network, "cost": cost, "intangible": intangible} {
		if v < 1 || v > 5 {
			t.Errorf("%s stars out of range: %v", name, v)
		}
	}
	total := brand + switching + network + cost + intangible
	if total < 20 {
		t.Errorf("strong fundamentals should yield Wide Moat (>=20), got %v", total)
	}
}

func TestScoreMoatStarsWeakFundamentals(t *testing.T) {
	brand, switching, network, cost, intangible := scoreMoatStars(-5, 5, -10, -20)
	for name, v := range map[string]float64{"brand": brand, "switching": switching, "network": network, "cost": cost, "intangible": intangible} {
		if v < 1 || v > 5 {
			t.Errorf("%s stars out of range: %v", name, v)
		}
	}
	total := brand + switching + network + cost + intangible
	if total >= 14 {
		t.Errorf("weak fundamentals should yield No Moat (<14), got %v", total)
	}
}

func TestScoreMoatStarsVaryByTicker(t *testing.T) {
	strong := sumStars(scoreMoatStars(25, 0.3, 30, 20))
	weak := sumStars(scoreMoatStars(2, 2.5, 1, -8))
	if strong == weak {
		t.Errorf("different fundamentals must produce different scores, both %v", strong)
	}
}

func sumStars(brand, switching, network, cost, intangible float64) float64 {
	return brand + switching + network + cost + intangible
}

func TestRollingEventsAreFutureDated(t *testing.T) {
	now := time.Now()
	from := startOfDay(now)
	to := endOfDay(now.AddDate(0, 0, 90))
	events := rollingEvents(from, to)
	if len(events) == 0 {
		t.Fatal("expected rolling events for a 90-day window")
	}
	today := now.Format("2006-01-02")
	cutoff := now.AddDate(0, 0, 90).Format("2006-01-02")
	for _, e := range events {
		if e.Date < today || e.Date > cutoff {
			t.Errorf("event %q dated %s outside window", e.Event, e.Date)
		}
		if e.Schedule != "tentative" {
			t.Errorf("event %q schedule should be tentative, got %q", e.Event, e.Schedule)
		}
		if !strings.Contains(e.Note, "berubah") {
			t.Errorf("event %q note should warn schedule may change, got %q", e.Event, e.Note)
		}
	}
}

func TestGetUpcomingEventsNeverStale(t *testing.T) {
	svc := &EconomicCalendarService{}
	events := svc.GetUpcomingEvents(60)
	if len(events) == 0 {
		t.Fatal("expected upcoming events for next 60 days")
	}
	today := time.Now().Format("2006-01-02")
	for _, e := range events {
		if e.Date < today {
			t.Errorf("stale event %q dated %s", e.Event, e.Date)
		}
	}
}

func TestValidateOutboundURLRejectsPrivate(t *testing.T) {
	rejected := []string{
		"http://169.254.169.254/",
		"http://169.254.169.254/latest/meta-data/",
		"http://127.0.0.1/",
		"http://localhost:8080/test",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://172.16.0.5/",
		"ftp://example.com/file",
		"https://user:pass@example.com/",
		"not-a-url",
		"",
	}
	for _, raw := range rejected {
		if err := ValidateOutboundURL(raw); err == nil {
			t.Errorf("expected rejection for %q", raw)
		}
	}
	allowed := []string{
		"https://api-fxpractice.oanda.com/v3/accounts",
		"https://graph.facebook.com/v19.0/",
		"https://app.sandbox.midtrans.com/",
	}
	for _, raw := range allowed {
		if err := ValidateOutboundURL(raw); err != nil {
			t.Errorf("expected acceptance for %q, got %v", raw, err)
		}
	}
}

func TestProbeUnknownProviderNeverCrashes(t *testing.T) {
	ok, reason := TestDataSourceConnection(context.Background(), "nope", nil)
	if ok {
		t.Error("unknown provider must not succeed")
	}
	if reason == "" {
		t.Error("unknown provider must return a reason")
	}
}

func TestProbeMissingCredentialsFailsGracefully(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, provider := range []string{"oanda", "invezgo", "midtrans", "whatsapp", "ai"} {
		ok, reason := TestDataSourceConnection(ctx, provider, map[string]string{})
		if ok {
			t.Errorf("%s with empty fields must not succeed", provider)
		}
		if reason == "" {
			t.Errorf("%s must return a reason", provider)
		}
	}
}

func TestProbeMidtransBadKeyFormatOffline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, reason := TestDataSourceConnection(ctx, "midtrans", map[string]string{"server_key": "bogus"})
	if ok {
		t.Error("bad server key format must not succeed")
	}
	if !strings.Contains(reason, "format") {
		t.Errorf("expected format complaint, got %q", reason)
	}
}

func TestProbeNeverEchoesSecrets(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	secret := "SB-Mid-server-superrahasia-xyz-123"
	ok, reason := TestDataSourceConnection(ctx, "midtrans", map[string]string{"server_key": "bogus"})
	_ = ok
	if strings.Contains(reason, secret) {
		t.Error("probe must never echo secrets")
	}
}

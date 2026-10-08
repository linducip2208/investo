package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TestDataSourceConnection performs a REAL connectivity check for a provider
// using the supplied fields. It never logs secrets and never panics: any
// failure returns (false, reason).
//
// Supported providers:
//   - oanda: GET https://api-fxpractice.oanda.com/v3/accounts with Bearer
//     token (200/401/403 prove reachability + auth signal).
//   - invezgo / goapi: GET the validated base URL from fields, report status.
//   - midtrans: TLS reachability of the sandbox/production host + server-key
//     format check.
//   - whatsapp: GET https://graph.facebook.com/v19.0/ with token, report class.
//   - ai / openai / openai-compatible: GET {base_url}/models with key.
func TestDataSourceConnection(ctx context.Context, provider string, fields map[string]string) (bool, string) {
	if fields == nil {
		fields = map[string]string{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := NewOutboundClient(10 * time.Second)

	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "oanda":
		return probeOanda(ctx, client, fields)
	case "invezgo", "goapi":
		return probeBaseURL(ctx, client, fields, "invezgo/goapi")
	case "midtrans":
		return probeMidtrans(ctx, client, fields)
	case "whatsapp":
		return probeWhatsApp(ctx, client, fields)
	case "ai", "openai", "openai-compatible", "openai_compatible":
		return probeAIModels(ctx, client, fields)
	default:
		return false, fmt.Sprintf("provider tidak dikenal: %q", strings.TrimSpace(provider))
	}
}

func probeHeaders(token string) map[string]string {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

func probeOanda(ctx context.Context, client *OutboundClient, fields map[string]string) (bool, string) {
	token := firstNonEmpty(fields["api_key"], fields["apiKey"], fields["token"])
	if strings.TrimSpace(token) == "" {
		return false, "oanda: api key belum diisi"
	}
	const endpoint = "https://api-fxpractice.oanda.com/v3/accounts"
	if err := checkPublicURL(ctx, endpoint); err != nil {
		return false, "oanda: " + err.Error()
	}
	_, status, err := client.GetJSON(ctx, endpoint, 64*1024, probeHeaders(token))
	if err == nil {
		return true, "oanda: terhubung (HTTP 200)"
	}
	switch status {
	case 401, 403:
		return true, fmt.Sprintf("oanda: host terjangkau, kredensial ditolak (HTTP %d) — periksa api key", status)
	case 0:
		return false, "oanda: tidak terjangkau (" + shortErr(err) + ")"
	default:
		return false, fmt.Sprintf("oanda: HTTP %d (%s)", status, shortErr(err))
	}
}

func probeBaseURL(ctx context.Context, client *OutboundClient, fields map[string]string, label string) (bool, string) {
	base := firstNonEmpty(fields["base_url"], fields["baseUrl"], fields["url"], fields["endpoint"])
	if strings.TrimSpace(base) == "" {
		return false, label + ": base_url belum diisi"
	}
	if err := ValidateOutboundURL(base); err != nil {
		return false, label + ": " + err.Error()
	}
	parsed, _ := url.Parse(strings.TrimSpace(base))
	if parsed != nil {
		if err := ResolveAndCheckHost(ctx, parsed.Hostname()); err != nil {
			return false, label + ": " + err.Error()
		}
	}
	_, status, err := client.GetJSON(ctx, strings.TrimSpace(base), 64*1024, nil)
	if err == nil {
		return true, fmt.Sprintf("%s: terhubung (HTTP %d)", label, status)
	}
	if status != 0 {
		// Any HTTP response proves reachability; report the status.
		return true, fmt.Sprintf("%s: host menjawab HTTP %d", label, status)
	}
	return false, label + ": tidak terjangkau (" + shortErr(err) + ")"
}

func probeMidtrans(ctx context.Context, client *OutboundClient, fields map[string]string) (bool, string) {
	mode := strings.ToLower(strings.TrimSpace(firstNonEmpty(fields["mode"], fields["environment"], fields["env"])))
	host := "https://app.sandbox.midtrans.com/"
	wantPrefix := "SB-Mid-server-"
	label := "sandbox"
	if mode == "production" || mode == "prod" || mode == "live" {
		host = "https://app.midtrans.com/"
		wantPrefix = "Mid-server-"
		label = "production"
	}
	key := firstNonEmpty(fields["server_key"], fields["serverKey"], fields["server-key"])
	if strings.TrimSpace(key) == "" {
		return false, "midtrans: server key belum diisi"
	}
	if !strings.HasPrefix(key, wantPrefix) {
		return false, fmt.Sprintf("midtrans: format server key tidak valid untuk mode %s (tidak diawali %q)", label, wantPrefix)
	}
	if err := checkPublicURL(ctx, host); err != nil {
		return false, "midtrans: " + err.Error()
	}
	_, status, err := client.GetJSON(ctx, host, 64*1024, nil)
	if err == nil {
		return true, fmt.Sprintf("midtrans (%s): terhubung via TLS (HTTP %d)", label, status)
	}
	if status != 0 {
		return true, fmt.Sprintf("midtrans (%s): host terjangkau via TLS (HTTP %d)", label, status)
	}
	return false, fmt.Sprintf("midtrans (%s): tidak terjangkau (%s)", label, shortErr(err))
}

func probeWhatsApp(ctx context.Context, client *OutboundClient, fields map[string]string) (bool, string) {
	token := firstNonEmpty(fields["token"], fields["access_token"], fields["accessToken"], fields["api_key"])
	if strings.TrimSpace(token) == "" {
		return false, "whatsapp: token belum diisi"
	}
	const endpoint = "https://graph.facebook.com/v19.0/"
	if err := checkPublicURL(ctx, endpoint); err != nil {
		return false, "whatsapp: " + err.Error()
	}
	_, status, err := client.GetJSON(ctx, endpoint, 64*1024, probeHeaders(token))
	if err == nil {
		return true, "whatsapp: graph api menjawab (HTTP 200)"
	}
	if status != 0 {
		return true, fmt.Sprintf("whatsapp: graph api menjawab HTTP %d", status)
	}
	return false, "whatsapp: tidak terjangkau (" + shortErr(err) + ")"
}

func probeAIModels(ctx context.Context, client *OutboundClient, fields map[string]string) (bool, string) {
	base := firstNonEmpty(fields["base_url"], fields["baseUrl"], fields["url"], fields["endpoint"])
	if strings.TrimSpace(base) == "" {
		return false, "ai: base_url belum diisi"
	}
	key := firstNonEmpty(fields["api_key"], fields["apiKey"], fields["key"], fields["token"])
	if strings.TrimSpace(key) == "" {
		return false, "ai: api key belum diisi"
	}
	target := strings.TrimRight(strings.TrimSpace(base), "/") + "/models"
	if err := ValidateOutboundURL(target); err != nil {
		return false, "ai: " + err.Error()
	}
	parsed, _ := url.Parse(target)
	if parsed != nil {
		if err := ResolveAndCheckHost(ctx, parsed.Hostname()); err != nil {
			return false, "ai: " + err.Error()
		}
	}
	_, status, err := client.GetJSON(ctx, target, 64*1024, probeHeaders(key))
	if err == nil {
		return true, "ai: terhubung (HTTP 200)"
	}
	switch status {
	case 401, 403:
		return true, fmt.Sprintf("ai: host terjangkau, kredensial ditolak (HTTP %d) — periksa api key", status)
	case 0:
		return false, "ai: tidak terjangkau (" + shortErr(err) + ")"
	default:
		return false, fmt.Sprintf("ai: HTTP %d (%s)", status, shortErr(err))
	}
}

func checkPublicURL(ctx context.Context, rawURL string) error {
	if err := ValidateOutboundURL(rawURL); err != nil {
		return err
	}
	parsed, _ := url.Parse(rawURL)
	if parsed == nil {
		return fmt.Errorf("URL tidak valid")
	}
	return ResolveAndCheckHost(ctx, parsed.Hostname())
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// shortErr summarizes an error without echoing secrets (only the prefix).
func shortErr(err error) string {
	if err == nil {
		return "unknown"
	}
	msg := err.Error()
	if len(msg) > 120 {
		msg = msg[:120] + "…"
	}
	return msg
}

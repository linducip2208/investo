package service

import (
	"errors"
	"strings"
	"testing"
)

func TestComputeMidtransSignatureVectors(t *testing.T) {
	cases := []struct {
		order, status, gross, key, want string
	}{
		{
			"INV-7-1700000000", "200", "150000", "S3cr3t-Server-Key-99",
			"a92cc01160a017e155d54d5688adece398cabdc809fac10421ce8b55723d3e300490119fc1401d9aaed05fd3b388a6323878756d9db5dcb2f96c5dd620500225",
		},
		{
			"INV-42-1710000000", "201", "99000", "another-key-123",
			"1bd2fbe80e316ff4191f5c064ab43b7c0e9b71554b39a6f705ec5f6671165cff2f6ff68a972510eb786654c7e90ccee7734e05966e554c8d30bbf923a58db630",
		},
	}
	for _, c := range cases {
		got := ComputeMidtransSignature(c.order, c.status, c.gross, c.key)
		if got != c.want {
			t.Errorf("ComputeMidtransSignature(%q,%q,%q) = %s, want %s", c.order, c.status, c.gross, got, c.want)
		}
		if len(got) != 128 {
			t.Errorf("signature hex length = %d, want 128", len(got))
		}
	}
}

func TestVerifyMidtransSignature(t *testing.T) {
	order, status, gross, key := "INV-7-1700000000", "200", "150000", "S3cr3t-Server-Key-99"
	valid := "a92cc01160a017e155d54d5688adece398cabdc809fac10421ce8b55723d3e300490119fc1401d9aaed05fd3b388a6323878756d9db5dcb2f96c5dd620500225"

	if !VerifyMidtransSignature(order, status, gross, key, valid) {
		t.Error("valid signature should verify")
	}
	if !VerifyMidtransSignature(order, status, gross, key, strings.ToUpper(valid)) {
		t.Error("uppercase hex signature should verify")
	}
	if VerifyMidtransSignature(order, status, "150001", key, valid) {
		t.Error("tampered gross_amount must not verify")
	}
	if VerifyMidtransSignature(order, status, gross, "wrong-key", valid) {
		t.Error("wrong server key must not verify")
	}
	if VerifyMidtransSignature(order, status, gross, key, "deadbeef") {
		t.Error("short signature must not verify")
	}
	if VerifyMidtransSignature(order, status, gross, key, "") {
		t.Error("empty signature must not verify (fail closed)")
	}
	if VerifyMidtransSignature(order, status, gross, "", valid) {
		t.Error("empty server key must not verify (fail closed)")
	}
	if VerifyMidtransSignature("", status, gross, key, valid) {
		t.Error("empty order_id must not verify (fail closed)")
	}
}

func TestParseOrderID(t *testing.T) {
	uid, ts, err := ParseOrderID("INV-123-1700000000")
	if err != nil {
		t.Fatalf("valid order_id rejected: %v", err)
	}
	if uid != 123 || ts != 1700000000 {
		t.Errorf("ParseOrderID = (%d,%d), want (123,1700000000)", uid, ts)
	}

	invalid := []string{
		"",
		"INV-123",
		"INV-abc-123",
		"INV-0-123",
		"INV-123-0",
		"INV-123-456-789",
		"inv-123-456",
		"INV-123-456 ",
		" INV-123-456",
		"../INV-1-2",
		"INV-1-2\n",
		"TX-123-456",
	}
	for _, in := range invalid {
		if _, _, err := ParseOrderID(in); err == nil {
			t.Errorf("ParseOrderID(%q) should fail", in)
		}
	}
}

func TestActivePlansFallback(t *testing.T) {
	s := &PaymentService{}
	plans := s.activePlans()
	for _, want := range []string{"free", "pro", "enterprise", "whitelabel"} {
		if !plans[want] {
			t.Errorf("fallback plans should contain %q", want)
		}
	}
	if plans["diamond"] {
		t.Error("fallback plans must not contain unknown plan")
	}
	if !s.isValidPlan("Pro") {
		t.Error("plan validation should be case-insensitive")
	}
	if s.isValidPlan("diamond") {
		t.Error("unknown plan should be rejected")
	}
	if s.isValidPlan("") {
		t.Error("empty plan should be rejected")
	}
}

func TestCreateTransactionRejectsUnknownPlan(t *testing.T) {
	s := &PaymentService{}
	if _, err := s.CreateTransaction(1, "diamond", 99000); err == nil {
		t.Error("unknown plan should be rejected")
	}
}

func TestCreateTransactionNoServerKey(t *testing.T) {
	s := &PaymentService{}
	_, err := s.CreateTransaction(7, "pro", 99000)
	if err == nil {
		t.Fatal("expected payment provider not configured error")
	}
	if !errors.Is(err, ErrPaymentNotConfigured) {
		t.Errorf("error should wrap ErrPaymentNotConfigured, got: %v", err)
	}
	if !strings.Contains(err.Error(), "payment provider not configured") {
		t.Errorf("error message should state provider not configured, got: %v", err)
	}
}

func TestHandleCallbackFailClosed(t *testing.T) {
	s := &PaymentService{}

	if err := s.HandleCallback(nil); err == nil {
		t.Error("nil payload should fail")
	}
	if err := s.HandleCallback(map[string]interface{}{"order_id": "bogus"}); err == nil {
		t.Error("malformed order_id should fail")
	}

	validFormat := map[string]interface{}{
		"order_id":           "INV-7-1700000000",
		"status_code":        "200",
		"gross_amount":       "150000",
		"signature_key":      "a92cc01160a017e155d54d5688adece398cabdc809fac10421ce8b55723d3e300490119fc1401d9aaed05fd3b388a6323878756d9db5dcb2f96c5dd620500225",
		"transaction_status": "settlement",
	}
	if err := s.HandleCallback(validFormat); !errors.Is(err, ErrPaymentNotConfigured) {
		t.Errorf("unconfigured provider should fail closed with ErrPaymentNotConfigured, got: %v", err)
	}
}

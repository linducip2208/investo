package seo

import (
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	k1 := generateKey()
	k2 := generateKey()
	if len(k1) != 32 {
		t.Errorf("key length = %d, want 32", len(k1))
	}
	if k1 == k2 {
		t.Error("two generated keys should differ")
	}
	for _, c := range k1 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("key contains non-hex char %q", c)
		}
	}
}

func TestNewIndexNowServiceHostNormalization(t *testing.T) {
	cases := map[string]string{
		"https://investo.whitelabel.co.id": "investo.whitelabel.co.id",
		"http://example.com":               "example.com",
		"example.com":                      "example.com",
	}
	for in, want := range cases {
		s := NewIndexNowService(nil, in)
		if s.Host != want {
			t.Errorf("Host = %q, want %q", s.Host, want)
		}
	}
}

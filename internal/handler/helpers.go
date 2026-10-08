package handler

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"

	"investo/internal/model"
)

func safeUser(u *model.User) *model.User {
	if u == nil {
		return &model.User{Name: "Guest", Email: "", Role: "visitor"}
	}
	return u
}

// renderHTML buffers output so template failures never return a partial page.
func renderHTML(w http.ResponseWriter, templates *template.Template, name string, data interface{}) {
	var output bytes.Buffer
	if templates == nil {
		log.Printf("render template %s: template set is nil", name)
		http.Error(w, "Halaman sedang tidak tersedia", http.StatusInternalServerError)
		return
	}
	if err := templates.ExecuteTemplate(&output, name, data); err != nil {
		log.Printf("render template %s: %v", name, err)
		http.Error(w, "Halaman sedang tidak tersedia", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = output.WriteTo(w)
}

type contactView struct {
	Email       string
	EmailURL    string
	WhatsApp    string
	WhatsAppURL string
	HasContact  bool
}

func configuredContact() contactView {
	email := strings.TrimSpace(os.Getenv("INVESTO_CONTACT_EMAIL"))
	whatsApp := strings.TrimSpace(os.Getenv("INVESTO_CONTACT_WHATSAPP"))
	view := contactView{}

	if parsed, err := url.Parse("mailto:" + email); email != "" && err == nil && parsed.Opaque != "" {
		view.Email = email
		view.EmailURL = parsed.String()
	}

	var digits strings.Builder
	for _, char := range whatsApp {
		if unicode.IsDigit(char) {
			digits.WriteRune(char)
		}
	}
	if digits.Len() >= 8 {
		view.WhatsApp = "+" + digits.String()
		view.WhatsAppURL = "https://wa.me/" + digits.String()
	}
	view.HasContact = view.EmailURL != "" || view.WhatsAppURL != ""
	return view
}

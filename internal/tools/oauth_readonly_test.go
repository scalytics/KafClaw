package tools

import "testing"

func TestScopeHasAny(t *testing.T) {
	if !scopeHasAny("openid https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/gmail.readonly") {
		t.Fatal("expected scope match")
	}
	if !scopeHasAny("Mail.Read Calendars.Read", "mail.read") {
		t.Fatal("expected case-insensitive scope match")
	}
	if scopeHasAny("openid email", "Calendars.Read") {
		t.Fatal("unexpected scope match")
	}
}

func TestClamp(t *testing.T) {
	if got := clamp(0, 1, 50); got != 1 {
		t.Fatalf("expected min clamp, got %d", got)
	}
	if got := clamp(100, 1, 50); got != 50 {
		t.Fatalf("expected max clamp, got %d", got)
	}
	if got := clamp(10, 1, 50); got != 10 {
		t.Fatalf("expected unchanged value, got %d", got)
	}
}

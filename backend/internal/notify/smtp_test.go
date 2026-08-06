package notify

import (
	"encoding/base64"
	"strings"
	"testing"
)

// A relay without SMTPUTF8 support bounces anything with raw UTF-8 in it, so
// what goes on the wire has to be plain ASCII whatever the caller passes in.
func TestBuildMessageIsASCIIOnly(t *testing.T) {
	subject := "Netglance — new device on LAN: caffè"
	body := "Vendor: Ubiquiti — VLAN 20\nName: —\n"

	msg := string(buildMessage("netglance@example.org", []string{"alert@example.org"}, subject, body))

	for i := 0; i < len(msg); i++ {
		if msg[i] > 127 {
			t.Fatalf("non-ASCII byte %q at offset %d", msg[i], i)
		}
	}
	if !strings.Contains(msg, "Subject: =?UTF-8?q?") {
		t.Errorf("subject not RFC 2047 encoded:\n%s", msg)
	}

	// The body must survive the round trip, not just be ASCII.
	_, encoded, found := strings.Cut(msg, "\r\n\r\n")
	if !found {
		t.Fatal("no header/body separator")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(encoded), "\r\n", ""))
	if err != nil {
		t.Fatalf("body is not valid base64: %v", err)
	}
	if string(decoded) != body {
		t.Errorf("body round-trip failed:\ngot  %q\nwant %q", decoded, body)
	}

	// Plain ASCII subjects should pass through untouched, not get encoded.
	plain := string(buildMessage("a@b.c", []string{"d@e.f"}, "Netglance test email", "hello"))
	if !strings.Contains(plain, "Subject: Netglance test email\r\n") {
		t.Errorf("ASCII subject was needlessly encoded:\n%s", plain)
	}
}

// Base64 body lines must stay within the RFC 2045 limit.
func TestWrapBase64LineLength(t *testing.T) {
	long := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("netglance ", 40)))
	for _, line := range strings.Split(strings.TrimSpace(wrapBase64(long)), "\r\n") {
		if len(line) > 76 {
			t.Errorf("line of %d chars exceeds 76", len(line))
		}
	}
}

// A display name with an accent in the From field is the other way raw UTF-8
// used to reach the wire, and the envelope must get the bare address anyway.
func TestAddressHandling(t *testing.T) {
	msg := string(buildMessage("Nètglance <netglance@example.org>", []string{"Città <alert@example.org>"}, "test", "hi"))
	for i := 0; i < len(msg); i++ {
		if msg[i] > 127 {
			t.Fatalf("non-ASCII byte %q at offset %d in:\n%s", msg[i], i, msg)
		}
	}
	if !strings.Contains(msg, "From: =?utf-8?q?N=C3=A8tglance?= <netglance@example.org>") {
		t.Errorf("From not encoded as expected:\n%s", msg)
	}

	cases := map[string]string{
		"Nètglance <netglance@example.org>": "netglance@example.org",
		"  alert@example.org  ":             "alert@example.org",
		"not an address":                    "not an address", // passed through, not dropped
	}
	for in, want := range cases {
		if got := bareAddr(in); got != want {
			t.Errorf("bareAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

package notify

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Config struct {
	Host       string
	Port       int
	UseTLS     bool
	UseAuth    bool
	Username   string
	Password   string
	From       string
	Recipients []string
}

func Send(cfg Config, subject, body string) error {
	if cfg.Host == "" {
		return errors.New("smtp host not configured")
	}
	if cfg.Port == 0 {
		cfg.Port = 25
	}
	if cfg.From == "" {
		return errors.New("smtp from not configured")
	}
	if len(cfg.Recipients) == 0 {
		return errors.New("smtp recipients not configured")
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	msg := buildMessage(cfg.From, cfg.Recipients, subject, body)
	if cfg.UseTLS && cfg.Port == 465 {
		return sendImplicitTLS(addr, cfg, msg)
	}
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()
	if cfg.UseTLS {
		if err := c.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if cfg.UseAuth {
		if err := c.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	return submit(c, cfg.From, cfg.Recipients, msg)
}

func sendImplicitTLS(addr string, cfg Config, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()
	if cfg.UseAuth {
		if err := c.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	return submit(c, cfg.From, cfg.Recipients, msg)
}

func submit(c *smtp.Client, from string, to []string, msg []byte) error {
	// The envelope carries bare addresses: a display name belongs in the
	// header, and MAIL FROM would choke on one.
	if err := c.Mail(bareAddr(from)); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, r := range to {
		if err := c.Rcpt(bareAddr(r)); err != nil {
			return fmt.Errorf("rcpt %s: %w", r, err)
		}
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return fmt.Errorf("write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}

// buildMessage renders a message that is pure 7-bit ASCII on the wire.
//
// Raw UTF-8 anywhere in a mail — an em dash in the subject, an accented host
// name — makes Postfix ask the next hop for SMTPUTF8, and a relay that doesn't
// offer it bounces the message outright ("no server was found that supports
// SMTPUTF8"). So the subject goes through RFC 2047 encoded-words and the body
// through base64; both are no-ops for plain ASCII content.
func buildMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", headerAddr(from))
	fmt.Fprintf(&b, "To: %s\r\n", headerAddrList(to))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	body = wrapBase64(base64.StdEncoding.EncodeToString([]byte(body)))
	b.WriteString(body)
	return []byte(b.String())
}

// headerAddr re-renders an address for a header. net/mail encodes a display
// name as an RFC 2047 word when it isn't ASCII — the same trap the subject had,
// reachable by typing an accented name in the From field. Input that doesn't
// parse is passed through untouched: better an odd header than a mail we refuse
// to send over a formatting quibble.
func headerAddr(s string) string {
	a, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return s
	}
	return a.String()
}

func headerAddrList(list []string) string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, headerAddr(s))
	}
	return strings.Join(out, ", ")
}

// bareAddr strips a display name down to the address itself, for the envelope.
func bareAddr(s string) string {
	s = strings.TrimSpace(s)
	if a, err := mail.ParseAddress(s); err == nil {
		return a.Address
	}
	return s
}

// wrapBase64 folds a base64 payload at 76 characters, the limit RFC 2045 sets
// for a body line. Postfix accepts longer, stricter relays do not.
func wrapBase64(s string) string {
	const width = 76
	var b strings.Builder
	for len(s) > width {
		b.WriteString(s[:width])
		b.WriteString("\r\n")
		s = s[width:]
	}
	b.WriteString(s)
	b.WriteString("\r\n")
	return b.String()
}

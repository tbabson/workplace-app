package notification

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

type Mailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewMailer(host string, port int, username, password, from string) *Mailer {
	return &Mailer{host: host, port: port, username: username, password: password, from: from}
}

var emailTpl = template.Must(template.New("email").Parse(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;">
  <div style="max-width:600px;margin:auto;background:#fff;border-radius:8px;padding:30px;">
    <h2 style="color:#333;">{{.Title}}</h2>
    <p style="color:#555;line-height:1.6;">{{.Body}}</p>
    <hr style="border:none;border-top:1px solid #eee;margin:20px 0;">
    <p style="color:#999;font-size:12px;">You are receiving this because you are a member of Workplace.</p>
  </div>
</body>
</html>`))

func (m *Mailer) Send(to, name, subject, body string) error {
	var buf bytes.Buffer
	if err := emailTpl.Execute(&buf, map[string]string{
		"Title": subject,
		"Body":  body,
	}); err != nil {
		return err
	}

	msg := fmt.Sprintf(
		"From: Workplace <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, buf.String(),
	)

	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}

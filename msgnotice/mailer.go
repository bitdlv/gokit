package msgnotice

import (
	"github.com/bitdlv/gokit/mail"
	"github.com/bitdlv/gokit/msgnotice/template"
)

// 邮件
type Mailer struct {
	to         string
	from       string
	subject    string
	tpl        string
	value      map[string]interface{}
	mailClient *mail.MailPool //邮件服务客户端
}

func NewMailer(smtpServer string, smtpPort int, smtpUser, smtpPassword string) *Mailer {
	return &Mailer{
		from:       smtpUser,
		mailClient: mail.NewMailPool(10, smtpServer, smtpPort, smtpUser, smtpPassword), //邮件服务
	}
}

func (m *Mailer) Send() error {
	return m.mailClient.SendMailByTemplate(m.from, m.to, m.subject, m.tpl, m.value)
}

func (m *Mailer) SetFrom() *Mailer {
	m.from = template.MailAccount
	return m
}

func (m *Mailer) SetCustomFrom(mailAccount string) *Mailer {
	m.from = mailAccount
	return m
}

func (m *Mailer) SetTo(to string) *Mailer {
	m.to = to
	return m
}

func (m *Mailer) SetSubject(subject string) *Mailer {
	m.subject = subject
	return m
}

func (m *Mailer) SetValue(value map[string]interface{}) *Mailer {
	m.value = value
	return m
}

func (m *Mailer) SetTpl(tpl string) *Mailer {
	m.tpl = tpl
	return m
}

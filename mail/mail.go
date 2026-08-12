package mail

import (
	"bytes"
	"strings"
	"text/template"

	"gopkg.in/gomail.v2"
)

const (
	HTML = "text/html"
)

type Mail struct {
	emsg   *gomail.Message
	dialer *gomail.Dialer
}

type MailConf struct {
	SMTPServer string
	ServerPort int
	UserName   string
	Pwd        string
}

func NewMail(conf MailConf) *Mail {
	mail := &Mail{emsg: gomail.NewMessage()}
	mail.dialer = gomail.NewDialer(conf.SMTPServer, conf.ServerPort, conf.UserName, conf.Pwd)

	return mail
}

func (m *Mail) Native() *gomail.Message {
	return m.emsg
}

func (m *Mail) To(to ...string) *Mail {
	if len(to) == 0 {
		return m
	}
	m.emsg.SetHeader("To", to...)
	return m
}

func (m *Mail) From(from string) *Mail {
	m.emsg.SetHeader("From", from)
	return m
}

func (m *Mail) AddToCC(ccEmail, ccName string) *Mail {
	m.emsg.SetAddressHeader("Cc", ccEmail, ccName)
	return m
}

func (m *Mail) SetSubject(subject string) *Mail {
	m.emsg.SetHeader("Subject", subject)
	return m
}

// 附件
func (m *Mail) Attach(filename string, settings ...gomail.FileSetting) *Mail {
	m.emsg.Attach(filename, settings...)
	return m
}

func (m *Mail) SendTemplate(ctype, temp string, data any) error {
	if temp == "" {
		return nil
	}
	tpl := template.Must(template.New("query").Funcs(fn()).Parse(temp))
	sb := bytes.NewBuffer([]byte{})
	err := tpl.Execute(sb, data)
	if err != nil {
		return err
	}
	m.emsg.SetBody(ctype, sb.String())

	return m.dialer.DialAndSend(m.emsg)
}

func (m *Mail) Send(ctype, template string) error {
	m.emsg.SetBody(ctype, template)
	return m.dialer.DialAndSend(m.emsg)
}

func fn() template.FuncMap {
	return template.FuncMap{
		"tolow": func(s string) string {
			return strings.ToLower(s[:1]) + s[1:]
		},
		"trim": func(s, cutset string) string {
			return strings.Trim(s, cutset)
		},
	}
}

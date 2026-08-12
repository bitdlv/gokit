package mail

import (
	"bytes"
	"gopkg.in/gomail.v2"
	"html/template"
	"log"
)

type MailPool struct {
	pool   chan *gomail.Message
	dialer *gomail.Dialer
}

func NewMailPool(poolSize int, smtpServer string, smtpPort int, smtpUser, smtpPassword string) *MailPool {
	pool := &MailPool{
		pool:   make(chan *gomail.Message, poolSize),
		dialer: gomail.NewDialer(smtpServer, smtpPort, smtpUser, smtpPassword),
	}

	// Initialize the pool with new Message objects
	for i := 0; i < poolSize; i++ {
		pool.pool <- gomail.NewMessage()
	}

	return pool
}

// SendMailByTemplate 通过模板发送邮件
func (p *MailPool) SendMailByTemplate(from, to, subject, templateText string, data map[string]interface{}) error {
	// Get a message from the pool
	msg := <-p.pool

	// Set the email parameters
	msg.SetHeader("From", from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)

	// Create HTML template for email body
	tpl := template.Must(template.New("email").Parse(templateText))

	// Execute the template with data to generate the email body
	emailBody := new(bytes.Buffer)
	err := tpl.Execute(emailBody, data)

	if err != nil {
		log.Fatal(err)
	}
	msg.SetBody("text/html", emailBody.String()) // Set the email body using the generated HTML content

	// Send the email and handle any errors
	err = p.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}

	// Reset the message before returning it to the pool
	msg.Reset()
	p.pool <- msg // Return the message to the pool
	return nil    // Indicate successful execution of the method if no errors occurred during execution of this function.
}

// SendMail 正常发送邮件
func (p *MailPool) SendMail(from, to, subject, body string) error {
	// 从连接池中获取一个消息对象
	msg := <-p.pool

	// 设置邮件参数
	msg.SetHeader("From", from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	// 使用消息对象发送邮件
	err := p.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}

	msg.Reset()
	// 将消息对象放回连接池
	p.pool <- msg

	return nil
}

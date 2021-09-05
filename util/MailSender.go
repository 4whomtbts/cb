package util

import (
	"fmt"
	"net/smtp"
)

type MailSender struct {
	serverId string
	mail string
	password string
	receiverList []string
}

func NewMailSender(serverId, mail, password string, receiverList []string) *MailSender {
	return &MailSender {
		serverId: serverId,
		mail: mail,
		password: password,
		receiverList: receiverList,
	}
}

func toHtml(content string) string {
	container := "<html><body>"
	return container + content + container
}

func (m *MailSender) Send(title string, content string) {
	from := m.mail
	password := m.password

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	headerSubject := fmt.Sprintf("Subject: %s\r\n", title)
	//headerBlank := "\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)
	content = toHtml(fmt.Sprintf("<p><h3>메일 발신 서킷브레이커 : %s</h3></p>", m.serverId) + content)

	msg := []byte(headerSubject + mime + content)
	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, m.receiverList, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Email Sent Successfully!")
}

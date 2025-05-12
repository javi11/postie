package poster

import (
	"fmt"
	"time"
)

type Article struct {
	MessageID  string
	Subject    string
	From       string
	Date       time.Time
	Body       []byte
	Bytes      int
	Group      string
	PartNumber int
	TotalParts int
	FileName   string
}

func NewArticle(messageID string, subject string, from string, body []byte, group string, partNumber, totalParts int, fileName string) *Article {
	return &Article{
		MessageID:  messageID,
		Subject:    subject,
		From:       from,
		Date:       time.Now(),
		Body:       body,
		Bytes:      len(body),
		Group:      group,
		PartNumber: partNumber,
		TotalParts: totalParts,
		FileName:   fileName,
	}
}

func (a *Article) Header() string {
	return fmt.Sprintf("From: %s\r\n"+
		"Subject: %s\r\n"+
		"Newsgroups: %s\r\n"+
		"Message-ID: %s\r\n"+
		"Date: %s\r\n"+
		a.From,
		a.Subject,
		a.Group,
		a.MessageID,
		a.Date.Format(time.RFC1123Z),
	)
}

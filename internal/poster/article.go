package poster

import (
	"bytes"
	"fmt"
	"time"
)

type Article struct {
	MessageID  string
	Subject    string
	From       string
	Date       time.Time
	Body       []byte
	Lines      int
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
		Lines:      bytes.Count(body, []byte{'\n'}) + 1,
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
		"Lines: %d\r\n"+
		"Bytes: %d\r\n\r\n",
		a.From,
		a.Subject,
		a.Group,
		a.MessageID,
		a.Date.Format(time.RFC1123Z),
		a.Lines,
		a.Bytes,
	)
}

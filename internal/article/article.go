package article

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"hash/crc32"
	"io"
	"math/big"
	mrand "math/rand"
	"strings"
	"time"
)

type Encoder interface {
	Encode(p []byte) []byte
}

// Article represents a Usenet article
type Article struct {
	MessageID       string
	Subject         string
	OriginalSubject string
	From            string
	Groups          []string
	PartNumber      int
	TotalParts      int
	FileName        string
	Date            time.Time
	FileNumber      int
	Offset          int64
	Size            uint64
	FileSize        int64
	OriginalName    string
	CustomHeaders   map[string]string
	XNxgHeader      string
	Hash            string
}

// New creates a new Article
func New(
	messageID,
	subject,
	originalSubject,
	from string,
	groups []string,
	partNumber,
	totalParts int,
	fileSize int64,
	fileName string,
	fileNumber int,
	originalName string,
	customHeaders map[string]string,
) *Article {
	return &Article{
		MessageID:       messageID,
		Subject:         subject,
		OriginalSubject: originalSubject,
		From:            from,
		Groups:          groups,
		PartNumber:      partNumber,
		TotalParts:      totalParts,
		FileSize:        fileSize,
		FileName:        fileName,
		FileNumber:      fileNumber,
		OriginalName:    originalName,
		Date:            time.Now(),
		CustomHeaders:   customHeaders,
	}
}

// EncodeBytes encodes the article body using the provided encoder
func (a *Article) EncodeBytes(encoder Encoder, body []byte) (io.Reader, error) {
	headers := make(map[string]string)

	if a.CustomHeaders != nil {
		for k, v := range a.CustomHeaders {
			headers[k] = v
		}
	}

	headers["Subject"] = a.Subject
	headers["From"] = a.From
	headers["Newsgroups"] = strings.Join(a.Groups, ",")
	headers["Message-ID"] = fmt.Sprintf("<%s>", a.MessageID)
	headers["Date"] = a.Date.UTC().Format(time.RFC1123)

	if a.XNxgHeader != "" {
		headers["X-Nxg"] = a.XNxgHeader
	}

	header := ""
	for k, v := range headers {
		header += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	header += fmt.Sprintf("\r\n=ybegin part=%d total=%d line=128 size=%d name=%s\r\n=ypart begin=%d end=%d\r\n",
		a.PartNumber, a.TotalParts, a.FileSize, a.FileName, a.Offset+1, a.Offset+int64(a.Size))

	// Encoded data
	encoded := encoder.Encode(body)

	// yEnc end line
	h := crc32.NewIEEE()
	_, err := h.Write(body)
	if err != nil {
		return nil, err
	}
	footer := fmt.Sprintf("\r\n=yend size=%d part=%d pcrc32=%08X\r\n", a.Size, a.PartNumber, h.Sum32())

	size := len(header) + len(encoded) + len(footer)
	buf := bytes.NewBuffer(make([]byte, 0, size))

	_, err = buf.WriteString(header)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(encoded)
	if err != nil {
		return nil, err
	}
	_, err = buf.WriteString(footer)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		// Generate random index
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("error generating random string: %w", err)
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}

// GenerateMessageID generates a message ID following the obfuscation pattern
func GenerateMessageID() (string, error) {
	// Format: {rand(32)}@{rand(8)}.{rand(3)}
	rand32, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	rand8, err := generateRandomString(8)
	if err != nil {
		return "", err
	}
	rand3, err := generateRandomString(3)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s@%s.%s", rand32, rand8, rand3), nil
}

// GenerateFrom generates a From header following the obfuscation pattern
func GenerateFrom() (string, error) {
	// Format: {rand(14)} <{rand(14)}@{rand(5)}.{rand(3)}>
	rand14a, err := generateRandomString(14)
	if err != nil {
		return "", err
	}
	rand14b, err := generateRandomString(14)
	if err != nil {
		return "", err
	}
	rand5, err := generateRandomString(5)
	if err != nil {
		return "", err
	}
	rand3, err := generateRandomString(3)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s <%s@%s.%s>", rand14a, rand14b, rand5, rand3), nil
}

// GenerateSubject generates a subject following the obfuscation pattern
func GenerateSubject(fileNumber int, totalFiles int, fileName string, partNumber int, numSegments int) string {
	return fmt.Sprintf("[%v/%v] \"%v\" - yEnc (%v/%v)", fileNumber, totalFiles, fileName, partNumber, numSegments)
}

func GenerateRandomSubject() string {
	rand32, err := generateRandomString(32)
	if err != nil {
		return ""
	}

	return rand32
}

func RandomDateWithinLast6Hours() time.Time {
	now := time.Now()
	millisecondsIn6Hours := 6 * 60 * 60 * 1000
	randomMilliseconds := mrand.Intn(millisecondsIn6Hours)
	randomDuration := time.Duration(randomMilliseconds) * time.Millisecond
	randomDate := now.Add(-randomDuration)

	return randomDate
}

// GenerateRandomFilename generates a random filename
func GenerateRandomFilename() string {
	rand32, err := generateRandomString(32)
	if err != nil {
		return ""
	}

	return rand32
}

package article

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"strings"
	"sync"
	"time"

	"github.com/mnightingale/rapidyenc"
)

// encoderBufferPool provides reusable buffers for article encoding to reduce GC pressure
var encoderBufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate 900KB (slightly larger than typical article size)
		return bytes.NewBuffer(make([]byte, 0, 900*1024))
	},
}

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

// Encode encodes the article body using yEnc encoding.
// Returns the encoded reader, a cleanup function to return the buffer to the pool, and any error.
// The caller MUST call the cleanup function when done with the reader.
func (a *Article) Encode(body []byte) (io.Reader, func(), error) {
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

	// Get buffer from pool and reset it
	buff := encoderBufferPool.Get().(*bytes.Buffer)
	buff.Reset()

	// Cleanup function to return buffer to pool
	cleanup := func() {
		encoderBufferPool.Put(buff)
	}

	buff.WriteString(header + "\r\n")

	encoder, err := rapidyenc.NewEncoder(buff, rapidyenc.Meta{
		FileName:   a.FileName,
		FileSize:   a.FileSize,
		PartNumber: int64(a.PartNumber),
		TotalParts: int64(a.TotalParts),
		Offset:     int64(a.Offset),
		PartSize:   int64(a.Size),
	})
	if err != nil {
		cleanup() // Return buffer to pool on error
		return nil, nil, fmt.Errorf("error creating encoder: %w", err)
	}

	_, errWrite := encoder.Write(body)

	if err := encoder.Close(); err != nil {
		cleanup() // Return buffer to pool on error
		return nil, nil, fmt.Errorf("error closing encoder: %w", err)
	}

	if errWrite != nil {
		cleanup() // Return buffer to pool on error
		return nil, nil, fmt.Errorf("error writing article body: %w", errWrite)
	}

	return buff, cleanup, nil
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

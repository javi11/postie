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
type Article interface {
	GetMessageID() string
	GetOriginalSubject() string
	GetSubject() string
	GetFrom() string
	GetGroups() []string
	GetPartNumber() int
	GetTotalParts() int
	GetFileName() string
	GetDate() time.Time
	GetOffset() int64
	GetSize() uint64
	GetOriginalName() string
	SetOffset(offset int64)
	SetSize(size uint64)
	GetFileNumber() int
	SetDate(date time.Time)
	SetXNxgHeader(xNxgHeader string)
	EncodeBytes(encoder Encoder, body []byte) (io.Reader, error)
}

func (a *article) SetXNxgHeader(xNxgHeader string) {
	a.XNxgHeader = xNxgHeader
}

func (a *article) SetDate(date time.Time) {
	a.date = date
}

// Article represents an NNTP article
type article struct {
	messageID       string
	subject         string
	originalSubject string
	from            string
	groups          []string
	partNumber      int
	totalParts      int
	fileName        string
	date            time.Time
	fileNumber      int
	offset          int64
	size            uint64
	fileSize        int64
	originalName    string
	customHeaders   map[string]string
	XNxgHeader      string
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
) Article {
	return &article{
		messageID:       messageID,
		subject:         subject,
		originalSubject: originalSubject,
		from:            from,
		groups:          groups,
		partNumber:      partNumber,
		totalParts:      totalParts,
		fileSize:        fileSize,
		fileName:        fileName,
		fileNumber:      fileNumber,
		originalName:    originalName,
		date:            time.Now(),
		customHeaders:   customHeaders,
	}
}

func (a *article) EncodeBytes(encoder Encoder, body []byte) (io.Reader, error) {
	headers := make(map[string]string)

	if a.customHeaders != nil {
		for k, v := range a.customHeaders {
			headers[k] = v
		}
	}

	headers["Subject"] = a.subject
	headers["From"] = a.from
	headers["Newsgroups"] = strings.Join(a.groups, ",")
	headers["Message-ID"] = fmt.Sprintf("<%s>", a.messageID)
	headers["Date"] = a.date.UTC().Format(time.RFC1123)

	if a.XNxgHeader != "" {
		headers["X-Nxg"] = a.XNxgHeader
	}

	header := ""
	for k, v := range headers {
		header += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	header += fmt.Sprintf("\r\n=ybegin part=%d total=%d line=128 size=%d name=%s\r\n=ypart begin=%d end=%d\r\n",
		a.partNumber, a.totalParts, a.fileSize, a.fileName, a.offset+1, a.offset+int64(a.size))

	// Encoded data
	encoded := encoder.Encode(body)

	// yEnc end line
	h := crc32.NewIEEE()
	_, err := h.Write(body)
	if err != nil {
		return nil, err
	}
	footer := fmt.Sprintf("\r\n=yend size=%d part=%d pcrc32=%08X\r\n", a.size, a.partNumber, h.Sum32())

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

// GetFileNumber returns the file number
func (a *article) GetFileNumber() int {
	return a.fileNumber
}

// GetOriginalName returns the original filename
func (a *article) GetOriginalName() string {
	return a.originalName
}

// GetOriginalSubject returns the original subject
func (a *article) GetOriginalSubject() string {
	return a.originalSubject
}

// GetMessageID returns the message ID
func (a *article) GetMessageID() string {
	return a.messageID
}

// GetSubject returns the subject
func (a *article) GetSubject() string {
	return a.subject
}

// GetFrom returns the from header
func (a *article) GetFrom() string {
	return a.from
}

// GetGroup returns the newsgroup
func (a *article) GetGroups() []string {
	return a.groups
}

// GetPartNumber returns the part number
func (a *article) GetPartNumber() int {
	return a.partNumber
}

// GetTotalParts returns the total number of parts
func (a *article) GetTotalParts() int {
	return a.totalParts
}

// GetFileName returns the original filename
func (a *article) GetFileName() string {
	return a.fileName
}

// GetDate returns the article date
func (a *article) GetDate() time.Time {
	return a.date
}

// GetOffset returns the file offset
func (a *article) GetOffset() int64 {
	return a.offset
}

// GetSize returns the article size
func (a *article) GetSize() uint64 {
	return a.size
}

// SetOffset sets the file offset
func (a *article) SetOffset(offset int64) {
	a.offset = offset
}

// SetSize sets the article size
func (a *article) SetSize(size uint64) {
	a.size = size
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
func GenerateSubject(fileNumber int, totalFiles int, fileName string, fileSize int, partNumber int, numSegments int) string {
	return fmt.Sprintf("[%v/%v] \"%v\" - %v - yEnc (%v/%v)", fileNumber, totalFiles, fileName, fileSize, partNumber, numSegments)
}

func GenerateRandomSubject() string {
	rand32, err := generateRandomString(32)
	if err != nil {
		return ""
	}

	return rand32
}

func GetRandomDateWithinLast6Hours() time.Time {
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

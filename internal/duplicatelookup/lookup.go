// Package duplicatelookup implements the optional generic sampled-file lookup protocol.
package duplicatelookup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const Algorithm = "sample-sha256-v1"
const maxNZBSize = 64 << 20

type Request struct {
	Algorithm string `json:"algorithm"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`
}

// Fingerprint checks size and modification time around all local sample reads.
func Fingerprint(ctx context.Context, path string) (Request, error) {
	f, err := os.Open(path)
	if err != nil {
		return Request{}, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return Request{}, err
	}
	if !before.Mode().IsRegular() {
		return Request{}, fmt.Errorf("duplicate lookup requires a regular file")
	}
	name := filepath.Base(path)
	if strings.ContainsAny(name, `/\`) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return Request{}, fmt.Errorf("invalid lookup filename")
	}
	h := sha256.New()
	h.Write([]byte(Algorithm + "\x00"))
	binary.Write(h, binary.LittleEndian, uint64(before.Size()))
	size := before.Size()
	offsets := []int64{0}
	length := size
	if size > 196608 {
		length = 65536
		offsets = []int64{0, (size - 65536) / 2, size - 65536}
	}
	for _, offset := range offsets {
		if err := ctx.Err(); err != nil {
			return Request{}, err
		}
		binary.Write(h, binary.LittleEndian, uint64(offset))
		binary.Write(h, binary.LittleEndian, uint64(length))
		if _, err := io.CopyN(h, io.NewSectionReader(f, offset, length), length); err != nil {
			return Request{}, err
		}
	}
	after, err := f.Stat()
	if err != nil {
		return Request{}, err
	}
	current, err := os.Stat(path)
	if err != nil {
		return Request{}, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, current) || before.Size() != current.Size() || !before.ModTime().Equal(current.ModTime()) {
		return Request{}, fmt.Errorf("source changed during duplicate lookup")
	}
	if err := ctx.Err(); err != nil {
		return Request{}, err
	}
	return Request{Algorithm: Algorithm, Name: name, Size: size, Hash: hex.EncodeToString(h.Sum(nil))}, nil
}

// Lookup returns nil on an explicit miss. Every other failure stops the caller.
func Lookup(ctx context.Context, endpoint, key string, timeout time.Duration, path string) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return nil, fmt.Errorf("invalid duplicate lookup URL")
	}
	if timeout <= 0 || timeout > 5*time.Minute {
		return nil, fmt.Errorf("duplicate lookup timeout must be between 0 and 5 minutes")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	initial, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	fingerprint, err := Fingerprint(ctx, path)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(fingerprint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-nzb")
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duplicate lookup request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duplicate lookup returned HTTP %d", resp.StatusCode)
	}
	media, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || media != "application/x-nzb" {
		return nil, fmt.Errorf("duplicate lookup response is not application/x-nzb")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxNZBSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxNZBSize {
		return nil, fmt.Errorf("duplicate lookup NZB exceeds 64 MiB")
	}
	if err := ValidateNZB(data); err != nil {
		return nil, err
	}
	final, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(initial, final) || initial.Size() != final.Size() || !initial.ModTime().Equal(final.ModTime()) {
		return nil, fmt.Errorf("source changed during duplicate lookup")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

// ValidateNZB rejects incomplete XML and entries without usable segments.
func ValidateNZB(data []byte) error {
	var doc struct {
		XMLName xml.Name `xml:"nzb"`
		Files   []struct {
			Subject  string   `xml:"subject,attr"`
			Groups   []string `xml:"groups>group"`
			Segments []struct {
				Bytes  int64  `xml:"bytes,attr"`
				Number int    `xml:"number,attr"`
				ID     string `xml:",chardata"`
			} `xml:"segments>segment"`
		} `xml:"file"`
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	if err := d.Decode(&doc); err != nil {
		return fmt.Errorf("invalid lookup NZB: %w", err)
	}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid lookup NZB: %w", err)
		}
		if text, ok := tok.(xml.CharData); !ok || strings.TrimSpace(string(text)) != "" {
			return fmt.Errorf("trailing content in lookup NZB")
		}
	}
	if len(doc.Files) == 0 {
		return fmt.Errorf("lookup NZB contains no files")
	}
	for _, f := range doc.Files {
		if strings.TrimSpace(f.Subject) == "" {
			return fmt.Errorf("lookup NZB file has no subject")
		}
		usableGroup := false
		for _, group := range f.Groups {
			if strings.TrimSpace(group) != "" {
				usableGroup = true
			}
		}
		if !usableGroup {
			return fmt.Errorf("lookup NZB file has no newsgroups")
		}

		if len(f.Segments) == 0 {
			return fmt.Errorf("lookup NZB contains a file without segments")
		}
		seen := map[int]bool{}
		for _, s := range f.Segments {
			if s.Number <= 0 || s.Bytes <= 0 || strings.TrimSpace(s.ID) == "" || seen[s.Number] {
				return fmt.Errorf("lookup NZB contains an invalid segment")
			}
			seen[s.Number] = true
		}
		for n := 1; n <= len(f.Segments); n++ {
			if !seen[n] {
				return fmt.Errorf("lookup NZB has missing segments")
			}
		}
	}
	return nil
}

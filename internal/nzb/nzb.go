package nzb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/javi11/nzbparser"
	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/config"
	"github.com/klauspost/compress/zstd"
)

// NZBGenerator defines the interface for generating NZB files
type NZBGenerator interface {
	// AddArticle adds an article to the generator
	AddArticle(article article.Article)
	// AddFileHash adds a hash for a file
	AddFileHash(filename string, hash string)
	// Generate creates an NZB file
	Generate(outputPath string) error
}

// Generator creates NZB files
type Generator struct {
	articles          map[string][]article.Article // filename -> articles
	filesHash         map[string]string            // filename -> checksums
	segmentSize       uint64                       // size of each segment in bytes
	compressionConfig config.NzbCompressionConfig  // compression configuration
}

// NewGenerator creates a new NZB generator
func NewGenerator(segmentSize uint64, compressionConfig config.NzbCompressionConfig) NZBGenerator {
	return &Generator{
		articles:          make(map[string][]article.Article),
		filesHash:         make(map[string]string),
		segmentSize:       segmentSize,
		compressionConfig: compressionConfig,
	}
}

// AddArticle adds an article to the generator
func (g *Generator) AddArticle(art article.Article) {
	filename := art.GetOriginalName()

	// Check if we already have this article (by message ID)
	for i, existingArt := range g.articles[filename] {
		if existingArt.GetMessageID() == art.GetMessageID() {
			// Replace the existing article
			g.articles[filename][i] = art
			return
		}
	}

	// If we didn't find an existing article with the same message ID, append it
	g.articles[filename] = append(g.articles[filename], art)
}

// Generate creates an NZB file for all files
func (g *Generator) Generate(outputPath string) error {
	if len(g.articles) == 0 {
		return fmt.Errorf("no articles found")
	}

	// Create NZB file
	nzbFile := &nzbparser.Nzb{
		Meta: map[string]string{
			"date":       time.Now().Format(time.RFC3339),
			"chunk_size": fmt.Sprintf("%d", g.segmentSize),
		},
	}

	// Add all files to NZB
	fileNumber := 0
	for filename, articles := range g.articles {
		if len(articles) == 0 {
			continue
		}

		// Sort articles by part number
		sort.Slice(articles, func(i, j int) bool {
			return articles[i].GetPartNumber() < articles[j].GetPartNumber()
		})

		// Calculate file size from all segments
		var fileSize int64
		for _, a := range articles {
			fileSize += int64(a.GetSize())
		}

		// Create file entry
		file := nzbparser.NzbFile{
			Subject:       articles[0].GetOriginalSubject(),
			Groups:        articles[0].GetGroups(),
			Poster:        articles[0].GetFrom(),
			Date:          int(time.Now().Unix()),
			Bytes:         fileSize,
			Number:        articles[0].GetFileNumber(),
			TotalSegments: len(articles),
			Filename:      articles[0].GetOriginalName(),
		}

		// Add checksum if available
		if hash, ok := g.filesHash[filename]; ok {
			file.FileHash = hash
		}

		// Add segments
		for i, a := range articles {
			// Use configured segment size for all segments except the last one
			segmentSize := g.segmentSize
			if i == len(articles)-1 {
				segmentSize = a.GetSize()
			}

			segment := nzbparser.NzbSegment{
				Bytes:  int(segmentSize),
				Number: a.GetPartNumber(),
				ID:     a.GetMessageID(),
			}
			file.Segments = append(file.Segments, segment)
		}

		// Add file to NZB
		nzbFile.Files = append(nzbFile.Files, file)
		fileNumber++
	}

	// Sort files
	sort.Slice(nzbFile.Files, func(i, j int) bool {
		return nzbFile.Files[i].Number < nzbFile.Files[j].Number
	})

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Write NZB file
	data, err := nzbparser.Write(nzbFile)
	if err != nil {
		return fmt.Errorf("error writing NZB file: %w", err)
	}

	// Apply compression if enabled
	finalPath := outputPath
	if g.compressionConfig.Enabled {
		switch g.compressionConfig.Type {
		case config.CompressionTypeZstd:
			finalPath = outputPath + ".zst"
			if err := g.compressWithZstd(data, finalPath); err != nil {
				return fmt.Errorf("error compressing NZB file with zstd: %w", err)
			}
		case config.CompressionTypeBrotli:
			finalPath = outputPath + ".br"
			if err := g.compressWithBrotli(data, finalPath); err != nil {
				return fmt.Errorf("error compressing NZB file with brotli: %w", err)
			}
		default:
			// No compression or unknown type, write the file as is
			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				return fmt.Errorf("error writing NZB file: %w", err)
			}
		}
	} else {
		// No compression, write the file as is
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("error writing NZB file: %w", err)
		}
	}

	return nil
}

// compressWithZstd compresses data with zstd and writes it to the given path
func (g *Generator) compressWithZstd(data []byte, outputPath string) error {
	// Create the file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating compressed file: %w", err)
	}
	defer f.Close()

	// Create zstd encoder with the configured level
	level := zstd.EncoderLevel(g.compressionConfig.Level)
	enc, err := zstd.NewWriter(f, zstd.WithEncoderLevel(level))
	if err != nil {
		return fmt.Errorf("error creating zstd encoder: %w", err)
	}
	defer enc.Close()

	// Write compressed data
	if _, err := enc.Write(data); err != nil {
		return fmt.Errorf("error writing compressed data: %w", err)
	}

	return nil
}

// compressWithBrotli compresses data with brotli and writes it to the given path
func (g *Generator) compressWithBrotli(data []byte, outputPath string) error {
	// Create the file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating compressed file: %w", err)
	}
	defer f.Close()

	// Create brotli writer with the configured level
	w := brotli.NewWriterLevel(f, g.compressionConfig.Level)
	defer w.Close()

	// Write compressed data
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("error writing compressed data: %w", err)
	}

	return nil
}

// AddFileHash adds a hash for a file
func (g *Generator) AddFileHash(filename string, hash string) {
	g.filesHash[filename] = hash
}

// Parse reads an NZB file
func Parse(path string) (*nzbparser.Nzb, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading NZB file: %w", err)
	}
	return nzbparser.ParseString(string(data))
}

// Validate checks if an NZB file is valid
func Validate(path string) error {
	nzbFile, err := Parse(path)
	if err != nil {
		return fmt.Errorf("error parsing NZB file: %w", err)
	}

	if len(nzbFile.Files) == 0 {
		return fmt.Errorf("NZB file contains no files")
	}

	for _, file := range nzbFile.Files {
		if len(file.Segments) == 0 {
			return fmt.Errorf("file %s has no segments", file.Subject)
		}
	}

	return nil
}

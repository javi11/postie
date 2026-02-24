package par2

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/javi11/par2go"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/progress"
	"github.com/javi11/postie/pkg/fileinfo"
)

var parregexp = regexp.MustCompile(`(?i)(\.vol\d+\+(\d+))?\.par2$`)

// Par2Executor defines the interface for executing par2 commands.
type Par2Executor interface {
	Create(ctx context.Context, files []fileinfo.FileInfo) ([]string, error)
}

// NativeExecutor implements Par2Executor using the built-in Go PAR2 creator.
type NativeExecutor struct {
	articleSize uint64
	cfg         *config.Par2Config
	jobProgress progress.JobProgress
}

// New creates a new NativeExecutor.
func New(articleSize uint64, cfg *config.Par2Config, jobProgress progress.JobProgress) *NativeExecutor {
	return &NativeExecutor{
		articleSize: articleSize,
		cfg:         cfg,
		jobProgress: jobProgress,
	}
}

// checkExistingPar2Files checks if PAR2 files already exist for the given source file.
func (p *NativeExecutor) checkExistingPar2Files(ctx context.Context, sourceFile fileinfo.FileInfo) ([]string, bool) {
	var dirPath string
	if p.cfg.TempDir != "" {
		dirPath = p.cfg.TempDir
	} else {
		dirPath = filepath.Dir(sourceFile.Path)
	}

	return checkExistingPar2FilesInPath(ctx, sourceFile, dirPath)
}

// checkExistingPar2FilesInDir checks if PAR2 files already exist in a specific directory.
func (p *NativeExecutor) checkExistingPar2FilesInDir(ctx context.Context, sourceFile fileinfo.FileInfo, dirPath string) ([]string, bool) {
	return checkExistingPar2FilesInPath(ctx, sourceFile, dirPath)
}

// checkExistingPar2FilesInPath is the shared implementation for checking existing PAR2 files.
func checkExistingPar2FilesInPath(ctx context.Context, sourceFile fileinfo.FileInfo, dirPath string) ([]string, bool) {
	baseName := filepath.Base(sourceFile.Path)
	par2FileName := baseName + ".par2"
	mainPar2Path := filepath.Join(dirPath, par2FileName)

	// Check if main PAR2 file exists
	if _, err := os.Stat(mainPar2Path); os.IsNotExist(err) {
		return nil, false
	}

	// Collect all existing PAR2 files (main + volume files)
	var existingPaths []string
	existingPaths = append(existingPaths, mainPar2Path)

	// Find all volume files
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		slog.WarnContext(ctx, "Failed to read directory for existing par2 volumes", "error", err)
		return nil, false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, baseName) && strings.Contains(name, ".vol") && strings.HasSuffix(name, ".par2") {
			existingPaths = append(existingPaths, filepath.Join(dirPath, name))
		}
	}

	slog.InfoContext(ctx, "Found existing PAR2 files, skipping generation",
		"sourceFile", sourceFile.Path, "par2Files", len(existingPaths))

	return existingPaths, true
}

// Create creates PAR2 parity files for the given input files.
func (p *NativeExecutor) Create(ctx context.Context, files []fileinfo.FileInfo) ([]string, error) {
	slog.InfoContext(ctx, "Starting par2 creation process", "executor", "NativeExecutor")

	var createdPar2Paths []string
	for _, file := range files {
		if filepath.Ext(file.Path) == ".par2" {
			continue
		}

		// Check if PAR2 files already exist for this file
		if existingPaths, exists := p.checkExistingPar2Files(ctx, file); exists {
			createdPar2Paths = append(createdPar2Paths, existingPaths...)
			continue
		}

		// Determine output directory
		var dirPath string
		if p.cfg.TempDir != "" {
			dirPath = p.cfg.TempDir
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				slog.ErrorContext(ctx, "Failed to create temp directory", "path", dirPath, "error", err)
				dirPath = filepath.Dir(file.Path)
			}
		} else {
			dirPath = filepath.Dir(file.Path)
		}

		paths, err := p.createPar2ForFile(ctx, file, dirPath)
		if err != nil {
			return nil, err
		}
		createdPar2Paths = append(createdPar2Paths, paths...)
	}

	return createdPar2Paths, nil
}

// CreateInDirectory creates PAR2 files with optional output directory specification.
func (p *NativeExecutor) CreateInDirectory(ctx context.Context, files []fileinfo.FileInfo, outputDir string) ([]string, error) {
	slog.InfoContext(ctx, "Starting par2 creation process", "executor", "NativeExecutor", "outputDir", outputDir)

	var createdPar2Paths []string
	for _, file := range files {
		if filepath.Ext(file.Path) == ".par2" {
			continue
		}

		// Determine output directory based on parameters and configuration
		var dirPath string
		if outputDir != "" {
			dirPath = outputDir
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				slog.ErrorContext(ctx, "Failed to create output directory", "path", dirPath, "error", err)
				dirPath = filepath.Dir(file.Path)
			} else {
				if existingPaths, exists := p.checkExistingPar2FilesInDir(ctx, file, dirPath); exists {
					createdPar2Paths = append(createdPar2Paths, existingPaths...)
					continue
				}
			}
		} else if p.cfg.TempDir != "" {
			dirPath = p.cfg.TempDir
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				slog.ErrorContext(ctx, "Failed to create temp directory", "path", dirPath, "error", err)
				dirPath = filepath.Dir(file.Path)
			} else {
				if existingPaths, exists := p.checkExistingPar2Files(ctx, file); exists {
					createdPar2Paths = append(createdPar2Paths, existingPaths...)
					continue
				}
			}
		} else {
			dirPath = filepath.Dir(file.Path)
			if existingPaths, exists := p.checkExistingPar2Files(ctx, file); exists {
				createdPar2Paths = append(createdPar2Paths, existingPaths...)
				continue
			}
		}

		paths, err := p.createPar2ForFile(ctx, file, dirPath)
		if err != nil {
			return nil, err
		}
		createdPar2Paths = append(createdPar2Paths, paths...)
	}

	return createdPar2Paths, nil
}

// createPar2ForFile creates PAR2 files for a single input file in the given directory.
func (p *NativeExecutor) createPar2ForFile(ctx context.Context, file fileinfo.FileInfo, dirPath string) ([]string, error) {
	parBlockSize := calculateParBlockSize(file.Size, p.articleSize)
	par2FileName := filepath.Base(file.Path) + ".par2"
	par2Path := filepath.Join(dirPath, par2FileName)

	// Parse redundancy to determine number of recovery blocks
	redundancyPct := parseRedundancyPercentage(p.cfg.Redundancy, file.Size, parBlockSize)
	numInputSlices := int(math.Ceil(float64(file.Size) / float64(parBlockSize)))
	if numInputSlices == 0 {
		numInputSlices = 1
	}
	numRecovery := max(int(math.Ceil(float64(numInputSlices)*redundancyPct/100.0)), 1)

	slog.DebugContext(ctx, "PAR2 creation parameters",
		"file", file.Path,
		"blockSize", parBlockSize,
		"inputSlices", numInputSlices,
		"recoveryBlocks", numRecovery,
		"redundancy", redundancyPct)

	// Initialize progress tracking
	progressID := uuid.New()
	progressName := fmt.Sprintf("PAR2: %s", filepath.Base(file.Path))
	var pg progress.Progress
	if p.jobProgress != nil {
		pg = p.jobProgress.AddProgress(progressID, progressName, progress.ProgressTypePar2Generation, 100)
	}

	opts := par2go.Options{
		SliceSize:   int(parBlockSize),
		NumRecovery: numRecovery,
		Creator:     "Postie",
		OnProgress: func(phase string, pct float64) {
			if pg == nil {
				return
			}
			// Map phases to progress: hashing=0-20%, encoding=20-95%, writing=95-100%
			var overallPct float64
			switch phase {
			case "hashing":
				overallPct = pct * 20
			case "encoding":
				overallPct = 20 + pct*75
			case "writing":
				overallPct = 95 + pct*5
			}
			delta := int64(overallPct) - pg.GetCurrent()
			if delta > 0 {
				pg.UpdateProgress(delta)
			}
		},
	}

	err := par2go.Create(ctx, par2Path, []string{file.Path}, opts)
	if err != nil {
		if ctx.Err() == context.Canceled {
			slog.InfoContext(ctx, "Par2 creation cancelled", "path", file.Path)
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to create par2 for %s: %w", file.Path, err)
	}

	if p.jobProgress != nil {
		p.jobProgress.FinishProgress(progressID)
	}

	slog.InfoContext(ctx, "Par2 creation completed successfully", "file", file.Path)

	// Collect all created PAR2 files (main + volumes)
	var createdPaths []string
	createdPaths = append(createdPaths, par2Path)

	baseName := filepath.Base(file.Path)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		slog.WarnContext(ctx, "Failed to read directory to find par2 volumes", "error", err)
		return createdPaths, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, baseName) && strings.Contains(name, ".vol") && strings.HasSuffix(name, ".par2") {
			createdPaths = append(createdPaths, filepath.Join(dirPath, name))
		}
	}

	return createdPaths, nil
}

// parseRedundancyPercentage parses the redundancy config string into a percentage.
// Supports formats: "10", "10%", "1n*1.2" (ParPar formula).
func parseRedundancyPercentage(redundancy string, fileSize uint64, blockSize uint64) float64 {
	redundancy = strings.TrimSpace(redundancy)

	// Try ParPar formula: "Xn*Y" where X is multiplier and Y is factor
	if strings.Contains(redundancy, "n") {
		// Parse "1n*1.2" style formulas
		// This means: ceil(inputSlices * factor) recovery blocks
		// Convert to percentage
		parts := strings.SplitN(redundancy, "*", 2)
		if len(parts) == 2 {
			factor, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				// Convert to percentage: factor * 100 gives us the percentage
				// "1n*1.2" means 120% recovery blocks relative to input
				nParts := strings.SplitN(parts[0], "n", 2)
				multiplier := 1.0
				if nParts[0] != "" {
					if m, err := strconv.ParseFloat(nParts[0], 64); err == nil {
						multiplier = m
					}
				}
				return multiplier * factor * 100
			}
		}
	}

	// Try percentage: "10%" or "10"
	cleaned := strings.TrimSuffix(redundancy, "%")
	if pct, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return pct
	}

	// Default: 10%
	slog.Warn("Could not parse redundancy, defaulting to 10%", "redundancy", redundancy)
	return 10.0
}

// IsPar2File returns true if the given path matches a PAR2 file pattern.
func IsPar2File(path string) bool {
	return parregexp.MatchString(path)
}

// calculateParBlockSize calculates the appropriate PAR2 block size for the given file.
func calculateParBlockSize(fileSize uint64, articleSize uint64) uint64 {
	maxParBlocks := uint64(32768)

	if fileSize/articleSize < maxParBlocks {
		return articleSize
	}
	minParBlockSize := (fileSize / maxParBlocks) + 1
	multiplier := minParBlockSize / articleSize
	if minParBlockSize%articleSize != 0 {
		multiplier++
	}
	return multiplier * articleSize
}

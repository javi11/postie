package par2

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/progress"
	"github.com/javi11/postie/pkg/fileinfo"
)

// parparProgressRe matches parpar's progress output: "Finished          : 23.45%"
var parparProgressRe = regexp.MustCompile(`Finished\s*:\s*([\d.]+)%`)

// BinaryExecutor implements Par2Executor by shelling out to an external parpar binary.
type BinaryExecutor struct {
	articleSize uint64
	cfg         *config.Par2Config
	jobProgress progress.JobProgress
}

// NewBinaryExecutor creates a new BinaryExecutor.
func NewBinaryExecutor(articleSize uint64, cfg *config.Par2Config, jobProgress progress.JobProgress) *BinaryExecutor {
	return &BinaryExecutor{articleSize: articleSize, cfg: cfg, jobProgress: jobProgress}
}

// Create creates PAR2 parity files using the parpar binary.
func (b *BinaryExecutor) Create(ctx context.Context, files []fileinfo.FileInfo) ([]string, error) {
	return b.CreateInDirectory(ctx, files, "")
}

// CreateInDirectory creates PAR2 files in the specified output directory using the parpar binary.
func (b *BinaryExecutor) CreateInDirectory(ctx context.Context, files []fileinfo.FileInfo, outputDir string) ([]string, error) {
	var all []string
	for _, file := range files {
		if filepath.Ext(file.Path) == ".par2" {
			continue
		}
		dirPath := b.resolveDir(file, outputDir)
		if existing, ok := checkExistingPar2FilesInPath(ctx, file, dirPath); ok {
			all = append(all, existing...)
			continue
		}
		paths, err := b.runParpar(ctx, file, dirPath)
		if err != nil {
			return nil, err
		}
		all = append(all, paths...)
	}
	return all, nil
}

func (b *BinaryExecutor) resolveDir(file fileinfo.FileInfo, outputDir string) string {
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err == nil {
			return outputDir
		}
	} else if b.cfg.TempDir != "" {
		if err := os.MkdirAll(b.cfg.TempDir, 0755); err == nil {
			return b.cfg.TempDir
		}
	}
	return filepath.Dir(file.Path)
}

func (b *BinaryExecutor) runParpar(ctx context.Context, file fileinfo.FileInfo, dirPath string) ([]string, error) {
	blockSize := calculateParBlockSize(file.Size, b.articleSize)
	redundancyPct := parseRedundancyPercentage(b.cfg.Redundancy, file.Size, blockSize)
	numInputSlices := int(math.Ceil(float64(file.Size) / float64(blockSize)))
	if numInputSlices == 0 {
		numInputSlices = 1
	}
	numRecovery := max(int(math.Ceil(float64(numInputSlices)*redundancyPct/100.0)), 1)

	baseName := filepath.Base(file.Path)
	outputBase := filepath.Join(dirPath, baseName)

	args := []string{
		"-s", fmt.Sprintf("%db", blockSize),
		"-r", fmt.Sprintf("%d", numRecovery),
		"-o", outputBase,
	}
	args = append(args, b.cfg.ParparExtraArgs...)
	args = append(args, file.Path)

	// Progress: emit start
	progressID := uuid.New()
	progressName := fmt.Sprintf("PAR2: %s", baseName)
	var pg progress.Progress
	if b.jobProgress != nil {
		pg = b.jobProgress.AddProgress(progressID, progressName, progress.ProgressTypePar2Generation, 100)
	}

	slog.InfoContext(ctx, "Invoking parpar binary",
		"binary", b.cfg.ParparBinaryPath, "file", file.Path, "outputBase", outputBase,
		"blockSize", blockSize, "recoverySlices", numRecovery, "extraArgs", b.cfg.ParparExtraArgs)

	cmd := exec.CommandContext(ctx, b.cfg.ParparBinaryPath, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("parpar stdout pipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("parpar start for %s: %w", file.Path, err)
	}

	// Stream stdout, parse progress updates.
	// parpar writes progress as "Finished          : 23.45%\r" using carriage
	// returns to overwrite the line in place, so we split on both \r and \n.
	var stdoutBuf bytes.Buffer
	var lastPct int64
	scanner := bufio.NewScanner(io.TeeReader(stdoutPipe, &stdoutBuf))
	scanner.Split(splitOnCROrLF)
	for scanner.Scan() {
		if pg == nil {
			continue
		}
		if m := parparProgressRe.FindStringSubmatch(scanner.Text()); len(m) > 1 {
			if pct, parseErr := strconv.ParseFloat(m[1], 64); parseErr == nil {
				newPct := int64(pct)
				if newPct > lastPct {
					pg.UpdateProgress(newPct - lastPct)
					lastPct = newPct
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "Parpar cancelled", "file", file.Path)
			return nil, ctx.Err()
		}
		combined := strings.TrimSpace(stdoutBuf.String())
		if s := strings.TrimSpace(stderrBuf.String()); s != "" {
			if combined != "" {
				combined += "\n"
			}
			combined += s
		}
		return nil, fmt.Errorf("parpar failed for %s: %w\noutput: %s", file.Path, err, combined)
	}

	slog.InfoContext(ctx, "Parpar completed", "file", file.Path)

	// Ensure progress reaches 100% even if the last line wasn't parsed.
	if pg != nil && b.jobProgress != nil {
		if lastPct < 100 {
			pg.UpdateProgress(100 - lastPct)
		}
		b.jobProgress.FinishProgress(progressID)
	}

	// Collect output files
	var created []string
	mainPar2 := outputBase + ".par2"
	if _, statErr := os.Stat(mainPar2); statErr == nil {
		created = append(created, mainPar2)
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		slog.WarnContext(ctx, "Failed to read dir after parpar", "error", err)
		return created, nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, baseName) && strings.Contains(name, ".vol") && strings.HasSuffix(name, ".par2") {
			created = append(created, filepath.Join(dirPath, name))
		}
	}
	return created, nil
}

// splitOnCROrLF is a bufio.SplitFunc that splits on either \r or \n.
// parpar uses \r to overwrite the progress line in place on a terminal.
func splitOnCROrLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

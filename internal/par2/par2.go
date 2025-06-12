package par2

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/schollz/progressbar/v3"
)

type parExeType string

const (
	par2   parExeType = "par2"
	parpar parExeType = "parpar"
)

var (
	execCommand = exec.CommandContext
	parregexp   = regexp.MustCompile(`(?i)(\.vol\d+\+(\d+))?\.par2$`)
)

// ProgressCallback defines the interface for PAR2 progress notifications
type ProgressCallback func(stage string, current, total int64, details string, speed float64, secondsLeft float64, elapsedTime float64)

// Par2Executor defines the interface for executing par2 commands.
type Par2Executor interface {
	Create(ctx context.Context, tmpPath string) ([]string, error)
	SetProgressCallback(callback ProgressCallback)
}

// Par2CmdExecutor implements Par2Executor using the command line.
type Par2CmdExecutor struct {
	articleSize uint64
	cfg         *config.Par2Config
	parExeType  parExeType
	callback    ProgressCallback
}

func New(ctx context.Context, articleSize uint64, cfg *config.Par2Config) *Par2CmdExecutor {
	// detect par executable
	parExe := filepath.Base(cfg.Par2Path)
	parExeFileName := strings.ToLower(parExe[:len(parExe)-len(filepath.Ext(parExe))])

	return &Par2CmdExecutor{
		articleSize: articleSize,
		cfg:         cfg,
		parExeType:  parExeType(parExeFileName),
	}
}

// SetProgressCallback sets the progress callback function
func (p *Par2CmdExecutor) SetProgressCallback(callback ProgressCallback) {
	p.callback = callback
}

// Repair executes the par2 command to repair files in the target folder.
func (p *Par2CmdExecutor) Create(ctx context.Context, files []fileinfo.FileInfo) ([]string, error) {
	slog.InfoContext(ctx, "Starting par2 creation process", "executor", "Par2CmdExecutor")

	var (
		createdPar2Paths []string
		parameters       []string
	)
	for _, file := range files {
		if filepath.Ext(file.Path) == ".par2" {
			continue
		}

		parBlockSize := calculateParBlockSize(file.Size, p.articleSize)
		par2FileName := filepath.Base(file.Path) + ".par2"
		dirPath := filepath.Dir(file.Path)
		par2Path := filepath.Join(dirPath, par2FileName)

		// set parameters
		switch p.parExeType {
		case par2:
			parameters = append(parameters, "create", "-l")
			parameters = append(parameters, fmt.Sprintf("-s%v", parBlockSize))
			parameters = append(parameters, fmt.Sprintf("-r%v", p.cfg.Redundancy))
			parameters = append(parameters, fmt.Sprintf("%v", par2Path))
			parameters = append(parameters, p.cfg.ExtraPar2Options...)
			parameters = append(parameters, file.Path)
		case parpar:
			parameters = append(parameters, fmt.Sprintf("-p%vB", p.cfg.VolumeSize))
			parameters = append(parameters, fmt.Sprintf("-s%vB", parBlockSize))
			parameters = append(parameters, fmt.Sprintf("-r%v", p.cfg.Redundancy))
			parameters = append(parameters, fmt.Sprintf("-o%v", par2Path))
			parameters = append(parameters, "--overwrite")
			parameters = append(parameters, fmt.Sprintf("--slice-size-multiple=%vB", parBlockSize))
			parameters = append(parameters, fmt.Sprintf("--max-input-slices=%v", p.cfg.MaxInputSlices))
			parameters = append(parameters, p.cfg.ExtraPar2Options...)
			parameters = append(parameters, file.Path)
		default:
			return nil, fmt.Errorf("unknown par executable: %s", p.cfg.Par2Path)
		}

		// Use the package-level variable instead of calling exec.CommandContext directly
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}

		slog.DebugContext(ctx, fmt.Sprintf("Par command: %s in dir %s with parameters %v", p.cfg.Par2Path, dir, parameters))

		cmd := execCommand(ctx, p.cfg.Par2Path, parameters...)
		cmd.Dir = dir
		slog.DebugContext(ctx, fmt.Sprintf("Par command: %s in dir %s", cmd.String(), cmd.Dir))

		var cmdReader io.ReadCloser

		switch p.parExeType {
		case par2:
			if cmdReader, err = cmd.StdoutPipe(); err != nil {
				return nil, fmt.Errorf("failed to get stdout pipe for par2: %w", err)
			}
		case parpar:
			if cmdReader, err = cmd.StderrPipe(); err != nil {
				return nil, fmt.Errorf("failed to get stderr pipe for parpar: %w", err)
			}
		}

		scanner := bufio.NewScanner(cmdReader)
		scanner.Split(scanLines)

		wg := sync.WaitGroup{}
		var parProgressBar *progressbar.ProgressBar

		wg.Add(1)
		go func() {
			defer wg.Done()
			processing := false
			// Ensure parProgressBar is initialized before use
			parProgressBar = progressbar.NewOptions(100, // Use 100 as max for percentage
				progressbar.OptionSetDescription(fmt.Sprintf("Creating par2 files for %s...", file.Path)),
				progressbar.OptionSetRenderBlankState(true),
				progressbar.OptionThrottle(time.Millisecond*100),
				progressbar.OptionShowElapsedTimeOnFinish(),
				progressbar.OptionClearOnFinish(),
				progressbar.OptionOnCompletion(func() {
					// new line after progress bar
					fmt.Println()
				}),
			)
			defer func() {
				_ = parProgressBar.Finish() // Ensure finish is called on success
				_ = parProgressBar.Close()  // Close the progress bar when done
				fmt.Println()
			}()

			// Notify callback about PAR2 start
			if p.callback != nil {
				p.callback("par2_creating", 0, 100, fmt.Sprintf("Creating PAR2 files for %s", filepath.Base(file.Path)), 0.0, 0.0, 0.0)
			}

			for scanner.Scan() {
				output := strings.Trim(scanner.Text(), " \r\n")
				if output != "" && !strings.Contains(output, "%") {
					slog.DebugContext(ctx, fmt.Sprintf("PAR2 STDOUT: %v", output))
				}

				exp := regexp.MustCompile(`(\d+)\.?\d*%`)
				if output != "" && exp.MatchString(output) {
					if !processing && strings.Contains(output, "Processing:") {
						processing = true
						if parProgressBar != nil {
							_ = parProgressBar.Finish() // Close the progress bar when done
							_ = parProgressBar.Close()
							parProgressBar = nil
						}

						parProgressBar = progressbar.NewOptions(100, // Use 100 as max for percentage
							progressbar.OptionSetDescription(fmt.Sprintf("Processing par2 files for %s...", file.Path)),
							progressbar.OptionSetRenderBlankState(true),
							progressbar.OptionThrottle(time.Millisecond*100),
							progressbar.OptionShowElapsedTimeOnFinish(),
							progressbar.OptionClearOnFinish(),
							progressbar.OptionOnCompletion(func() {
								// new line after progress bar
								fmt.Println()
							}),
						)

						// Notify callback about processing start
						if p.callback != nil {
							p.callback("par2_processing", 0, 100, fmt.Sprintf("Processing PAR2 files for %s", filepath.Base(file.Path)), 0.0, 0.0, 0.0)
						}
					}

					percentStr := exp.FindStringSubmatch(output)
					if len(percentStr) > 1 {
						percentInt, err := strconv.Atoi(percentStr[1])
						if err == nil {
							err = parProgressBar.Set(percentInt)
							if err != nil {
								slog.ErrorContext(ctx, "Error setting progress bar", "error", err)
							}

							// Notify callback with current progress
							if p.callback != nil {
								stage := "par2_creating"
								if processing {
									stage = "par2_processing"
								}
								details := fmt.Sprintf("PAR2 %s: %d%% complete", stage, percentInt)
								p.callback(stage, int64(percentInt), 100, details, 0.0, 0.0, 0.0)
							}
						}
					}
				}
			}

		}()

		if err = cmd.Run(); err != nil {
			if ctx.Err() == context.Canceled {
				slog.InfoContext(ctx, "Par2 creation cancelled", "path", file.Path)
				return nil, ctx.Err()
			}

			return nil, fmt.Errorf("failed to run par2 command '%s': %w", cmd.String(), err)
		}

		wg.Wait()

		// Check if PAR2 creation was successful
		if _, err := os.Stat(par2Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("par2 file was not created: %s", par2Path)
		}

		createdPar2Paths = append(createdPar2Paths, par2Path)

		// Notify callback about PAR2 completion
		if p.callback != nil {
			p.callback("par2_completed", 100, 100, fmt.Sprintf("PAR2 creation completed for %s", filepath.Base(file.Path)), 0.0, 0.0, 0.0)
		}

		slog.InfoContext(ctx, "Par2 creation completed successfully")

		// After successful creation, collect all par2 volume files
		baseName := filepath.Base(file.Path)

		// Find all volume files
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			slog.WarnContext(ctx, "Failed to read directory to find par2 volumes", "error", err)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			// Match patterns like .vol0+1.par2, .vol1+1.par2, etc.
			if strings.HasPrefix(name, baseName) && strings.Contains(name, ".vol") && strings.HasSuffix(name, ".par2") {
				createdPar2Paths = append(createdPar2Paths, filepath.Join(dirPath, name))
			}
		}
	}

	return createdPar2Paths, nil
}

func IsPar2File(path string) bool {
	return parregexp.MatchString(path)
}

// scanLines is a helper for bufio.Scanner to split lines correctly
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		if data[i] == '\n' {
			// We have a line terminated by single newline.
			return i + 1, data[0:i], nil
		}

		advance = i + 1
		if len(data) > i+1 && data[i+1] == '\n' {
			advance += 1
		}

		return advance, data[0:i], nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

func calculateParBlockSize(fileSize uint64, articleSize uint64) uint64 {
	maxParBlocks := uint64(32768)

	if fileSize/articleSize < maxParBlocks {
		return articleSize
	} else {
		minParBlockSize := (fileSize / maxParBlocks) + 1
		multiplier := minParBlockSize / articleSize
		if minParBlockSize%articleSize != 0 {
			multiplier++
		}
		return multiplier * articleSize
	}
}

package par2

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

var (
	execCommand = exec.CommandContext
	parregexp   = regexp.MustCompile(`(?i)(\.vol\d+\+(\d+))?\.par2$`)

	// par2 exit codes
	par2ExitCodes = map[int]string{
		0: "Success",
		1: "Repair possible",
		2: "Repair not possible",
		3: "Invalid command line arguments",
		4: "Insufficient critical data to verify",
		5: "Repair failed",
		6: "FileIO Error",
		7: "Logic Error",
		8: "Out of memory",
	}
)

// Par2Executor defines the interface for executing par2 commands.
type Par2Executor interface {
	Create(ctx context.Context, tmpPath string) ([]string, error)
}

// Par2CmdExecutor implements Par2Executor using the command line.
type Par2CmdExecutor struct {
	ExePath     string
	articleSize int64
}

func New(ctx context.Context, par2ExePath string, articleSize int64) *Par2CmdExecutor {
	return &Par2CmdExecutor{
		ExePath:     par2ExePath,
		articleSize: articleSize,
	}
}

// Repair executes the par2 command to repair files in the target folder.
func (p *Par2CmdExecutor) Create(ctx context.Context, files []string) ([]string, error) {
	slog.InfoContext(ctx, "Starting par2 creation process", "executor", "Par2CmdExecutor")

	par2Exe := p.ExePath
	if par2Exe == "" {
		par2Exe = "par2" // Default if path is empty
		slog.WarnContext(ctx, "Par2 executable path is empty, defaulting to 'par2'")
	}

	var createdPar2Paths []string
	for _, file := range files {
		par2FileName := filepath.Base(file) + ".par2"
		dirPath := filepath.Dir(file)
		par2Path := filepath.Join(dirPath, par2FileName)

		// set parameters
		parameters := append(
			[]string{},
			"create",
			// blocksize equals to article size
			"-s"+strconv.FormatInt(p.articleSize, 10),
			// 10% recovery data
			"-r10",
			par2Path,
			file,
		)

		// Use the package-level variable instead of calling exec.CommandContext directly
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}

		cmd := execCommand(ctx, par2Exe, parameters...)
		cmd.Dir = dir
		slog.DebugContext(ctx, fmt.Sprintf("Par command: %s in dir %s", cmd.String(), cmd.Dir))

		cmdErr, err := cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stderr pipe for par2: %w", err)
		}

		// create a pipe for the output of the program
		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe for par2: %w", err)
		}

		scanner := bufio.NewScanner(cmdReader)
		scanner.Split(scanLines)

		errScanner := bufio.NewScanner(cmdErr)
		errScanner.Split(scanLines)

		var stderrOutput strings.Builder

		mu := sync.Mutex{}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for errScanner.Scan() {
				line := strings.TrimSpace(errScanner.Text())
				if line != "" {
					slog.DebugContext(ctx, "PAR2 STDERR:", "line", line)
					mu.Lock()
					stderrOutput.WriteString(line + "\n")
					mu.Unlock()
				}
			}
		}()

		var parProgressBar *progressbar.ProgressBar

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Ensure parProgressBar is initialized before use
			parProgressBar = progressbar.NewOptions(100, // Use 100 as max for percentage
				progressbar.OptionSetDescription(fmt.Sprintf("Creating par2 files for %s...", file)),
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
				_ = parProgressBar.Close() // Close the progress bar when done
			}()

			for scanner.Scan() {
				output := strings.Trim(scanner.Text(), " \r\n")
				if output != "" && !strings.Contains(output, "%") {
					slog.DebugContext(ctx, fmt.Sprintf("PAR2 STDOUT: %v", output))
				}

				exp := regexp.MustCompile(`(\d+)\.?\d*%`)
				if output != "" && exp.MatchString(output) {
					percentStr := exp.FindStringSubmatch(output)
					if len(percentStr) > 1 {
						percentInt, err := strconv.Atoi(percentStr[1])
						if err == nil {
							_ = parProgressBar.Set(percentInt)
						}
					}
				}
			}

		}()

		if err = cmd.Run(); err != nil {
			mu.Lock()
			output := stderrOutput.String()
			mu.Unlock()

			if exitError, ok := err.(*exec.ExitError); ok {
				if parProgressBar != nil {
					_ = parProgressBar.Close() // Attempt to close/clear on error too
				}

				if errMsg, ok := par2ExitCodes[exitError.ExitCode()]; ok {
					// Specific known error codes from par2
					fullErrMsg := fmt.Sprintf("par2 exited with code %d: %s. Stderr: %s", exitError.ExitCode(), errMsg, output)
					slog.ErrorContext(ctx, fullErrMsg)
					// Treat specific codes as potentially non-fatal or requiring different handling
					// For now, return all as errors, but could customize (e.g., ignore exit code 1 if repair was possible)
					return nil, errors.New(fullErrMsg)
				}
				// Unknown exit code
				unknownErrMsg := fmt.Sprintf("par2 exited with unknown code %d. Stderr: %s", exitError.ExitCode(), output)
				slog.ErrorContext(ctx, unknownErrMsg)
				return nil, errors.New(unknownErrMsg)
			}
			// Error not related to exit code (e.g., command not found)
			return nil, fmt.Errorf("failed to run par2 command '%s': %w. Stderr: %s", cmd.String(), err, output)
		}

		if parProgressBar != nil {
			_ = parProgressBar.Finish() // Ensure finish is called on success
		}

		wg.Wait()

		slog.InfoContext(ctx, "Par2 creation completed successfully")

		// After successful creation, collect all par2 volume files
		baseName := filepath.Base(file)

		// Add the main par2 file
		createdPar2Paths = append(createdPar2Paths, par2Path)

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

package par2

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/fileinfo"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	articleSize := uint64(512 * 1024) // 512 KB
	cfg := &config.Par2Config{
		Par2Path:         "/usr/bin/par2",
		Redundancy:       "10",
		VolumeSize:       256 * 1024, // 256 KB
		MaxInputSlices:   2000,
		ExtraPar2Options: []string{"-q"},
	}

	par2Executor := New(ctx, articleSize, cfg)

	if par2Executor == nil {
		t.Fatal("Expected non-nil Par2CmdExecutor")
	}

	if par2Executor.articleSize != articleSize {
		t.Errorf("Expected articleSize to be %d, got %d", articleSize, par2Executor.articleSize)
	}

	if par2Executor.cfg != cfg {
		t.Errorf("Expected cfg to match the provided config")
	}

	if par2Executor.parExeType != par2 {
		t.Errorf("Expected parExeType to be %s, got %s", par2, par2Executor.parExeType)
	}
}

func TestNewWithParpar(t *testing.T) {
	ctx := context.Background()
	articleSize := uint64(512 * 1024) // 512 KB
	cfg := &config.Par2Config{
		Par2Path:         "/usr/bin/parpar",
		Redundancy:       "10",
		VolumeSize:       256 * 1024, // 256 KB
		MaxInputSlices:   2000,
		ExtraPar2Options: []string{"-q"},
	}

	par2Executor := New(ctx, articleSize, cfg)

	if par2Executor == nil {
		t.Fatal("Expected non-nil Par2CmdExecutor")
	}

	if par2Executor.parExeType != parpar {
		t.Errorf("Expected parExeType to be %s, got %s", parpar, par2Executor.parExeType)
	}
}

func TestIsPar2File(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"file.par2", true},
		{"file.vol0+1.par2", true},
		{"file.vol1+2.par2", true},
		{"file.PAR2", true}, // Case insensitive
		{"file.VOL0+1.PAR2", true},
		{"file.txt", false},
		{"file.par", false},
		{"file.par2.txt", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := IsPar2File(tc.path)
			if result != tc.expected {
				t.Errorf("IsPar2File(%s) expected %v, got %v", tc.path, tc.expected, result)
			}
		})
	}
}

func TestCalculateParBlockSize(t *testing.T) {
	testCases := []struct {
		fileSize    uint64
		articleSize uint64
		expected    uint64
	}{
		{10 * 1024 * 1024, 512 * 1024, 512 * 1024},         // 10MB with 512KB article size
		{32768*512*1024 + 1, 512 * 1024, 512 * 1024 * 2},   // Just over the max par blocks threshold
		{32768*512*1024 - 1, 512 * 1024, 512 * 1024},       // Just under the max par blocks threshold
		{32768*512*1024*3 + 1, 512 * 1024, 512 * 1024 * 4}, // Well over the threshold
		{1024, 512 * 1024, 512 * 1024},                     // Very small file
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case %d", i), func(t *testing.T) {
			result := calculateParBlockSize(tc.fileSize, tc.articleSize)
			if result != tc.expected {
				t.Errorf("calculateParBlockSize(%d, %d) expected %d, got %d",
					tc.fileSize, tc.articleSize, tc.expected, result)
			}
		})
	}
}

func TestScanLines(t *testing.T) {
	testCases := []struct {
		name            string
		data            []byte
		atEOF           bool
		expectedAdvance int
		expectedToken   []byte
		expectedErr     error
	}{
		{
			name:            "Unix line ending",
			data:            []byte("line1\nline2"),
			atEOF:           false,
			expectedAdvance: 6,
			expectedToken:   []byte("line1"),
			expectedErr:     nil,
		},
		{
			name:            "Windows line ending",
			data:            []byte("line1\r\nline2"),
			atEOF:           false,
			expectedAdvance: 7,
			expectedToken:   []byte("line1"),
			expectedErr:     nil,
		},
		{
			name:            "Carriage return only",
			data:            []byte("line1\rline2"),
			atEOF:           false,
			expectedAdvance: 6,
			expectedToken:   []byte("line1"),
			expectedErr:     nil,
		},
		{
			name:            "EOF without newline",
			data:            []byte("lastline"),
			atEOF:           true,
			expectedAdvance: 8,
			expectedToken:   []byte("lastline"),
			expectedErr:     nil,
		},
		{
			name:            "Empty data at EOF",
			data:            []byte{},
			atEOF:           true,
			expectedAdvance: 0,
			expectedToken:   nil,
			expectedErr:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			advance, token, err := scanLines(tc.data, tc.atEOF)

			if tc.expectedErr == nil && err != nil {
				t.Errorf("Expected no error, got %v", err)
			} else if tc.expectedErr != nil && err == nil {
				t.Errorf("Expected error %v, got nil", tc.expectedErr)
			}

			if advance != tc.expectedAdvance {
				t.Errorf("Expected advance %d, got %d", tc.expectedAdvance, advance)
			}

			if (token == nil && tc.expectedToken != nil) ||
				(token != nil && tc.expectedToken == nil) ||
				(token != nil && tc.expectedToken != nil && string(token) != string(tc.expectedToken)) {
				t.Errorf("Expected token %q, got %q", tc.expectedToken, token)
			}
		})
	}
}

// Let's create a different approach for the mock command
// Using the actual exec.Command pattern in Go for testing

func TestCreatePar2WithPar2Command(t *testing.T) {
	// Save and restore the original execCommand
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the exec.CommandContext
	execCommand = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		// Use a fake "echo" command instead that will just succeed
		cs := []string{"-n", "Processing: 10%\nProcessing: 50%\nProcessing: 100%"}
		cmd := exec.CommandContext(ctx, "echo", cs...)
		return cmd
	}

	// Create a Par2CmdExecutor with the mock command
	par2Executor := &Par2CmdExecutor{
		articleSize: 512 * 1024,
		cfg: &config.Par2Config{
			Par2Path:         "/usr/bin/par2",
			Redundancy:       "10",
			VolumeSize:       256 * 1024,
			MaxInputSlices:   2000,
			ExtraPar2Options: []string{"-q"},
		},
		parExeType: par2,
	}

	files := []fileinfo.FileInfo{
		{
			Path: testFile,
			Size: 1024 * 1024, // 1MB
		},
	}

	// Rather than trying to fully mock the command execution,
	// we'll mock the function that's called so that it returns a simple
	// successful command that outputs the expected format
	_, err = par2Executor.Create(ctx, files)
	if err != nil {
		t.Fatalf("par2Executor.Create failed: %v", err)
	}
}

func TestCreatePar2WithParparCommand(t *testing.T) {
	// Save and restore the original execCommand
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the exec.CommandContext
	execCommand = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		// Use a fake command that will succeed
		cs := []string{"-n", "Processing: 10%\nProcessing: 50%\nProcessing: 100%"}
		cmd := exec.CommandContext(ctx, "echo", cs...)
		return cmd
	}

	// Create a Par2CmdExecutor with the mock command
	par2Executor := &Par2CmdExecutor{
		articleSize: 512 * 1024,
		cfg: &config.Par2Config{
			Par2Path:         "/usr/bin/parpar",
			Redundancy:       "10",
			VolumeSize:       256 * 1024,
			MaxInputSlices:   2000,
			ExtraPar2Options: []string{"-q"},
		},
		parExeType: parpar,
	}

	files := []fileinfo.FileInfo{
		{
			Path: testFile,
			Size: 1024 * 1024, // 1MB
		},
	}

	_, err = par2Executor.Create(ctx, files)
	if err != nil {
		t.Fatalf("par2Executor.Create failed: %v", err)
	}
}

func TestCreatePar2CommandFailed(t *testing.T) {
	// Save and restore the original execCommand
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the exec.CommandContext to return an error by using a non-existent command
	execCommand = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		// Command will fail because it doesn't exist
		return exec.CommandContext(ctx, "nonexistentcommand")
	}

	// Create a Par2CmdExecutor with the mock command
	par2Executor := &Par2CmdExecutor{
		articleSize: 512 * 1024,
		cfg: &config.Par2Config{
			Par2Path:         "/usr/bin/par2",
			Redundancy:       "10",
			VolumeSize:       256 * 1024,
			MaxInputSlices:   2000,
			ExtraPar2Options: []string{"-q"},
		},
		parExeType: par2,
	}

	files := []fileinfo.FileInfo{
		{
			Path: testFile,
			Size: 1024 * 1024, // 1MB
		},
	}

	_, err = par2Executor.Create(ctx, files)
	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
}

// MockCmd is a mock implementation of exec.Cmd
type MockCmd struct {
	*exec.Cmd   // embed the real Cmd to satisfy the interface
	args        []string
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	returnError error
}

// StdoutPipe returns a mock stdout pipe
func (m *MockCmd) StdoutPipe() (io.ReadCloser, error) {
	return m.stdout, nil
}

// StderrPipe returns a mock stderr pipe
func (m *MockCmd) StderrPipe() (io.ReadCloser, error) {
	return m.stderr, nil
}

// Run runs the mock command
func (m *MockCmd) Run() error {
	return m.returnError
}

// String gets the command as a string
func (m *MockCmd) String() string {
	return fmt.Sprintf("mock-cmd %s", strings.Join(m.args, " "))
}

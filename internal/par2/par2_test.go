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
	"time"

	"github.com/google/uuid"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/pausable"
	"github.com/javi11/postie/internal/progress"
	"github.com/javi11/postie/pkg/fileinfo"
)

// mockProgress implements the Progress interface for testing
type mockProgress struct {
	id         uuid.UUID
	name       string
	progType   progress.ProgressType
	total      int64
	current    int64
	isComplete bool
	isPaused   bool
	startTime  time.Time
}

func (m *mockProgress) UpdateProgress(processed int64) {
	m.current += processed
	if m.current >= m.total {
		m.isComplete = true
	}
}

func (m *mockProgress) Finish() {
	m.isComplete = true
	m.current = m.total
}

func (m *mockProgress) GetState() progress.ProgressState {
	return progress.ProgressState{
		Max:            m.total,
		CurrentNum:     m.current,
		CurrentPercent: m.GetPercentage(),
		CurrentBytes:   float64(m.current),
		SecondsSince:   time.Since(m.startTime).Seconds(),
		SecondsLeft:    0, // Simplified for testing
		KBsPerSecond:   0, // Simplified for testing
		Description:    m.name,
		Type:           m.progType,
		IsStarted:      true,
		IsPaused:       m.isPaused,
	}
}

func (m *mockProgress) GetID() uuid.UUID               { return m.id }
func (m *mockProgress) GetName() string                { return m.name }
func (m *mockProgress) GetType() progress.ProgressType { return m.progType }
func (m *mockProgress) GetCurrent() int64              { return m.current }
func (m *mockProgress) GetTotal() int64                { return m.total }
func (m *mockProgress) GetPercentage() float64 {
	if m.total == 0 {
		return 0
	}
	return float64(m.current) / float64(m.total) * 100
}
func (m *mockProgress) IsComplete() bool        { return m.isComplete }
func (m *mockProgress) GetStartTime() time.Time { return m.startTime }
func (m *mockProgress) GetElapsedTime() time.Duration {
	return time.Since(m.startTime)
}
func (m *mockProgress) SetPaused(paused bool) { m.isPaused = paused }
func (m *mockProgress) IsPaused() bool        { return m.isPaused }

// mockJobProgress implements the JobProgress interface for testing
type mockJobProgress struct {
	progresses map[uuid.UUID]*mockProgress
}

func newMockJobProgress() *mockJobProgress {
	return &mockJobProgress{
		progresses: make(map[uuid.UUID]*mockProgress),
	}
}

func (m *mockJobProgress) AddProgress(id uuid.UUID, name string, pType progress.ProgressType, total int64) progress.Progress {
	prog := &mockProgress{
		id:        id,
		name:      name,
		progType:  pType,
		total:     total,
		current:   0,
		startTime: time.Now(),
	}
	m.progresses[id] = prog
	return prog
}

func (m *mockJobProgress) FinishProgress(id uuid.UUID) {
	if prog, exists := m.progresses[id]; exists {
		prog.Finish()
	}
}

func (m *mockJobProgress) GetProgress(id uuid.UUID) progress.Progress {
	return m.progresses[id]
}

func (m *mockJobProgress) GetAllProgress() map[uuid.UUID]progress.Progress {
	result := make(map[uuid.UUID]progress.Progress)
	for id, prog := range m.progresses {
		result[id] = prog
	}
	return result
}

func (m *mockJobProgress) GetAllProgressState() []progress.ProgressState {
	var states []progress.ProgressState
	for _, prog := range m.progresses {
		states = append(states, prog.GetState())
	}
	return states
}

func (m *mockJobProgress) GetJobID() string { return "test-job" }
func (m *mockJobProgress) Close()           {}
func (m *mockJobProgress) SetAllPaused(paused bool) {
	for _, prog := range m.progresses {
		prog.SetPaused(paused)
	}
}

func TestNew(t *testing.T) {
	t.Run("with regular context", func(t *testing.T) {
		ctx := context.Background()
		articleSize := uint64(512 * 1024) // 512 KB
		cfg := &config.Par2Config{
			Par2Path:         "/usr/bin/par2",
			Redundancy:       "10",
			VolumeSize:       256 * 1024, // 256 KB
			MaxInputSlices:   2000,
			ExtraPar2Options: []string{"-q"},
		}

		par2Executor := New(ctx, articleSize, cfg, nil)

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
	})

	t.Run("with pausable context", func(t *testing.T) {
		ctx := pausable.NewContext(context.Background())
		articleSize := uint64(512 * 1024) // 512 KB
		cfg := &config.Par2Config{
			Par2Path:         "/usr/bin/par2",
			Redundancy:       "10",
			VolumeSize:       256 * 1024, // 256 KB
			MaxInputSlices:   2000,
			ExtraPar2Options: []string{"-q"},
		}

		par2Executor := New(ctx, articleSize, cfg, nil)

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
	})
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

	par2Executor := New(ctx, articleSize, cfg, nil)

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
	// Save and restore the original commandFunc
	originalCommandFunc := commandFunc
	defer func() { commandFunc = originalCommandFunc }()

	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the commandFunc to create a script that creates the expected par2 file
	commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		// Find the output file from the args (the par2 file path for par2 command)
		var outputFile string
		if len(args) >= 4 {
			// For par2 command, the output file is typically one of the later arguments
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg
					break
				}
			}
		}

		// Create a shell script that creates the par2 file and outputs progress
		script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 10%%"
echo "Processing: 50%%"  
echo "Processing: 100%%"
`, outputFile)

		cmd := exec.CommandContext(ctx, "sh", "-c", script)
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
		parExeType:  par2,
		jobProgress: newMockJobProgress(),
	}

	files := []fileinfo.FileInfo{
		{
			Path: testFile,
			Size: 1024 * 1024, // 1MB
		},
	}

	createdFiles, err := par2Executor.Create(ctx, files)
	if err != nil {
		t.Fatalf("par2Executor.Create failed: %v", err)
	}

	// Verify that at least one par2 file was created
	if len(createdFiles) == 0 {
		t.Fatal("Expected at least one par2 file to be created")
	}

	// Verify the created file exists
	expectedPar2File := testFile + ".par2"
	if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
		t.Fatalf("Expected par2 file %s was not created", expectedPar2File)
	}

	// Verify the created file is in the returned list
	found := false
	for _, createdFile := range createdFiles {
		if createdFile == expectedPar2File {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected par2 file %s not found in created files list: %v", expectedPar2File, createdFiles)
	}
}

func TestCreatePar2WithParparCommand(t *testing.T) {
	// Save and restore the original commandFunc
	originalCommandFunc := commandFunc
	defer func() { commandFunc = originalCommandFunc }()

	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the commandFunc to create a script that creates the expected par2 file
	commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		// Find the output file from the args (after -o flag for parpar)
		var outputFile string
		for _, arg := range args {
			if strings.HasPrefix(arg, "-o") {
				outputFile = strings.TrimPrefix(arg, "-o")
				break
			}
		}

		// Create a shell script that creates the par2 file and outputs progress
		script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 10%%"
echo "Processing: 50%%"  
echo "Processing: 100%%"
`, outputFile)

		cmd := exec.CommandContext(ctx, "sh", "-c", script)
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
		parExeType:  parpar,
		jobProgress: newMockJobProgress(),
	}

	files := []fileinfo.FileInfo{
		{
			Path: testFile,
			Size: 1024 * 1024, // 1MB
		},
	}

	createdFiles, err := par2Executor.Create(ctx, files)
	if err != nil {
		t.Fatalf("par2Executor.Create failed: %v", err)
	}

	// Verify that at least one par2 file was created
	if len(createdFiles) == 0 {
		t.Fatal("Expected at least one par2 file to be created")
	}

	// Verify the created file exists
	expectedPar2File := testFile + ".par2"
	if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
		t.Fatalf("Expected par2 file %s was not created", expectedPar2File)
	}

	// Verify the created file is in the returned list
	found := false
	for _, createdFile := range createdFiles {
		if createdFile == expectedPar2File {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected par2 file %s not found in created files list: %v", expectedPar2File, createdFiles)
	}
}

func TestCreatePar2CommandFailed(t *testing.T) {
	// Save and restore the original commandFunc
	originalCommandFunc := commandFunc
	defer func() { commandFunc = originalCommandFunc }()

	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the commandFunc to return an error by using a non-existent command
	commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
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
		parExeType:  par2,
		jobProgress: newMockJobProgress(),
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

func TestCreatePar2WithPausableContext(t *testing.T) {
	// Save and restore the original commandFunc
	originalCommandFunc := commandFunc
	defer func() { commandFunc = originalCommandFunc }()

	pausableCtx := pausable.NewContext(context.Background())
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.bin")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock the commandFunc to create a script that creates the expected par2 file
	commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		// Find the output file from the args (the par2 file path for par2 command)
		var outputFile string
		if len(args) >= 4 {
			// For par2 command, the output file is typically one of the later arguments
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg
					break
				}
			}
		}

		// Create a shell script that creates the par2 file and outputs progress
		script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 10%%"
echo "Processing: 50%%"  
echo "Processing: 100%%"
`, outputFile)

		cmd := exec.CommandContext(ctx, "sh", "-c", script)
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
		parExeType:  par2,
		jobProgress: newMockJobProgress(),
	}

	files := []fileinfo.FileInfo{
		{
			Path: testFile,
			Size: 1024 * 1024, // 1MB
		},
	}

	// Test with pausable context - should work the same as regular context
	createdFiles, err := par2Executor.Create(pausableCtx, files)
	if err != nil {
		t.Fatalf("par2Executor.Create failed: %v", err)
	}

	// Verify that at least one par2 file was created
	if len(createdFiles) == 0 {
		t.Fatal("Expected at least one par2 file to be created")
	}

	// Verify the created file exists
	expectedPar2File := testFile + ".par2"
	if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
		t.Fatalf("Expected par2 file %s was not created", expectedPar2File)
	}

	// Verify the created file is in the returned list
	found := false
	for _, createdFile := range createdFiles {
		if createdFile == expectedPar2File {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected par2 file %s not found in created files list: %v", expectedPar2File, createdFiles)
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

func TestCreateInDirectory(t *testing.T) {
	t.Run("creates PAR2 files in specified output directory", func(t *testing.T) {
		// Save and restore the original commandFunc
		originalCommandFunc := commandFunc
		defer func() { commandFunc = originalCommandFunc }()

		ctx := context.Background()
		sourceDir := t.TempDir()
		outputDir := t.TempDir()
		testFile := filepath.Join(sourceDir, "testfile.bin")

		// Create a test file
		err := os.WriteFile(testFile, []byte("test content for par2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Mock the commandFunc to create par2 files in the output directory
		commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
			var outputFile string
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg
					break
				}
			}

			// Create mock PAR2 files (main + volume)
			script := fmt.Sprintf(`#!/bin/bash
touch "%s"
touch "%s.vol0+1.par2"
echo "Processing: 100%%"`, outputFile, strings.TrimSuffix(outputFile, ".par2"))

			return exec.CommandContext(ctx, "sh", "-c", script)
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 1024 * 1024},
		}

		createdFiles, err := par2Executor.CreateInDirectory(ctx, files, outputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		// Verify files were created in output directory, not source directory
		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		expectedPar2File := filepath.Join(outputDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in output directory", expectedPar2File)
		}

		// Verify it's NOT in the source directory
		sourcePar2File := filepath.Join(sourceDir, "testfile.bin.par2")
		if _, err := os.Stat(sourcePar2File); err == nil {
			t.Fatalf("PAR2 file should not be in source directory %s", sourcePar2File)
		}

		// Verify the created file is in the returned list
		found := false
		for _, createdFile := range createdFiles {
			if createdFile == expectedPar2File {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected PAR2 file %s not found in created files list: %v", expectedPar2File, createdFiles)
		}
	})

	t.Run("uses default behavior when outputDir is empty", func(t *testing.T) {
		originalCommandFunc := commandFunc
		defer func() { commandFunc = originalCommandFunc }()

		ctx := context.Background()
		tempDir := t.TempDir()
		configTempDir := filepath.Join(tempDir, "temp")
		err := os.MkdirAll(configTempDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}

		testFile := filepath.Join(tempDir, "testfile.bin")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
			var outputFile string
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg
					break
				}
			}

			script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 100%%"`, outputFile)

			return exec.CommandContext(ctx, "sh", "-c", script)
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path: "/usr/bin/par2",
				Redundancy: "10",
				TempDir: configTempDir,
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 1024 * 1024},
		}

		createdFiles, err := par2Executor.CreateInDirectory(ctx, files, "") // empty outputDir
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		// Should be created in configTempDir (default behavior)
		expectedPar2File := filepath.Join(configTempDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in config temp directory", expectedPar2File)
		}
	})

	t.Run("reuses existing PAR2 files in output directory", func(t *testing.T) {
		ctx := context.Background()
		sourceDir := t.TempDir()
		outputDir := t.TempDir()
		testFile := filepath.Join(sourceDir, "testfile.bin")

		// Create a test file
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Pre-create PAR2 files in output directory
		mainPar2 := filepath.Join(outputDir, "testfile.bin.par2")
		volPar2 := filepath.Join(outputDir, "testfile.bin.vol0+1.par2")
		err = os.WriteFile(mainPar2, []byte("par2 data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create main PAR2 file: %v", err)
		}
		err = os.WriteFile(volPar2, []byte("volume data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create volume PAR2 file: %v", err)
		}

		// Track if command was called
		commandCalled := false
		originalCommandFunc := commandFunc
		defer func() { commandFunc = originalCommandFunc }()

		commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
			commandCalled = true
			return exec.CommandContext(ctx, "echo", "should not be called")
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 1024 * 1024},
		}

		createdFiles, err := par2Executor.CreateInDirectory(ctx, files, outputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		// Verify existing files were returned
		if len(createdFiles) < 2 {
			t.Fatalf("Expected at least 2 existing PAR2 files to be returned, got %d", len(createdFiles))
		}

		// Verify command was NOT called (files already existed)
		if commandCalled {
			t.Fatal("PAR2 command should not have been called when files already exist")
		}

		// Verify returned paths include existing files
		foundMain := false
		foundVol := false
		for _, path := range createdFiles {
			if path == mainPar2 {
				foundMain = true
			}
			if path == volPar2 {
				foundVol = true
			}
		}

		if !foundMain {
			t.Fatal("Main PAR2 file not found in returned paths")
		}
		if !foundVol {
			t.Fatal("Volume PAR2 file not found in returned paths")
		}
	})

	t.Run("falls back when output directory creation fails", func(t *testing.T) {
		originalCommandFunc := commandFunc
		defer func() { commandFunc = originalCommandFunc }()

		ctx := context.Background()
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")

		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Use an invalid output directory (file path instead of directory)
		invalidOutputDir := filepath.Join(tempDir, "file.txt")
		err = os.WriteFile(invalidOutputDir, []byte("not a directory"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid output path: %v", err)
		}

		commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
			var outputFile string
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg
					break
				}
			}

			script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 100%%"`, outputFile)

			return exec.CommandContext(ctx, "sh", "-c", script)
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 1024 * 1024},
		}

		// Should fall back to source file directory
		createdFiles, err := par2Executor.CreateInDirectory(ctx, files, invalidOutputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		// Verify it fell back to source directory
		expectedPar2File := filepath.Join(tempDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in fallback directory", expectedPar2File)
		}
	})

	t.Run("creates nested output directory structure", func(t *testing.T) {
		originalCommandFunc := commandFunc
		defer func() { commandFunc = originalCommandFunc }()

		ctx := context.Background()
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")

		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Nested directory that doesn't exist yet
		nestedOutputDir := filepath.Join(tempDir, "output", "nested", "deep")

		commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
			var outputFile string
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg
					break
				}
			}

			script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 100%%"`, outputFile)

			return exec.CommandContext(ctx, "sh", "-c", script)
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 1024 * 1024},
		}

		createdFiles, err := par2Executor.CreateInDirectory(ctx, files, nestedOutputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		// Verify nested directory was created
		if _, err := os.Stat(nestedOutputDir); os.IsNotExist(err) {
			t.Fatalf("Nested output directory %s was not created", nestedOutputDir)
		}

		// Verify PAR2 file was created in nested directory
		expectedPar2File := filepath.Join(nestedOutputDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in nested directory", expectedPar2File)
		}
	})

	t.Run("creates PAR2 files for multiple source files", func(t *testing.T) {
		originalCommandFunc := commandFunc
		defer func() { commandFunc = originalCommandFunc }()

		ctx := context.Background()
		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		// Create multiple test files
		testFile1 := filepath.Join(sourceDir, "file1.bin")
		testFile2 := filepath.Join(sourceDir, "file2.bin")
		testFile3 := filepath.Join(sourceDir, "file3.bin")

		err := os.WriteFile(testFile1, []byte("content 1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file 1: %v", err)
		}
		err = os.WriteFile(testFile2, []byte("content 2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file 2: %v", err)
		}
		err = os.WriteFile(testFile3, []byte("content 3"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file 3: %v", err)
		}

		commandFunc = func(ctx context.Context, command string, args ...string) *exec.Cmd {
			// Find the LAST .par2 file in args (parameters accumulate across iterations)
			var outputFile string
			for _, arg := range args {
				if strings.HasSuffix(arg, ".par2") {
					outputFile = arg // Keep updating to get the last one
				}
			}

			script := fmt.Sprintf(`#!/bin/bash
touch "%s"
echo "Processing: 100%%"`, outputFile)

			return exec.CommandContext(ctx, "sh", "-c", script)
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile1, Size: 1024 * 1024},
			{Path: testFile2, Size: 1024 * 1024},
			{Path: testFile3, Size: 1024 * 1024},
		}

		createdFiles, err := par2Executor.CreateInDirectory(ctx, files, outputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		// Verify all PAR2 files were created
		if len(createdFiles) < 3 {
			t.Fatalf("Expected at least 3 PAR2 files, got %d", len(createdFiles))
		}

		// Verify each file has its corresponding PAR2 file in output directory
		expectedFiles := []string{
			filepath.Join(outputDir, "file1.bin.par2"),
			filepath.Join(outputDir, "file2.bin.par2"),
			filepath.Join(outputDir, "file3.bin.par2"),
		}

		for _, expectedFile := range expectedFiles {
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Fatalf("Expected PAR2 file %s was not created", expectedFile)
			}

			found := false
			for _, createdFile := range createdFiles {
				if createdFile == expectedFile {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Expected PAR2 file %s not found in created files list", expectedFile)
			}
		}
	})
}

func TestCheckExistingPar2FilesInDir(t *testing.T) {
	t.Run("detects main PAR2 file and volume files", func(t *testing.T) {
		ctx := context.Background()
		tempDir := t.TempDir()

		// Create test files
		sourceFile := fileinfo.FileInfo{
			Path: filepath.Join(tempDir, "source", "testfile.bin"),
			Size: 1024 * 1024,
		}

		// Create PAR2 files in directory
		mainPar2 := filepath.Join(tempDir, "testfile.bin.par2")
		vol1 := filepath.Join(tempDir, "testfile.bin.vol0+1.par2")
		vol2 := filepath.Join(tempDir, "testfile.bin.vol1+2.par2")
		vol3 := filepath.Join(tempDir, "testfile.bin.vol3+4.par2")

		err := os.WriteFile(mainPar2, []byte("par2 data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create main PAR2 file: %v", err)
		}
		err = os.WriteFile(vol1, []byte("volume 1 data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create volume 1 file: %v", err)
		}
		err = os.WriteFile(vol2, []byte("volume 2 data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create volume 2 file: %v", err)
		}
		err = os.WriteFile(vol3, []byte("volume 3 data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create volume 3 file: %v", err)
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		paths, exists := par2Executor.checkExistingPar2FilesInDir(ctx, sourceFile, tempDir)

		// Verify files were detected
		if !exists {
			t.Fatal("Expected existing PAR2 files to be detected")
		}

		// Verify all files are returned
		if len(paths) != 4 {
			t.Fatalf("Expected 4 PAR2 files (1 main + 3 volumes), got %d", len(paths))
		}

		// Verify specific files are in the list
		foundMain := false
		foundVol1 := false
		foundVol2 := false
		foundVol3 := false

		for _, path := range paths {
			switch path {
			case mainPar2:
				foundMain = true
			case vol1:
				foundVol1 = true
			case vol2:
				foundVol2 = true
			case vol3:
				foundVol3 = true
			}
		}

		if !foundMain {
			t.Error("Main PAR2 file not found in returned paths")
		}
		if !foundVol1 {
			t.Error("Volume 1 file not found in returned paths")
		}
		if !foundVol2 {
			t.Error("Volume 2 file not found in returned paths")
		}
		if !foundVol3 {
			t.Error("Volume 3 file not found in returned paths")
		}
	})

	t.Run("returns false when no PAR2 files exist", func(t *testing.T) {
		ctx := context.Background()
		tempDir := t.TempDir()

		sourceFile := fileinfo.FileInfo{
			Path: filepath.Join(tempDir, "source", "testfile.bin"),
			Size: 1024 * 1024,
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		paths, exists := par2Executor.checkExistingPar2FilesInDir(ctx, sourceFile, tempDir)

		// Verify no files were detected
		if exists {
			t.Fatal("Expected no PAR2 files to be detected in empty directory")
		}

		// Verify empty slice is returned
		if len(paths) != 0 {
			t.Fatalf("Expected empty paths slice, got %d paths", len(paths))
		}
	})

	t.Run("handles directory read errors gracefully", func(t *testing.T) {
		ctx := context.Background()

		// Use a non-existent directory
		nonExistentDir := "/nonexistent/directory/path"

		sourceFile := fileinfo.FileInfo{
			Path: "/tmp/testfile.bin",
			Size: 1024 * 1024,
		}

		par2Executor := &Par2CmdExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Par2Path:   "/usr/bin/par2",
				Redundancy: "10",
			},
			parExeType:  par2,
			jobProgress: newMockJobProgress(),
		}

		paths, exists := par2Executor.checkExistingPar2FilesInDir(ctx, sourceFile, nonExistentDir)

		// Verify function handles error gracefully
		if exists {
			t.Fatal("Expected no PAR2 files to be detected for non-existent directory")
		}

		// Verify empty slice is returned
		if len(paths) != 0 {
			t.Fatalf("Expected empty paths slice, got %d paths", len(paths))
		}
	})
}

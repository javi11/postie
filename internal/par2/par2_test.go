package par2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/javi11/postie/internal/config"
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
		SecondsLeft:    0,
		KBsPerSecond:   0,
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

// createTestFile creates a test file with deterministic data.
func createTestFile(t *testing.T, path string, size int) {
	t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestNew(t *testing.T) {
	articleSize := uint64(512 * 1024)
	cfg := &config.Par2Config{
		Redundancy: "10",
	}

	executor := New(articleSize, cfg, nil)

	if executor == nil {
		t.Fatal("Expected non-nil NativeExecutor")
	}
	if executor.articleSize != articleSize {
		t.Errorf("Expected articleSize %d, got %d", articleSize, executor.articleSize)
	}
	if executor.cfg != cfg {
		t.Error("Expected cfg to match the provided config")
	}
}

func TestNewWithJobProgress(t *testing.T) {
	articleSize := uint64(512 * 1024)
	cfg := &config.Par2Config{
		Redundancy: "10",
	}
	jp := newMockJobProgress()

	executor := New(articleSize, cfg, jp)

	if executor == nil {
		t.Fatal("Expected non-nil NativeExecutor")
	}
	if executor.jobProgress != jp {
		t.Error("Expected jobProgress to match")
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
		{"file.PAR2", true},
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
		{10 * 1024 * 1024, 512 * 1024, 512 * 1024},
		{32768*512*1024 + 1, 512 * 1024, 512 * 1024 * 2},
		{32768*512*1024 - 1, 512 * 1024, 512 * 1024},
		{32768*512*1024*3 + 1, 512 * 1024, 512 * 1024 * 4},
		{1024, 512 * 1024, 512 * 1024},
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

func TestParseRedundancyPercentage(t *testing.T) {
	testCases := []struct {
		name       string
		redundancy string
		expected   float64
	}{
		{"simple number", "10", 10.0},
		{"percentage", "10%", 10.0},
		{"parpar formula 1n*1.2", "1n*1.2", 120.0},
		{"parpar formula 2n*0.5", "2n*0.5", 100.0},
		{"with whitespace", " 15 ", 15.0},
		{"invalid defaults to 10", "invalid", 10.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRedundancyPercentage(tc.redundancy, 1024*1024, 512*1024)
			if result != tc.expected {
				t.Errorf("parseRedundancyPercentage(%q) expected %f, got %f",
					tc.redundancy, tc.expected, result)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Run("creates PAR2 files for a single file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")
		createTestFile(t, testFile, 100000) // 100KB

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 100000},
		}

		createdFiles, err := executor.Create(context.Background(), files)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		// Verify main PAR2 file exists
		expectedPar2File := filepath.Join(tempDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created", expectedPar2File)
		}

		// Verify it's in the returned list
		found := slices.Contains(createdFiles, expectedPar2File)
		if !found {
			t.Fatalf("Main PAR2 file not in returned list: %v", createdFiles)
		}

		// Verify volume files were created
		hasVolume := false
		for _, f := range createdFiles {
			if strings.Contains(f, ".vol") {
				hasVolume = true
				break
			}
		}
		if !hasVolume {
			t.Error("Expected at least one volume file to be created")
		}
	})

	t.Run("skips .par2 files in input", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")
		createTestFile(t, testFile, 50000)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 50000},
			{Path: filepath.Join(tempDir, "existing.par2"), Size: 1000},
		}

		createdFiles, err := executor.Create(context.Background(), files)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Should only create PAR2 for the .bin file, not the .par2 file
		for _, f := range createdFiles {
			if strings.Contains(filepath.Base(f), "existing.par2") && !strings.Contains(f, "testfile") {
				t.Errorf("Should not have created PAR2 for .par2 input file: %s", f)
			}
		}
	})

	t.Run("reuses existing PAR2 files", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")
		createTestFile(t, testFile, 50000)

		// Pre-create PAR2 files
		mainPar2 := filepath.Join(tempDir, "testfile.bin.par2")
		volPar2 := filepath.Join(tempDir, "testfile.bin.vol00+01.par2")
		os.WriteFile(mainPar2, []byte("par2 data"), 0644)
		os.WriteFile(volPar2, []byte("volume data"), 0644)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 50000},
		}

		createdFiles, err := executor.Create(context.Background(), files)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Should return existing files
		if len(createdFiles) < 2 {
			t.Fatalf("Expected at least 2 existing PAR2 files, got %d", len(createdFiles))
		}

		foundMain := false
		foundVol := false
		for _, f := range createdFiles {
			if f == mainPar2 {
				foundMain = true
			}
			if f == volPar2 {
				foundVol = true
			}
		}
		if !foundMain {
			t.Error("Main PAR2 file not found in returned paths")
		}
		if !foundVol {
			t.Error("Volume PAR2 file not found in returned paths")
		}
	})

	t.Run("uses temp directory when configured", func(t *testing.T) {
		sourceDir := t.TempDir()
		tempDir := t.TempDir()
		testFile := filepath.Join(sourceDir, "testfile.bin")
		createTestFile(t, testFile, 100000)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
				TempDir:    tempDir,
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 100000},
		}

		createdFiles, err := executor.Create(context.Background(), files)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Verify files were created in temp directory
		for _, f := range createdFiles {
			if !strings.HasPrefix(f, tempDir) {
				t.Errorf("Expected file in temp dir %s, got %s", tempDir, f)
			}
		}

		// Verify NOT in source directory
		sourcePar2 := filepath.Join(sourceDir, "testfile.bin.par2")
		if _, err := os.Stat(sourcePar2); err == nil {
			t.Error("PAR2 file should not be in source directory")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")
		createTestFile(t, testFile, 100000)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 100000},
		}

		_, err := executor.Create(ctx, files)
		if err == nil {
			t.Error("Expected error from cancelled context")
		}
	})
}

func TestCreateInDirectory(t *testing.T) {
	t.Run("creates PAR2 files in specified output directory", func(t *testing.T) {
		sourceDir := t.TempDir()
		outputDir := t.TempDir()
		testFile := filepath.Join(sourceDir, "testfile.bin")
		createTestFile(t, testFile, 100000)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 100000},
		}

		createdFiles, err := executor.CreateInDirectory(context.Background(), files, outputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		// Verify files are in output directory
		expectedPar2File := filepath.Join(outputDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in output directory", expectedPar2File)
		}

		// Verify NOT in source directory
		sourcePar2File := filepath.Join(sourceDir, "testfile.bin.par2")
		if _, err := os.Stat(sourcePar2File); err == nil {
			t.Fatal("PAR2 file should not be in source directory")
		}
	})

	t.Run("uses default behavior when outputDir is empty", func(t *testing.T) {
		tempDir := t.TempDir()
		configTempDir := filepath.Join(tempDir, "temp")
		os.MkdirAll(configTempDir, 0755)

		testFile := filepath.Join(tempDir, "testfile.bin")
		createTestFile(t, testFile, 100000)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
				TempDir:    configTempDir,
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 100000},
		}

		createdFiles, err := executor.CreateInDirectory(context.Background(), files, "")
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		if len(createdFiles) == 0 {
			t.Fatal("Expected at least one PAR2 file to be created")
		}

		// Should be created in configTempDir
		expectedPar2File := filepath.Join(configTempDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in config temp directory", expectedPar2File)
		}
	})

	t.Run("reuses existing PAR2 files in output directory", func(t *testing.T) {
		sourceDir := t.TempDir()
		outputDir := t.TempDir()
		testFile := filepath.Join(sourceDir, "testfile.bin")
		createTestFile(t, testFile, 50000)

		// Pre-create PAR2 files in output directory
		mainPar2 := filepath.Join(outputDir, "testfile.bin.par2")
		volPar2 := filepath.Join(outputDir, "testfile.bin.vol0+1.par2")
		os.WriteFile(mainPar2, []byte("par2 data"), 0644)
		os.WriteFile(volPar2, []byte("volume data"), 0644)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 50000},
		}

		createdFiles, err := executor.CreateInDirectory(context.Background(), files, outputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		if len(createdFiles) < 2 {
			t.Fatalf("Expected at least 2 existing PAR2 files, got %d", len(createdFiles))
		}

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
			t.Error("Main PAR2 file not found in returned paths")
		}
		if !foundVol {
			t.Error("Volume PAR2 file not found in returned paths")
		}
	})

	t.Run("creates nested output directory structure", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.bin")
		createTestFile(t, testFile, 100000)

		nestedOutputDir := filepath.Join(tempDir, "output", "nested", "deep")

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile, Size: 100000},
		}

		createdFiles, err := executor.CreateInDirectory(context.Background(), files, nestedOutputDir)
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

		expectedPar2File := filepath.Join(nestedOutputDir, "testfile.bin.par2")
		if _, err := os.Stat(expectedPar2File); os.IsNotExist(err) {
			t.Fatalf("Expected PAR2 file %s was not created in nested directory", expectedPar2File)
		}
	})

	t.Run("creates PAR2 files for multiple source files", func(t *testing.T) {
		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		testFile1 := filepath.Join(sourceDir, "file1.bin")
		testFile2 := filepath.Join(sourceDir, "file2.bin")
		testFile3 := filepath.Join(sourceDir, "file3.bin")

		createTestFile(t, testFile1, 50000)
		createTestFile(t, testFile2, 50000)
		createTestFile(t, testFile3, 50000)

		executor := &NativeExecutor{
			articleSize: 10000,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		files := []fileinfo.FileInfo{
			{Path: testFile1, Size: 50000},
			{Path: testFile2, Size: 50000},
			{Path: testFile3, Size: 50000},
		}

		createdFiles, err := executor.CreateInDirectory(context.Background(), files, outputDir)
		if err != nil {
			t.Fatalf("CreateInDirectory failed: %v", err)
		}

		// Verify all PAR2 files were created
		if len(createdFiles) < 3 {
			t.Fatalf("Expected at least 3 PAR2 files (one per source), got %d", len(createdFiles))
		}

		expectedFiles := []string{
			filepath.Join(outputDir, "file1.bin.par2"),
			filepath.Join(outputDir, "file2.bin.par2"),
			filepath.Join(outputDir, "file3.bin.par2"),
		}

		for _, expectedFile := range expectedFiles {
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Fatalf("Expected PAR2 file %s was not created", expectedFile)
			}

			found := slices.Contains(createdFiles, expectedFile)
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

		sourceFile := fileinfo.FileInfo{
			Path: filepath.Join(tempDir, "source", "testfile.bin"),
			Size: 1024 * 1024,
		}

		// Create PAR2 files in directory
		mainPar2 := filepath.Join(tempDir, "testfile.bin.par2")
		vol1 := filepath.Join(tempDir, "testfile.bin.vol0+1.par2")
		vol2 := filepath.Join(tempDir, "testfile.bin.vol1+2.par2")
		vol3 := filepath.Join(tempDir, "testfile.bin.vol3+4.par2")

		os.WriteFile(mainPar2, []byte("par2 data"), 0644)
		os.WriteFile(vol1, []byte("volume 1 data"), 0644)
		os.WriteFile(vol2, []byte("volume 2 data"), 0644)
		os.WriteFile(vol3, []byte("volume 3 data"), 0644)

		executor := &NativeExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		paths, exists := executor.checkExistingPar2FilesInDir(ctx, sourceFile, tempDir)

		if !exists {
			t.Fatal("Expected existing PAR2 files to be detected")
		}

		if len(paths) != 4 {
			t.Fatalf("Expected 4 PAR2 files (1 main + 3 volumes), got %d", len(paths))
		}

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

		executor := &NativeExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		paths, exists := executor.checkExistingPar2FilesInDir(ctx, sourceFile, tempDir)

		if exists {
			t.Fatal("Expected no PAR2 files to be detected in empty directory")
		}
		if len(paths) != 0 {
			t.Fatalf("Expected empty paths slice, got %d paths", len(paths))
		}
	})

	t.Run("handles directory read errors gracefully", func(t *testing.T) {
		ctx := context.Background()
		nonExistentDir := "/nonexistent/directory/path"

		sourceFile := fileinfo.FileInfo{
			Path: "/tmp/testfile.bin",
			Size: 1024 * 1024,
		}

		executor := &NativeExecutor{
			articleSize: 512 * 1024,
			cfg: &config.Par2Config{
				Redundancy: "10",
			},
			jobProgress: newMockJobProgress(),
		}

		paths, exists := executor.checkExistingPar2FilesInDir(ctx, sourceFile, nonExistentDir)

		if exists {
			t.Fatal("Expected no PAR2 files to be detected for non-existent directory")
		}
		if len(paths) != 0 {
			t.Fatalf("Expected empty paths slice, got %d paths", len(paths))
		}
	})
}

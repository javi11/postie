package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestIgnorePatternFlag(t *testing.T) {
	flag := rootCmd.Flags().Lookup("ignore-pattern")
	if flag == nil {
		t.Fatal("missing --ignore-pattern flag")
	}
	previous := append([]string(nil), ignorePatterns...)
	changed := flag.Changed
	t.Cleanup(func() {
		if err := flag.Value.(interface{ Replace([]string) error }).Replace(previous); err != nil {
			t.Errorf("restore ignore patterns: %v", err)
		}
		flag.Changed = changed
	})
	if err := rootCmd.ParseFlags([]string{"--ignore-pattern", "*.tmp", "--ignore-pattern", "*.part", "--ignore-pattern", "*.!ut"}); err != nil {
		t.Fatalf("parse repeated ignore patterns: %v", err)
	}
	patterns, err := rootCmd.Flags().GetStringArray("ignore-pattern")
	if err != nil {
		t.Fatalf("get ignore patterns: %v", err)
	}
	want := []string{"*.tmp", "*.part", "*.!ut"}
	if !reflect.DeepEqual(patterns, want) {
		t.Errorf("ignore patterns = %v, want %v", patterns, want)
	}
}

func TestUploadFileSelectionBeforeConfiguration(t *testing.T) {
	previousConfig, previousDir, previousInput, previousPatterns := configPath, dirPath, inputFile, ignorePatterns
	t.Cleanup(func() {
		configPath, dirPath, inputFile, ignorePatterns = previousConfig, previousDir, previousInput, previousPatterns
	})
	dir := t.TempDir()
	file := filepath.Join(dir, "download.tmp")
	if err := os.WriteFile(file, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath = filepath.Join(dir, "missing-config.yaml")
	dirPath = dir
	for _, tc := range []struct {
		name     string
		input    string
		patterns []string
		wantErr  string
	}{
		{"all files ignored", "", []string{"*.tmp"}, "no files to post"},
		{"invalid pattern", "", []string{"["}, "invalid ignore pattern"},
		{"explicit input bypasses patterns", file, []string{"*.tmp", "["}, "missing-config.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inputFile, ignorePatterns = tc.input, tc.patterns
			err := rootCmd.RunE(rootCmd, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("upload error = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestCollectDirectoryFiles(t *testing.T) {
	dir := t.TempDir()
	names := []string{"keep.txt", "download.tmp", "download.part", "download.!ut", "nested/keep.txt", "nested/download.tmp", "folder.tmp/keep.txt", "file,with,commas.txt"}
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name     string
		patterns []string
		want     []string
	}{
		{"no patterns", nil, []string{"download.!ut", "download.part", "download.tmp", "file,with,commas.txt", "folder.tmp/keep.txt", "keep.txt", "nested/download.tmp", "nested/keep.txt"}},
		{"temporary files", []string{"*.tmp", "*.part", "*.!ut"}, []string{"file,with,commas.txt", "folder.tmp/keep.txt", "keep.txt", "nested/keep.txt"}},
		{"all files ignored", []string{"*"}, []string{}},
		{"literal commas", []string{"file,with,commas.txt"}, []string{"download.!ut", "download.part", "download.tmp", "folder.tmp/keep.txt", "keep.txt", "nested/download.tmp", "nested/keep.txt"}},
		{"stars inside character class", []string{"[*[]"}, []string{"download.!ut", "download.part", "download.tmp", "file,with,commas.txt", "folder.tmp/keep.txt", "keep.txt", "nested/download.tmp", "nested/keep.txt"}},
		{"escaped star", []string{`\*`}, []string{"download.!ut", "download.part", "download.tmp", "file,with,commas.txt", "folder.tmp/keep.txt", "keep.txt", "nested/download.tmp", "nested/keep.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files, err := collectDirectoryFiles(dir, tc.patterns)
			if err != nil {
				t.Fatalf("collect directory files: %v", err)
			}
			got := make([]string, 0, len(files))
			for _, file := range files {
				rel, err := filepath.Rel(dir, file.Path)
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, filepath.ToSlash(rel))
				if file.Size != uint64(len("content")) {
					t.Errorf("%s size = %d, want %d", rel, file.Size, len("content"))
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("collected files = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCollectDirectoryFilesErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		dir      string
		patterns []string
	}{
		{"missing directory", filepath.Join(t.TempDir(), "missing"), nil},
		{"invalid pattern", t.TempDir(), []string{"["}},
		{"invalid pattern after wildcard", t.TempDir(), []string{"download*["}},
		{"invalid pattern after multiple wildcards", t.TempDir(), []string{"download*part*["}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := collectDirectoryFiles(tc.dir, tc.patterns); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

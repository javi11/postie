package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kipsilabs/postie/internal/config"
)

func configLocationApp(t *testing.T) *App {
	t.Helper()
	source := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(source, []byte("# preserve comments and pending settings\noutput_dir: saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return &App{configPath: source, appPaths: &AppPaths{Config: source, Database: "original.db", Data: "original-data", Log: "original.log"}}
}

func TestRelocateConfigPersistsAndRoutesSubsequentSaves(t *testing.T) {
	app := configLocationApp(t)
	original := app.GetConfigPath()
	data, _ := os.ReadFile(original)
	dir := t.TempDir()
	path, err := app.relocateConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "config.yaml") || app.GetConfigPath() != path {
		t.Fatalf("active path = %q, returned %q", app.GetConfigPath(), path)
	}
	copied, _ := os.ReadFile(path)
	if string(copied) != string(data) {
		t.Fatalf("saved bytes changed: %q", copied)
	}
	resolved, err := resolveConfigPath(original)
	if err != nil || resolved != path {
		t.Fatalf("restart path = %q, error %v", resolved, err)
	}
	restarted := &App{configPath: resolved, appPaths: app.appPaths}
	cfg := config.GetDefaultConfig()
	cfg.OutputDir = "new-output"
	if err := restarted.saveConfigFile(&cfg); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(original)
	if string(after) != string(data) {
		t.Fatal("original backup overwritten by subsequent save")
	}
	saved, _ := os.ReadFile(path)
	if string(saved) == string(data) {
		t.Fatal("new config location not updated")
	}
	if app.appPaths.Config != original || app.appPaths.Database != "original.db" {
		t.Fatal("other application paths changed")
	}
}

func TestRelocateConfigFailurePreservesFilesAndPath(t *testing.T) {
	for _, scenario := range []string{"existing destination", "invalid directory", "preference write failure", "missing source", "web mode"} {
		t.Run(scenario, func(t *testing.T) {
			app := configLocationApp(t)
			original := app.GetConfigPath()
			dir := t.TempDir()
			destination := filepath.Join(dir, "config.yaml")
			switch scenario {
			case "existing destination":
				if err := os.WriteFile(destination, []byte("existing config"), 0600); err != nil {
					t.Fatal(err)
				}
			case "invalid directory":
				dir = filepath.Join(dir, "missing")
			case "preference write failure":
				if err := os.Mkdir(configLocationFile(original), 0700); err != nil {
					t.Fatal(err)
				}
			case "missing source":
				if err := os.Remove(original); err != nil {
					t.Fatal(err)
				}
			case "web mode":
				app.isWebMode = true
			}
			if _, err := app.relocateConfig(dir); err == nil {
				t.Fatal("expected relocation to fail")
			}
			if app.GetConfigPath() != original {
				t.Fatal("failed relocation changed active path")
			}
			if scenario == "existing destination" {
				bytes, _ := os.ReadFile(destination)
				if string(bytes) != "existing config" {
					t.Fatal("existing destination overwritten")
				}
			} else if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("failed relocation left destination: %v", err)
			}
			if scenario != "missing source" {
				if _, err := os.Stat(original); err != nil {
					t.Fatalf("original lost: %v", err)
				}
			}
		})
	}
}

func TestConfigLocationDefaultAndSameDirectory(t *testing.T) {
	app := configLocationApp(t)
	original := app.GetConfigPath()
	path, err := resolveConfigPath(original)
	if err != nil || path != original {
		t.Fatalf("default path = %q, %v", path, err)
	}
	path, err = app.relocateConfig(filepath.Dir(original))
	if err != nil || path != original {
		t.Fatalf("same directory = %q, %v", path, err)
	}
	if _, err := os.Stat(configLocationFile(original)); !os.IsNotExist(err) {
		t.Fatal("same directory wrote preference")
	}
}

func TestStartupPreservesUnavailableCustomConfig(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint("missing=", missing), func(t *testing.T) {
			app := configLocationApp(t)
			original := app.GetConfigPath()
			path := filepath.Join(t.TempDir(), "config.yaml")
			invalid := []byte("version: 99999\n")
			if !missing {
				if err := os.WriteFile(path, invalid, 0600); err != nil {
					t.Fatal(err)
				}
			}
			app.configPath = path
			app.isWebMode = true // Avoid Wails runtime calls in this startup integration test.
			app.Startup(context.Background())
			defer app.Shutdown()
			if missing {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("startup created config at unavailable custom location")
				}
			} else {
				data, _ := os.ReadFile(path)
				if string(data) != string(invalid) {
					t.Fatal("startup overwrote custom configuration")
				}
			}
			status := app.GetAppStatus()
			if !status.CriticalConfigError || status.Error == "" {
				t.Fatalf("startup error not exposed: %+v", status)
			}
			if app.GetConfigPath() != path || app.appPaths.Config != original {
				t.Fatal("startup fell back to stale backup")
			}
		})
	}
}

func isolatedAppPaths(t *testing.T) *AppPaths {
	t.Helper()
	root := t.TempDir()
	for _, key := range []string{"HOME", "APPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME"} {
		t.Setenv(key, root)
	}
	paths, err := GetAppPaths()
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

// Use the real constructor and Startup without opening a process-wide log
// sink. Every app created here owns cleanup before its TempDir is removed.
func newConfigLocationTestApp(t *testing.T) *App {
	t.Helper()
	app := newApp(func(path string) (string, error) { return path, nil })
	t.Cleanup(func() {
		app.Shutdown()
		if app.procCancel != nil {
			app.procCancel()
		}
		if app.queue != nil {
			if err := app.queue.Close(); err != nil {
				t.Errorf("close test queue: %v", err)
			}
		}
		if app.database != nil {
			if err := app.database.Close(); err != nil {
				t.Errorf("close test database: %v", err)
			}
		}
	})
	return app
}

func TestNewAppReportsUnreadableLocationPreference(t *testing.T) {
	paths := isolatedAppPaths(t)
	if err := os.WriteFile(paths.Config, []byte("version: 99999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configLocationFile(paths.Config), []byte("relative/path.yaml"), 0600); err != nil {
		t.Fatal(err)
	}
	app := newConfigLocationTestApp(t)
	app.isWebMode = true
	app.Startup(context.Background())
	status := app.GetAppStatus()
	if !status.CriticalConfigError || status.Error == "" {
		t.Fatalf("invalid location was silently ignored: %+v", status)
	}
	data, _ := os.ReadFile(paths.Config)
	if string(data) != "version: 99999\n" {
		t.Fatal("startup overwrote backup after location resolution failed")
	}
}

func TestNewAppLoadsRelocatedConfiguration(t *testing.T) {
	paths := isolatedAppPaths(t)
	cfg := config.GetDefaultConfig()
	cfg.Version = config.CurrentConfigVersion
	cfg.OutputDir = "saved-output"
	enabled := true
	cfg.Servers = []config.ServerConfig{{Host: "news.example.com", Port: 563, SSL: true, MaxConnections: 1, Enabled: &enabled, Role: config.ServerRoleUpload}}
	cfg.Database.DatabasePath = paths.Database
	if err := config.SaveConfig(&cfg, paths.Config); err != nil {
		t.Fatal(err)
	}
	app := newConfigLocationTestApp(t)
	path, err := app.relocateConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Distinguish the relocated file from the retained backup at startup.
	cfg.OutputDir = "relocated-output"
	if err := config.SaveConfig(&cfg, path); err != nil {
		t.Fatal(err)
	}
	restarted := newConfigLocationTestApp(t)
	restarted.isWebMode = true // The backend startup is identical; avoid emitting Wails events in a test.
	restarted.Startup(context.Background())
	loaded, err := restarted.GetConfig()
	if err != nil || loaded.OutputDir != "relocated-output" {
		t.Fatalf("restart did not load relocated YAML: config=%+v, err=%v", loaded, err)
	}
	if restarted.GetConfigPath() != path {
		t.Fatalf("restart path = %q", restarted.GetConfigPath())
	}
	// Web startup retains its previous default path behavior.
	web := newConfigLocationTestApp(t)
	web.SetWebMode(true)
	if web.GetConfigPath() != paths.Config {
		t.Fatal("desktop preference changed web configuration path")
	}
}

func TestCustomConfigurationErrorClearsAfterRestore(t *testing.T) {
	app := configLocationApp(t)
	app.configPath = filepath.Join(t.TempDir(), "config.yaml")
	app.isWebMode = true
	app.Startup(context.Background())
	defer app.Shutdown()
	if !app.GetAppStatus().CriticalConfigError {
		t.Fatal("missing custom config was not reported")
	}
	cfg := config.GetDefaultConfig()
	cfg.Version = config.CurrentConfigVersion
	enabled := true
	cfg.Servers = []config.ServerConfig{{Host: "news.example.com", Port: 563, SSL: true, MaxConnections: 1, Enabled: &enabled, Role: config.ServerRoleUpload}}
	if err := config.SaveConfig(&cfg, app.GetConfigPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GetConfig(); err != nil {
		t.Fatal(err)
	}
	if status := app.GetAppStatus(); status.CriticalConfigError || status.Error != "" {
		t.Fatalf("restored config still blocked: %+v", status)
	}
	app.criticalErrorMessage = "unrelated processor error"
	if _, err := app.GetConfig(); err != nil {
		t.Fatal(err)
	}
	if app.GetAppStatus().Error != "unrelated processor error" {
		t.Fatal("config recovery cleared unrelated error")
	}
}

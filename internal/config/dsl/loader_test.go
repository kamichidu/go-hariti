package dsl_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamichidu/go-hariti/internal/config/dsl"
)

func TestLoader_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create main file
	mainContent := `
include "plugins/core.hariti"
use main-plugin
`
	mainPath := filepath.Join(tmpDir, "main.hariti")
	if err := ioutil.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.hariti: %v", err)
	}

	// 2. Create plugins dir and core.hariti
	pluginsDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	coreContent := `
include '../common/helper.hariti'
use core-plugin
`
	if err := ioutil.WriteFile(filepath.Join(pluginsDir, "core.hariti"), []byte(coreContent), 0644); err != nil {
		t.Fatalf("failed to write core.hariti: %v", err)
	}

	// 3. Create common dir and helper.hariti
	commonDir := filepath.Join(tmpDir, "common")
	if err := os.MkdirAll(commonDir, 0755); err != nil {
		t.Fatalf("failed to create common dir: %v", err)
	}

	helperContent := `use helper-plugin`
	if err := ioutil.WriteFile(filepath.Join(commonDir, "helper.hariti"), []byte(helperContent), 0644); err != nil {
		t.Fatalf("failed to write helper.hariti: %v", err)
	}

	// Load main file
	g, err := dsl.LoadGraph(mainPath)
	if err != nil {
		t.Fatalf("failed to load graph: %v", err)
	}

	// We expect 3 bundles: main-plugin, core-plugin, and helper-plugin (merged in the correct compilation unit order)
	if len(g.Bundles) != 3 {
		t.Errorf("expected 3 bundles, got %d", len(g.Bundles))
	}

	expectedIDs := map[string]bool{
		"main-plugin":   true,
		"core-plugin":   true,
		"helper-plugin": true,
	}

	for _, b := range g.Bundles {
		if !expectedIDs[b.ID] {
			t.Errorf("unexpected bundle ID: %s", b.ID)
		}
	}
}

func TestLoader_CircularDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// file A includes file B
	// file B includes file A
	aPath := filepath.Join(tmpDir, "a.hariti")
	bPath := filepath.Join(tmpDir, "b.hariti")

	aContent := `include "b.hariti"`
	bContent := `include "a.hariti"`

	if err := ioutil.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatalf("failed to write a.hariti: %v", err)
	}
	if err := ioutil.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("failed to write b.hariti: %v", err)
	}

	_, err := dsl.LoadGraph(aPath)
	if err == nil {
		t.Error("expected circular dependency error, but got nil")
	} else if !strings.Contains(err.Error(), "circular include detected") {
		t.Errorf("expected 'circular include detected' error, got: %v", err)
	}
}

func TestLoader_Glob(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create main file with glob pattern include
	mainContent := `
include "plugins/*.hariti"
use main-plugin
`
	mainPath := filepath.Join(tmpDir, "main.hariti")
	if err := ioutil.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.hariti: %v", err)
	}

	// 2. Create plugins dir and multiple hariti files
	pluginsDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	if err := ioutil.WriteFile(filepath.Join(pluginsDir, "a.hariti"), []byte(`use a-plugin`), 0644); err != nil {
		t.Fatalf("failed to write a.hariti: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(pluginsDir, "b.hariti"), []byte(`use b-plugin`), 0644); err != nil {
		t.Fatalf("failed to write b.hariti: %v", err)
	}

	// Load main file
	g, err := dsl.LoadGraph(mainPath)
	if err != nil {
		t.Fatalf("failed to load graph: %v", err)
	}

	// We expect 3 bundles: main-plugin, a-plugin, b-plugin
	if len(g.Bundles) != 3 {
		t.Errorf("expected 3 bundles, got %d", len(g.Bundles))
	}

	expectedIDs := map[string]bool{
		"main-plugin": true,
		"a-plugin":    true,
		"b-plugin":    true,
	}

	for _, b := range g.Bundles {
		if !expectedIDs[b.ID] {
			t.Errorf("unexpected bundle ID: %s", b.ID)
		}
	}
}

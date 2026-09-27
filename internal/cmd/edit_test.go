package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunEditRequiresPRDExists(t *testing.T) {
	tmpDir := t.TempDir()

	opts := EditOptions{
		Name:    "nonexistent",
		BaseDir: tmpDir,
	}

	err := RunEdit(opts)
	if err == nil {
		t.Error("Expected error for non-existent PRD")
	}

	// Error message should suggest using chief new
	if err != nil {
		errStr := err.Error()
		if !contains(errStr, "chief new") {
			t.Errorf("Error should suggest chief new, got: %s", errStr)
		}
	}
}

func TestRunEditRejectsInvalidName(t *testing.T) {
	tmpDir := t.TempDir()

	opts := EditOptions{
		Name:    "invalid name with space",
		BaseDir: tmpDir,
	}

	err := RunEdit(opts)
	if err == nil {
		t.Error("Expected error for invalid name")
	}
}

func TestRunEditDefaultsToLegacyMainForBackwardCompat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main prd.md (the pre-rename default location)
	prdDir := filepath.Join(tmpDir, ".chief", "prds", "main")
	if err := os.MkdirAll(prdDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	prdMdPath := filepath.Join(prdDir, "prd.md")
	if err := os.WriteFile(prdMdPath, []byte("# Main PRD"), 0644); err != nil {
		t.Fatalf("Failed to create prd.md: %v", err)
	}

	// Existing projects created before the "default" rename must keep
	// resolving to their "main" PRD when no name is given.
	resolved := ResolveDefaultPRDName(tmpDir)
	if resolved != "main" {
		t.Errorf("Expected empty name to resolve to legacy 'main', got %q", resolved)
	}

	prdPath := filepath.Join(tmpDir, ".chief", "prds", resolved, "prd.md")
	if _, err := os.Stat(prdPath); os.IsNotExist(err) {
		t.Error("Expected default name to resolve to existing prd.md")
	}
}

func TestRunEditDefaultsToDefaultForNewProjects(t *testing.T) {
	tmpDir := t.TempDir()

	resolved := ResolveDefaultPRDName(tmpDir)
	if resolved != "default" {
		t.Errorf("Expected empty name to resolve to 'default' when no PRD exists, got %q", resolved)
	}
}

func TestEditOptionsDefaults(t *testing.T) {
	opts := EditOptions{}

	if opts.Name != "" {
		t.Error("Name should default to empty (filled later)")
	}
	if opts.BaseDir != "" {
		t.Error("BaseDir should default to empty (filled later)")
	}
}

func TestRunEditRequiresProvider(t *testing.T) {
	tmpDir := t.TempDir()
	prdDir := filepath.Join(tmpDir, ".chief", "prds", "main")
	if err := os.MkdirAll(prdDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	prdMdPath := filepath.Join(prdDir, "prd.md")
	if err := os.WriteFile(prdMdPath, []byte("# Main PRD"), 0644); err != nil {
		t.Fatalf("Failed to create prd.md: %v", err)
	}

	opts := EditOptions{
		Name:    "main",
		BaseDir: tmpDir,
	}

	err := RunEdit(opts)
	if err == nil {
		t.Fatal("expected provider validation error")
	}
	if !contains(err.Error(), "Provider") {
		t.Fatalf("expected error to mention Provider, got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

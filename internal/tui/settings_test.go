package tui

import (
	"strings"
	"testing"

	"github.com/minicodemonkey/chief/internal/config"
)

func TestSettingsOverlay_LoadFromConfig(t *testing.T) {
	s := NewSettingsOverlay()
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			Setup:               "npm install",
			AlwaysPrompt:        false,
			PromptBranchPattern: "^(main|master)$",
		},
		OnComplete: config.OnCompleteConfig{
			Push:     true,
			CreatePR: false,
		},
	}
	s.LoadFromConfig(cfg)

	// Order matches the YAML in docs/reference/configuration.md so the TUI
	// reads top-to-bottom in the same order users see in their config file.
	wantKeys := []string{
		"agent.watchdogTimeout",
		"worktree.setup",
		"worktree.alwaysPrompt",
		"worktree.promptBranchPattern",
		"bash.timeout",
		"onComplete.push",
		"onComplete.createPR",
	}
	if len(s.items) != len(wantKeys) {
		t.Fatalf("expected %d items, got %d", len(wantKeys), len(s.items))
	}
	for i, key := range wantKeys {
		if s.items[i].Key != key {
			t.Errorf("item %d: expected key=%s, got %s", i, key, s.items[i].Key)
		}
	}
	if s.items[1].StringVal != "npm install" {
		t.Errorf("worktree.setup: got val=%s", s.items[1].StringVal)
	}
	if s.items[2].BoolVal {
		t.Error("worktree.alwaysPrompt: expected false")
	}
	if s.items[3].StringVal != "^(main|master)$" {
		t.Errorf("worktree.promptBranchPattern: got val=%s", s.items[3].StringVal)
	}
	if !s.items[5].BoolVal {
		t.Error("onComplete.push: expected true")
	}
	if s.items[6].BoolVal {
		t.Error("onComplete.createPR: expected false")
	}
	if s.selectedIndex != 0 {
		t.Errorf("expected selectedIndex=0, got %d", s.selectedIndex)
	}
}

func TestSettingsOverlay_ApplyToConfig(t *testing.T) {
	s := NewSettingsOverlay()
	cfg := config.Default()
	s.LoadFromConfig(cfg)

	s.items[0].StringVal = "20m"             // agent.watchdogTimeout
	s.items[1].StringVal = "go mod download" // worktree.setup
	s.items[2].BoolVal = true                // worktree.alwaysPrompt
	s.items[3].StringVal = "^release/.*$"    // worktree.promptBranchPattern
	s.items[4].StringVal = "30s"             // bash.timeout
	s.items[5].BoolVal = true                // onComplete.push
	s.items[6].BoolVal = true                // onComplete.createPR

	resultCfg := config.Default()
	s.ApplyToConfig(resultCfg)

	if resultCfg.Agent.WatchdogTimeout != "20m" {
		t.Errorf("expected agent.watchdogTimeout='20m', got '%s'", resultCfg.Agent.WatchdogTimeout)
	}
	if resultCfg.Worktree.Setup != "go mod download" {
		t.Errorf("expected setup='go mod download', got '%s'", resultCfg.Worktree.Setup)
	}
	if resultCfg.Bash.Timeout != "30s" {
		t.Errorf("expected bash.timeout='30s', got '%s'", resultCfg.Bash.Timeout)
	}
	if !resultCfg.Worktree.AlwaysPrompt {
		t.Error("expected alwaysPrompt=true")
	}
	if resultCfg.Worktree.PromptBranchPattern != "^release/.*$" {
		t.Errorf("expected promptBranchPattern='^release/.*$', got '%s'", resultCfg.Worktree.PromptBranchPattern)
	}
	if !resultCfg.OnComplete.Push {
		t.Error("expected push=true")
	}
	if !resultCfg.OnComplete.CreatePR {
		t.Error("expected createPR=true")
	}
}

func TestSettingsOverlay_Navigation(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	if s.selectedIndex != 0 {
		t.Fatalf("expected initial index=0, got %d", s.selectedIndex)
	}

	last := len(s.items) - 1
	for i := 1; i <= last; i++ {
		s.MoveDown()
		if s.selectedIndex != i {
			t.Errorf("expected index=%d after MoveDown, got %d", i, s.selectedIndex)
		}
	}

	// Can't go beyond last item
	s.MoveDown()
	if s.selectedIndex != last {
		t.Errorf("expected index=%d (clamped), got %d", last, s.selectedIndex)
	}

	s.MoveUp()
	if s.selectedIndex != last-1 {
		t.Errorf("expected index=%d after MoveUp, got %d", last-1, s.selectedIndex)
	}

	// Can't go before first item
	for i := 0; i < 10; i++ {
		s.MoveUp()
	}
	if s.selectedIndex != 0 {
		t.Errorf("expected index=0 (clamped), got %d", s.selectedIndex)
	}
}

// selectKey moves the selection to the item with the given key and returns
// its index.
func selectKey(t *testing.T, s *SettingsOverlay, key string) int {
	t.Helper()
	for i, item := range s.items {
		if item.Key == key {
			s.selectedIndex = i
			return i
		}
	}
	t.Fatalf("no settings item with key %q", key)
	return -1
}

func TestSettingsOverlay_ToggleBool(t *testing.T) {
	s := NewSettingsOverlay()
	cfg := &config.Config{
		OnComplete: config.OnCompleteConfig{Push: false},
	}
	s.LoadFromConfig(cfg)

	selectKey(t, s, "onComplete.push")

	key, val := s.ToggleBool()
	if key != "onComplete.push" {
		t.Errorf("expected key='onComplete.push', got '%s'", key)
	}
	if !val {
		t.Error("expected val=true after toggle")
	}

	// Toggle back
	_, val = s.ToggleBool()
	if val {
		t.Error("expected val=false after second toggle")
	}
}

func TestSettingsOverlay_ToggleBool_OnStringItem(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	// Selected item is "Watchdog timeout" (string type, index 0)
	key, _ := s.ToggleBool()
	if key != "" {
		t.Errorf("expected empty key for string item toggle, got '%s'", key)
	}
}

func TestSettingsOverlay_RevertToggle(t *testing.T) {
	s := NewSettingsOverlay()
	cfg := &config.Config{
		OnComplete: config.OnCompleteConfig{Push: false},
	}
	s.LoadFromConfig(cfg)

	push := selectKey(t, s, "onComplete.push")
	s.ToggleBool()
	if !s.items[push].BoolVal {
		t.Fatal("expected true after toggle")
	}

	s.RevertToggle()
	if s.items[push].BoolVal {
		t.Error("expected false after revert")
	}
}

func TestSettingsOverlay_BashTimeoutValidation(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	selectKey(t, s, "bash.timeout")
	if s.GetSelectedItem().Key != "bash.timeout" {
		t.Fatalf("setup error: expected bash.timeout selected, got %q", s.GetSelectedItem().Key)
	}

	// Invalid duration: edit should be rejected, edit mode preserved.
	s.StartEditing()
	for _, ch := range "5minutes" {
		s.AddEditChar(ch)
	}
	s.ConfirmEdit()
	if !s.IsEditing() {
		t.Fatal("expected to remain in edit mode for invalid duration")
	}
	if s.editError == "" {
		t.Error("expected editError to be set for invalid duration")
	}
	if s.GetSelectedItem().StringVal != "" {
		t.Errorf("expected stored value to remain unchanged, got %q", s.GetSelectedItem().StringVal)
	}

	// Correct the buffer to a valid value: edit accepted, error cleared,
	// surrounding whitespace trimmed.
	s.editBuffer = "  30s  "
	s.ConfirmEdit()
	if s.IsEditing() {
		t.Error("expected to exit edit mode after valid duration")
	}
	if s.editError != "" {
		t.Errorf("expected editError cleared, got %q", s.editError)
	}
	if s.GetSelectedItem().StringVal != "30s" {
		t.Errorf("expected stored value '30s', got %q", s.GetSelectedItem().StringVal)
	}
}

func TestSettingsOverlay_BashTimeoutEmptyAccepted(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(&config.Config{Bash: config.BashConfig{Timeout: "5m"}})
	selectKey(t, s, "bash.timeout")
	if s.GetSelectedItem().Key != "bash.timeout" {
		t.Fatalf("setup error: expected bash.timeout selected, got %q", s.GetSelectedItem().Key)
	}

	s.StartEditing()
	// Clear the buffer entirely. An empty value is accepted; at runtime
	// BashTimeout returns 0 (no timeout) for empty.
	s.editBuffer = ""
	s.ConfirmEdit()
	if s.IsEditing() {
		t.Error("expected empty value to be accepted")
	}
	if s.GetSelectedItem().StringVal != "" {
		t.Errorf("expected stored value '', got %q", s.GetSelectedItem().StringVal)
	}
}

func TestSettingsOverlay_BashTimeoutNegativeRejected(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	selectKey(t, s, "bash.timeout")
	s.StartEditing()
	for _, ch := range "-10s" {
		s.AddEditChar(ch)
	}
	s.ConfirmEdit()
	if !s.IsEditing() {
		t.Error("expected negative duration to be rejected")
	}
	if s.editError == "" {
		t.Error("expected editError set for negative duration")
	}
}

func TestSettingsOverlay_AgentWatchdogTimeoutValidation(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	// agent.watchdogTimeout is the first item — no navigation needed.
	if s.GetSelectedItem().Key != "agent.watchdogTimeout" {
		t.Fatalf("setup error: expected agent.watchdogTimeout selected, got %q", s.GetSelectedItem().Key)
	}

	// Invalid duration: rejected, edit mode preserved.
	s.StartEditing()
	for _, ch := range "10minutes" {
		s.AddEditChar(ch)
	}
	s.ConfirmEdit()
	if !s.IsEditing() {
		t.Fatal("expected to remain in edit mode for invalid duration")
	}
	if s.editError == "" {
		t.Error("expected editError to be set for invalid duration")
	}
	if s.GetSelectedItem().StringVal != "" {
		t.Errorf("expected stored value to remain unchanged, got %q", s.GetSelectedItem().StringVal)
	}

	// Valid value with surrounding whitespace is trimmed and accepted.
	s.editBuffer = "  20m  "
	s.ConfirmEdit()
	if s.IsEditing() {
		t.Error("expected valid duration to be accepted")
	}
	if s.GetSelectedItem().StringVal != "20m" {
		t.Errorf("expected stored value '20m', got %q", s.GetSelectedItem().StringVal)
	}
}

func TestSettingsOverlay_AgentWatchdogTimeoutNegativeRejected(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	if s.GetSelectedItem().Key != "agent.watchdogTimeout" {
		t.Fatalf("setup error: expected agent.watchdogTimeout selected, got %q", s.GetSelectedItem().Key)
	}
	s.StartEditing()
	for _, ch := range "-5m" {
		s.AddEditChar(ch)
	}
	s.ConfirmEdit()
	if !s.IsEditing() {
		t.Error("expected negative duration to be rejected")
	}
	if s.editError == "" {
		t.Error("expected editError set for negative duration")
	}
}

func TestSettingsOverlay_StringEditing(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	s.MoveDown() // Select "Setup command" (worktree.setup, index 1) — duration
	// validation does not apply here, so an arbitrary value is accepted.
	if s.IsEditing() {
		t.Fatal("should not be editing initially")
	}

	s.StartEditing()
	if !s.IsEditing() {
		t.Fatal("should be editing after StartEditing")
	}
	if s.editBuffer != "" {
		t.Errorf("expected empty edit buffer, got '%s'", s.editBuffer)
	}

	s.AddEditChar('n')
	s.AddEditChar('p')
	s.AddEditChar('m')
	if s.editBuffer != "npm" {
		t.Errorf("expected 'npm', got '%s'", s.editBuffer)
	}

	s.DeleteEditChar()
	if s.editBuffer != "np" {
		t.Errorf("expected 'np' after delete, got '%s'", s.editBuffer)
	}

	if err := s.ConfirmEdit(); err != nil {
		t.Fatalf("ConfirmEdit unexpected error: %v", err)
	}
	if s.IsEditing() {
		t.Fatal("should not be editing after ConfirmEdit")
	}
	if s.items[1].StringVal != "np" {
		t.Errorf("expected StringVal='np', got '%s'", s.items[1].StringVal)
	}
}

func TestSettingsOverlay_CancelEdit(t *testing.T) {
	s := NewSettingsOverlay()
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{Setup: "original"},
	}
	s.LoadFromConfig(cfg)
	s.MoveDown() // worktree.setup (index 1)

	s.StartEditing()
	s.AddEditChar('x')
	s.CancelEdit()

	if s.IsEditing() {
		t.Fatal("should not be editing after CancelEdit")
	}
	if s.items[1].StringVal != "original" {
		t.Errorf("expected 'original' preserved, got '%s'", s.items[1].StringVal)
	}
}

func TestSettingsOverlay_StartEditingOnBoolItem(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	selectKey(t, s, "worktree.alwaysPrompt")

	s.StartEditing()
	if s.IsEditing() {
		t.Error("should not start editing on a bool item")
	}
}

func TestSettingsOverlay_ConfirmEdit_InvalidRegex(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	// Navigate to "Prompt branch pattern"
	pattern := selectKey(t, s, "worktree.promptBranchPattern")

	original := s.items[pattern].StringVal
	s.StartEditing()
	for s.editBuffer != "" {
		s.DeleteEditChar()
	}
	for _, ch := range "[bad" {
		s.AddEditChar(ch)
	}

	err := s.ConfirmEdit()
	if err == nil {
		t.Fatal("expected error from ConfirmEdit on invalid regex, got nil")
	}
	if !s.IsEditing() {
		t.Error("expected to remain in edit mode after rejection")
	}
	if s.items[pattern].StringVal != original {
		t.Errorf("expected item value unchanged on rejection, got %q (was %q)", s.items[pattern].StringVal, original)
	}
	if !strings.Contains(s.editError, "invalid regex") {
		t.Errorf("expected editError to mention invalid regex, got %q", s.editError)
	}
}

func TestSettingsOverlay_ConfirmEdit_ValidRegex(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	// Navigate to "Prompt branch pattern"
	pattern := selectKey(t, s, "worktree.promptBranchPattern")
	s.StartEditing()
	for s.editBuffer != "" {
		s.DeleteEditChar()
	}
	for _, ch := range "^release/.*$" {
		s.AddEditChar(ch)
	}

	if err := s.ConfirmEdit(); err != nil {
		t.Fatalf("ConfirmEdit unexpected error: %v", err)
	}
	if s.IsEditing() {
		t.Error("expected editor to close after valid input")
	}
	if s.items[pattern].StringVal != "^release/.*$" {
		t.Errorf("expected pattern saved, got %q", s.items[pattern].StringVal)
	}
}

func TestSettingsOverlay_ConfirmEdit_EmptyPatternIsValid(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	// Navigate to "Prompt branch pattern" and clear it.
	pattern := selectKey(t, s, "worktree.promptBranchPattern")
	s.StartEditing()
	for s.editBuffer != "" {
		s.DeleteEditChar()
	}

	if err := s.ConfirmEdit(); err != nil {
		t.Fatalf("ConfirmEdit on empty pattern returned error: %v", err)
	}
	if s.items[pattern].StringVal != "" {
		t.Errorf("expected empty pattern saved, got %q", s.items[pattern].StringVal)
	}
}

func TestSettingsOverlay_GHError(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	if s.HasGHError() {
		t.Fatal("should not have GH error initially")
	}

	s.SetGHError("gh not found")
	if !s.HasGHError() {
		t.Fatal("should have GH error after SetGHError")
	}

	s.DismissGHError()
	if s.HasGHError() {
		t.Fatal("should not have GH error after dismiss")
	}
}

func TestSettingsOverlay_Render(t *testing.T) {
	s := NewSettingsOverlay()
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{Setup: "npm install"},
		OnComplete: config.OnCompleteConfig{
			Push:     true,
			CreatePR: false,
		},
	}
	s.LoadFromConfig(cfg)
	s.SetSize(80, 24)

	rendered := s.Render()

	// Check header
	if !strings.Contains(rendered, "Settings") {
		t.Error("expected 'Settings' in header")
	}
	if !strings.Contains(rendered, ".chief/config.yaml") {
		t.Error("expected config path in header")
	}

	// Check section headers
	if !strings.Contains(rendered, "Agent") {
		t.Error("expected 'Agent' section")
	}
	if !strings.Contains(rendered, "Worktree") {
		t.Error("expected 'Worktree' section")
	}
	if !strings.Contains(rendered, "Bash") {
		t.Error("expected 'Bash' section")
	}
	if !strings.Contains(rendered, "On Complete") {
		t.Error("expected 'On Complete' section")
	}

	// Check values
	if !strings.Contains(rendered, "npm install") {
		t.Error("expected 'npm install' value")
	}
	if !strings.Contains(rendered, "Yes") {
		t.Error("expected 'Yes' for push")
	}
	if !strings.Contains(rendered, "No") {
		t.Error("expected 'No' for createPR")
	}

	// Check footer
	if !strings.Contains(rendered, "Esc: close") {
		t.Error("expected 'Esc: close' in footer")
	}
}

func TestSettingsOverlay_RenderEditError(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	s.SetSize(80, 24)

	// Navigate to promptBranchPattern, type a bad regex, attempt confirm.
	selectKey(t, s, "worktree.promptBranchPattern")
	s.StartEditing()
	for s.editBuffer != "" {
		s.DeleteEditChar()
	}
	for _, ch := range "[bad" {
		s.AddEditChar(ch)
	}
	_ = s.ConfirmEdit()

	rendered := s.Render()
	if !strings.Contains(rendered, "invalid regex") {
		t.Error("expected rendered output to contain 'invalid regex' message")
	}
}

func TestSettingsOverlay_RenderGHError(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	s.SetSize(80, 24)

	s.SetGHError("gh not found")
	rendered := s.Render()

	if !strings.Contains(rendered, "GitHub CLI Error") {
		t.Error("expected 'GitHub CLI Error' in rendered output")
	}
	if !strings.Contains(rendered, "gh not found") {
		t.Error("expected error message in rendered output")
	}
	if !strings.Contains(rendered, "Press any key to dismiss") {
		t.Error("expected dismiss hint in footer")
	}
}

func TestSettingsOverlay_RenderEditing(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	s.SetSize(80, 24)

	s.StartEditing()
	rendered := s.Render()

	if !strings.Contains(rendered, "Enter: save") {
		t.Error("expected 'Enter: save' in footer during editing")
	}
}

func TestSettingsOverlay_RenderSelectedIndicator(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	s.SetSize(80, 24)

	rendered := s.Render()

	// The selected item should have a ">" indicator
	if !strings.Contains(rendered, ">") {
		t.Error("expected '>' cursor indicator for selected item")
	}
}

func TestSettingsOverlay_RenderEmptyStringValue(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())
	s.SetSize(80, 24)

	rendered := s.Render()

	if !strings.Contains(rendered, "(not set)") {
		t.Error("expected '(not set)' for empty setup command")
	}
}

func TestSettingsOverlay_GetSelectedItem(t *testing.T) {
	s := NewSettingsOverlay()
	s.LoadFromConfig(config.Default())

	item := s.GetSelectedItem()
	if item == nil {
		t.Fatal("expected non-nil selected item")
	}
	if item.Key != "agent.watchdogTimeout" {
		t.Errorf("expected first item key='agent.watchdogTimeout', got '%s'", item.Key)
	}

	s.MoveDown()
	item = s.GetSelectedItem()
	if item.Key != "worktree.setup" {
		t.Errorf("expected second item key='worktree.setup', got '%s'", item.Key)
	}
}

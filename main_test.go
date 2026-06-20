package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestBuild ensures the package compiles successfully.
func TestBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
}

func runBinary(args ...string) (string, int) {
	bin := "./vibeship_test_bin"
	build := exec.Command("go", "build", "-o", bin, ".")
	if err := build.Run(); err != nil {
		return err.Error(), -1
	}
	cmd := exec.Command(bin, args...)
	out, _ := cmd.CombinedOutput()
	code := cmd.ProcessState.ExitCode()
	exec.Command("rm", "-f", bin).Run()
	return string(out), code
}

func TestDefaultMode(t *testing.T) {
	out, code := runBinary()
	if code != 0 {
		t.Errorf("default mode exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "not yet implemented") {
		t.Errorf("default mode output should mention 'not yet implemented', got: %q", out)
	}
	if !strings.Contains(out, "tui") {
		t.Errorf("default mode output should mention 'tui', got: %q", out)
	}
}

func TestCollectMode(t *testing.T) {
	out, code := runBinary("collect")
	if code != 0 {
		t.Errorf("collect mode exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "not yet implemented") {
		t.Errorf("collect mode output should mention 'not yet implemented', got: %q", out)
	}
	if !strings.Contains(out, "collect") {
		t.Errorf("collect mode output should mention 'collect', got: %q", out)
	}
}

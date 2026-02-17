package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureWorkRepoEmptyPath(t *testing.T) {
	if _, err := EnsureWorkRepo(""); err == nil {
		t.Fatal("expected error for empty work repo path")
	}
}

func TestEnsureWorkRepoCreatesStructureWithoutGitInPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH handling differs on windows")
	}

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "work-repo")

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	_ = os.Setenv("PATH", "")

	warn, err := EnsureWorkRepo(repoPath)
	if err != nil {
		t.Fatalf("ensure work repo: %v", err)
	}
	if !strings.Contains(warn, "git not found") {
		t.Fatalf("expected git-not-found warning, got %q", warn)
	}

	for _, dir := range []string{"requirements", "tasks", "docs", "memory"} {
		p := filepath.Join(repoPath, dir)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", p)
		}
	}
}

func TestEnsureWorkRepoReturnsWarningOnGitInitFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell shim for fake git command")
	}

	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}

	fakeGit := filepath.Join(binDir, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nexit 9\n"), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	_ = os.Setenv("PATH", binDir)

	repoPath := filepath.Join(tmpDir, "work-repo")
	warn, err := EnsureWorkRepo(repoPath)
	if err != nil {
		t.Fatalf("ensure work repo: %v", err)
	}
	if !strings.Contains(warn, "git init failed") {
		t.Fatalf("expected git init warning, got %q", warn)
	}
}

func TestEnsureWorkRepoNoWarningWhenGitDirExists(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "work-repo")
	if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	warn, err := EnsureWorkRepo(repoPath)
	if err != nil {
		t.Fatalf("ensure work repo: %v", err)
	}
	if warn != "" {
		t.Fatalf("expected no warning, got %q", warn)
	}
}

func TestResolveArtifactPath(t *testing.T) {
	if _, err := ResolveArtifactPath("", "docs", "x.md"); err == nil {
		t.Fatal("expected error for empty work repo path")
	}

	got, err := ResolveArtifactPath("/tmp/work", "tasks", "item.md")
	if err != nil {
		t.Fatalf("resolve tasks path: %v", err)
	}
	if got != filepath.Join("/tmp/work", "tasks", "item.md") {
		t.Fatalf("unexpected tasks path: %q", got)
	}

	got, err = ResolveArtifactPath("/tmp/work", "invalid-kind", "")
	if err != nil {
		t.Fatalf("resolve default path: %v", err)
	}
	if got != filepath.Join("/tmp/work", "docs", "artifact.md") {
		t.Fatalf("unexpected default path: %q", got)
	}
}

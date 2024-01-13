package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DavidGamba/dgtools/buildutils"
	"github.com/DavidGamba/dgtools/fsmodtime"
	"github.com/DavidGamba/dgtools/run"
)

func buildPlugin(dir string) error {
	files, modified, err := fsmodtime.Target(os.DirFS(dir),
		[]string{"bake.so"},
		[]string{"*.go", "go.mod", "go.sum"})
	if err != nil {
		return err
	}
	if modified {
		Logger.Printf("Found modifications on %v, rebuilding...\n", files)
		// Debug flags
		// return run.CMD("go", "build", "-buildmode=plugin", "-o=bake.so", "-trimpath", "-gcflags", "all=-N -l").Dir(dir).Log().Run()
		return run.CMD("go", "build", "-buildmode=plugin", "-o=bake.so").Dir(dir).Log().Run()
	}
	return nil
}

func findBakeDir(ctx context.Context) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// First case, we are withing the bake folder
	base := filepath.Base(wd)
	if base == "bake" {
		return ".", nil
	}

	// Second case, bake folder lives in CWD
	dir := filepath.Join(wd, "bake")
	if fi, err := os.Stat(dir); err == nil && fi.Mode().IsDir() {
		return dir, nil
	}

	// Third case, bake folder lives in module root
	modRoot, err := buildutils.GoModDir()
	if err != nil {
		return "", fmt.Errorf("failed to get go project root: %w", err)
	}
	dir = filepath.Join(modRoot, "bake")
	if fi, err := os.Stat(dir); err == nil && fi.Mode().IsDir() {
		return dir, nil
	}

	// Fourth case, bake folder lives in root of repo
	root, err := buildutils.GitRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get git repo root: %w", err)
	}
	dir = filepath.Join(root, "bake")
	if fi, err := os.Stat(dir); err == nil && fi.Mode().IsDir() {
		return dir, nil
	}

	return "", fmt.Errorf("bake directory not found")
}

func findBakeFiles(ctx context.Context) (string, error) {
	dir, err := findBakeDir(ctx)
	if err != nil {
		return "", err
	}

	err = buildPlugin(dir)
	if err != nil {
		return "", fmt.Errorf("failed to build: %w", err)
	}
	return filepath.Join(dir, "bake.so"), nil
}

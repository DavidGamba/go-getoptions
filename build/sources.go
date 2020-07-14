package build

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TODO: Print or return the srcs that were modified after the targets

// SourceAfter - Check if any of the sources was modified after any of the targets.
func SourceAfter(targets []string, srcs ...string) (bool, error) {
	sourceLatest, err := getLatestModTime(false, srcs...)
	if err != nil {
		// Missing sources or errors found, sources have been modified.
		return true, err
	}
	targetEarliest, err := getEarliestModTime(false, targets...)
	if err != nil {
		// Missing targets or errors found, need to regenerate targets.
		return true, err
	}
	if sourceLatest.After(targetEarliest) {
		// Sources modified after targets
		return true, nil
	}
	return false, nil
}

// getLatestModTime - Gets the latest modTime for a list of files.
// If any of the given files doesn't exist and ignoreMissing is set they are ignored.
func getLatestModTime(ignoreMissing bool, files ...string) (time.Time, error) {
	var latest time.Time
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			if ignoreMissing && os.IsNotExist(err) {
				continue
			}
			return latest, err
		}
		t := stat.ModTime()
		if t.After(latest) {
			latest = t
		}
	}
	return latest, nil
}

// getEarliestModTime - Gets the earliest modTime for a list of files.
// If any of the given files doesn't exist and ignoreMissing is set they are ignored.
func getEarliestModTime(ignoreMissing bool, files ...string) (time.Time, error) {
	var earliest time.Time
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			if ignoreMissing && os.IsNotExist(err) {
				continue
			}
			return earliest, err
		}
		t := stat.ModTime()
		if t.Before(earliest) {
			earliest = t
		}
	}
	return earliest, nil
}

var ErrNoMatchGlob = fmt.Errorf("glob didn't match any files")

// GlobExclude - Expands a list of globs and removes the expanded list of removes.
// It uses filepath.Glob for expansion so the glob semantics, details on glob syntax available here:
// https://golang.org/pkg/path/filepath/#Match
func GlobExclude(ignoreMissing bool, srcs []string, excludes ...string) ([]string, error) {
	m := map[string]struct{}{}
	for _, src := range srcs {
		files, err := filepath.Glob(src)
		if err != nil {
			return []string{}, err
		}
		if !ignoreMissing && len(files) == 0 {
			return []string{}, fmt.Errorf("%w: %s", ErrNoMatchGlob, src)
		}
		for _, file := range files {
			m[file] = struct{}{}
		}
	}
	for _, exclude := range excludes {
		files, err := filepath.Glob(exclude)
		if err != nil {
			return []string{}, err
		}
		if !ignoreMissing && len(files) == 0 {
			return []string{}, fmt.Errorf("%w: %s", ErrNoMatchGlob, exclude)
		}
		for _, file := range files {
			delete(m, file)
		}
	}
	results := []string{}
	for k := range m {
		results = append(results, k)
	}
	return results, nil
}

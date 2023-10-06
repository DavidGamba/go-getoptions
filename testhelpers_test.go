package getoptions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
)

func checkError(t *testing.T, got, expected error) {
	t.Helper()
	if (got == nil && expected != nil) || (got != nil && expected == nil) || (got != nil && expected != nil && !errors.Is(got, expected)) {
		t.Errorf("wrong error received: got = '%#v', want '%#v'", got, expected)
	}
}

func setupLogging() *bytes.Buffer {
	s := ""
	buf := bytes.NewBufferString(s)
	Logger.SetOutput(buf)
	return buf
}

// setupTestLogging - Defines an output for the default Logger and returns a
// function that prints the output if the output is not empty.
//
// Usage:
//
//	logTestOutput := setupTestLogging(t)
//	defer logTestOutput()
func setupTestLogging(t *testing.T) func() {
	s := ""
	buf := bytes.NewBufferString(s)
	Logger.SetOutput(buf)
	return func() {
		if len(buf.String()) > 0 {
			t.Log("\n" + buf.String())
		}
	}
}

// Test helper to compare two string outputs and find the first difference
func firstDiff(got, expected string) string {
	same := ""
	for i, gc := range got {
		if len([]rune(expected)) <= i {
			return fmt.Sprintf("got:\n%s\nIndex: %d | diff: got '%s' - exp '%s'\n", got, len(expected), got, expected)
		}
		if gc != []rune(expected)[i] {
			return fmt.Sprintf("got:\n%s\nIndex: %d | diff: got '%c' - exp '%c'\n%s\n", got, i, gc, []rune(expected)[i], same)
		}
		same += string(gc)
	}
	if len(expected) > len(got) {
		return fmt.Sprintf("got:\n%s\nIndex: %d | diff: got '%s' - exp '%s'\n", got, len(got), got, expected)
	}
	return ""
}

func getNode(tree *programTree, element ...string) (*programTree, error) {
	if len(element) == 0 {
		return tree, nil
	}
	if child, ok := tree.ChildCommands[element[0]]; ok {
		return getNode(child, element[1:]...)
	}
	return tree, fmt.Errorf("not found")
}

func stringPT(n *programTree) string {
	data, err := json.MarshalIndent(n.str(), "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func programTreeError(expected, got *programTree) string {
	return fmt.Sprintf("expected:\n%s\ngot:\n%s\n", stringPT(expected), stringPT(got))
}

func spewToFileDiff(t *testing.T, expected, got interface{}) string {
	return fmt.Sprintf("expected, got: %s %s\n", spewToFile(t, expected, "expected"), spewToFile(t, got, "got"))
}

func spewToFile(t *testing.T, e interface{}, label string) string {
	f, err := os.CreateTemp("/tmp/", "spew-")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, _ = f.WriteString(label + "\n")
	fmt.Fprintf(f, "%v\n", e)
	return f.Name()
}

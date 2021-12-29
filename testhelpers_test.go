package getoptions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	// "github.com/davecgh/go-spew/spew"
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
//   logTestOutput := setupTestLogging(t)
//   defer logTestOutput()
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
	// spewToFileDiff - This implementation shouldn't make it to a release, I don't want any external dependencies for this package.
	// spewConfig()
	return fmt.Sprintf("expected, got: %s %s\n", spewToFile(t, expected, "expected"), spewToFile(t, got, "got"))

	// comment out for release
	// return ""
}

// func spewConfig() {
// 	spew.Config = spew.ConfigState{
// 		Indent:                  "  ",
// 		MaxDepth:                0,
// 		DisableMethods:          false,
// 		DisablePointerMethods:   false,
// 		DisablePointerAddresses: true,
// 		DisableCapacities:       true,
// 		ContinueOnMethod:        false,
// 		SortKeys:                true,
// 		SpewKeys:                false,
// 	}
// }

func spewToFile(t *testing.T, e interface{}, label string) string {
	f, err := ioutil.TempFile("/tmp/", "spew-")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, _ = f.WriteString(label + "\n")
	// spew.Fdump(f, e)
	fmt.Fprintf(f, "%v\n", e)
	return f.Name()
}

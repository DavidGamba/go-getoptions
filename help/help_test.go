package help

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHelp(t *testing.T) {
	outputName := HelpName(filepath.Base(os.Args[0]), "", "")

	expectedName := `NAME:
    help.test
`
	if outputName != expectedName {
		t.Errorf("Got: '%s', expected: '%s'\n", outputName, expectedName)
	}

	// 	outputSynopsis := HelpSynopsis(filepath.Base(os.Args[0]), "", nil, []string{})
	// 	expectedSynopsis := `SYNOPSIS:
	//     help.test --int <int> <--req-list <item>...>... --str <string>
	//                        [--flag|-f] [--float|--fl <float64>] [--intSlice <int>]...
	//                        [--list <string>]... [--strMap <key=value>...]...
	//                        [--strSlice <my_value>...]... [--string <string>]
	//                        <command> [<args>]
	// `
	// if outputSynopsis != expectedSynopsis {
	// 	t.Errorf("Got: '%s', expected: '%s'\n", outputSynopsis, expectedSynopsis)
	// }
	firstDiff := func(got, expected string) string {
		same := ""
		for i, gc := range got {
			if len([]rune(expected)) <= i {
				return fmt.Sprintf("Index: %d | diff: got '%s' - exp '%s'\n", len(expected), got, expected)
			}
			if gc != []rune(expected)[i] {
				return fmt.Sprintf("Index: %d | diff: got '%c' - exp '%c'\n%s\n", i, gc, []rune(expected)[i], same)
			} else {
				same += string(gc)
			}
		}
		if len(expected) > len(got) {
			return fmt.Sprintf("Index: %d | diff: got '%s' - exp '%s'\n", len(got), got, expected)
		}
		return ""
	}
	if outputName != expectedName {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", outputName, expectedName)
		t.Fatalf("Unexpected name:\n%s", firstDiff(outputName, expectedName))
	}
	// if outputSynopsis != expectedSynopsis {
	// 	fmt.Printf("got:\n%s\nexpected:\n%s\n", outputSynopsis, expectedSynopsis)
	// 	t.Fatalf("Unexpected synopsis:\n%s", firstDiff(outputSynopsis, expectedSynopsis))
	// }
}

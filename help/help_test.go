package help

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DavidGamba/go-getoptions/option"
)

func firstDiff(got, expected string) string {
	same := ""
	for i, gc := range got {
		if len([]rune(expected)) <= i {
			return fmt.Sprintf("Index: %d | diff: got '%s' - exp '%s'\n", len(expected), got, expected)
		}
		if gc != []rune(expected)[i] {
			return fmt.Sprintf("Index: %d | diff: got '%c' - exp '%c'\nsame '%s'\n", i, gc, []rune(expected)[i], same)
		}
		same += string(gc)
	}
	if len(expected) > len(got) {
		return fmt.Sprintf("Index: %d | diff: got '%s' - exp '%s'\n", len(got), got, expected)
	}
	return ""
}

func TestHelp(t *testing.T) {
	scriptName := filepath.Base(os.Args[0])

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Name", Name(scriptName, "", ""), `NAME:
    help.test
`},
		{"Name", Name(scriptName, "log", ""), `NAME:
    help.test log
`},
		{"Name", Name(scriptName, "log", "logs output..."), `NAME:
    help.test log - logs output...
`},
		{"Synopsis", Synopsis(scriptName, "", "", nil, []string{}), `SYNOPSIS:
    help.test [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "", nil, []string{}), `SYNOPSIS:
    help.test log [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "<filename>", nil, []string{}), `SYNOPSIS:
    help.test log <filename>
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{option.New("bool", option.BoolType)}, []string{}),
			`SYNOPSIS:
    help.test log [--bool] [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{option.New("bool", option.BoolType).SetAlias("b")}, []string{}),
			`SYNOPSIS:
    help.test log [--bool|-b] [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				option.New("bool", option.BoolType).SetAlias("b"),
				option.New("int", option.IntType),
				option.New("float", option.Float64Type),
				option.New("ss", option.StringRepeatType),
				option.New("ii", option.IntRepeatType),
				option.New("m", option.StringMapType),
			}, []string{}),
			`SYNOPSIS:
    help.test log [--bool|-b] [--float <float64>] [--ii <int>]... [--int <int>]
                  [-m <key=value>]... [--ss <string>]... [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				option.New("bool", option.BoolType).SetAlias("b").SetRequired(""),
				option.New("int", option.IntType).SetRequired(""),
				option.New("float", option.Float64Type).SetRequired(""),
				option.New("ss", option.StringRepeatType).SetRequired(""),
				option.New("ii", option.IntRepeatType).SetRequired(""),
				option.New("m", option.StringMapType).SetRequired(""),
			}, []string{}),
			`SYNOPSIS:
    help.test log --bool|-b --float <float64> <--ii <int>>... --int <int>
                  <-m <key=value>>... <--ss <string>>... [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				option.New("bool", option.BoolType).SetAlias("b").SetRequired(""),
				option.New("int", option.IntType).SetRequired(""),
				option.New("float", option.Float64Type).SetRequired(""),
				option.New("ss", option.StringRepeatType).SetRequired(""),
				option.New("ii", option.IntRepeatType).SetRequired(""),
				option.New("m", option.StringMapType).SetRequired(""),
			}, []string{"log", "show"}),
			`SYNOPSIS:
    help.test log --bool|-b --float <float64> <--ii <int>>... --int <int>
                  <-m <key=value>>... <--ss <string>>... <command> [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				option.New("bool", option.BoolType).SetAlias("b").SetRequired(""),
				option.New("int", option.IntType).SetRequired(""),
				option.New("float", option.Float64Type).SetRequired(""),
				option.New("ss", option.StringRepeatType).SetRequired(""),
				option.New("ii", option.IntRepeatType).SetRequired(""),
				option.New("m", option.StringMapType).SetRequired(""),
				option.New("z", option.StringMapType).SetRequired(""),
			}, []string{"log", "show"}),
			`SYNOPSIS:
    help.test log --bool|-b --float <float64> <--ii <int>>... --int <int>
                  <-m <key=value>>... <--ss <string>>... <-z <key=value>>...
                  <command> [<args>]
`},
		{"OptionList", OptionList(nil), ""},
		{"OptionList", OptionList([]*option.Option{}), ""},
		{"OptionList", OptionList([]*option.Option{
			option.New("bool", option.BoolType).SetAlias("b").SetDefaultStr("false"),
			option.New("int", option.IntType).SetDefaultStr("0"),
			option.New("float", option.Float64Type).SetDefaultStr("0.0"),
			option.New("ss", option.StringRepeatType).SetDefaultStr("[]"),
			option.New("ii", option.IntRepeatType).SetDefaultStr("[]"),
			option.New("m", option.StringMapType).SetDefaultStr("{}"),
		}), `OPTIONS:
    --bool|-b            (default: false)

    --float <float64>    (default: 0.0)

    --ii <int>           (default: [])

    --int <int>          (default: 0)

    -m <key=value>       (default: {})

    --ss <string>        (default: [])

`},
		{"OptionList", OptionList([]*option.Option{
			option.New("bool", option.BoolType).SetAlias("b").SetRequired(""),
			option.New("int", option.IntType).SetRequired(""),
			option.New("float", option.Float64Type).SetRequired(""),
			option.New("ss", option.StringRepeatType).SetRequired(""),
			option.New("ii", option.IntRepeatType).SetRequired(""),
			option.New("m", option.StringMapType).SetRequired(""),
		}), `REQUIRED PARAMETERS:
    --bool|-b

    --float <float64>

    --ii <int>

    --int <int>

    -m <key=value>

    --ss <string>

`},
		{"OptionList", OptionList([]*option.Option{
			option.New("bool", option.BoolType).SetAlias("b").SetDefaultStr("false").SetDescription("bool"),
			option.New("int", option.IntType).SetDefaultStr("0").SetDescription("int\nmultiline description"),
			option.New("float", option.Float64Type).SetDefaultStr("0.0").SetDescription("float"),
			option.New("ss", option.StringRepeatType).SetDefaultStr("[]").SetDescription("string repeat"),
			option.New("ii", option.IntRepeatType).SetDefaultStr("[]").SetDescription("int repeat"),
			option.New("m", option.StringMapType).SetDefaultStr("{}").SetDescription("map"),
		}), `OPTIONS:
    --bool|-b            bool (default: false)

    --float <float64>    float (default: 0.0)

    --ii <int>           int repeat (default: [])

    --int <int>          int
                         multiline description (default: 0)

    -m <key=value>       map (default: {})

    --ss <string>        string repeat (default: [])

`},
		{"OptionList", OptionList([]*option.Option{
			option.New("bool", option.BoolType).SetAlias("b").SetDefaultStr("false").SetDescription("bool").SetRequired(""),
			option.New("int", option.IntType).SetDefaultStr("0").SetDescription("int\nmultiline description"),
			option.New("float", option.Float64Type).SetDefaultStr("0.0").SetDescription("float").SetRequired(""),
			option.New("string-repeat", option.StringRepeatType).SetDefaultStr("[]").SetDescription("string repeat").SetHelpArgName("my_value"),
			option.New("ii", option.IntRepeatType).SetDefaultStr("[]").SetDescription("int repeat").SetRequired(""),
			option.New("m", option.StringMapType).SetDefaultStr("{}").SetDescription("map"),
		}), `REQUIRED PARAMETERS:
    --bool|-b                     bool

    --float <float64>             float

    --ii <int>                    int repeat

OPTIONS:
    --int <int>                   int
                                  multiline description (default: 0)

    -m <key=value>                map (default: {})

    --string-repeat <my_value>    string repeat (default: [])

`},
		{"CommandList", CommandList(nil), ""},
		{"CommandList", CommandList(map[string]string{}), ""},
		{"CommandList", CommandList(
			map[string]string{"log": "log output", "show": "show output"},
		), `COMMANDS:
    log     log output
    show    show output
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Error\ngot: %s\n%s", tt.got, firstDiff(tt.got, tt.expected))
			}
		})
	}
}

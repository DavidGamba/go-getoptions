package help

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DavidGamba/go-getoptions/internal/option"
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

	boolOpt := func() *option.Option { b := false; return option.New("bool", option.BoolType, &b).SetAlias("b") }
	intOpt := func() *option.Option { i := 0; return option.New("int", option.IntType, &i) }
	floatOpt := func() *option.Option { f := 0.0; return option.New("float", option.Float64Type, &f) }
	ssOpt := func() *option.Option { ss := []string{}; return option.New("ss", option.StringRepeatType, &ss) }
	iiOpt := func() *option.Option { ii := []int{}; return option.New("ii", option.IntRepeatType, &ii) }
	mOpt := func() *option.Option { m := map[string]string{}; return option.New("m", option.StringMapType, &m) }

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Name", Name("", scriptName, ""), `NAME:
    help.test
`},
		{"Name", Name(scriptName, "log", ""), `NAME:
    help.test log
`},
		{"Name", Name(scriptName, "log", "logs output..."), `NAME:
    help.test log - logs output...
`},
		{"Name", Name(scriptName, "multiline", "multiline\ndescription\nthat is very long"), `NAME:
    help.test multiline - multiline
        description
        that is very long
`},
		{"Synopsis", Synopsis("", scriptName, "", nil, []string{}), `SYNOPSIS:
    help.test [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "", nil, []string{}), `SYNOPSIS:
    help.test log [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "<filename>", nil, []string{}), `SYNOPSIS:
    help.test log <filename>
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{func() *option.Option { b := false; return option.New("bool", option.BoolType, &b) }()}, []string{}),
			`SYNOPSIS:
    help.test log [--bool] [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{boolOpt()}, []string{}),
			`SYNOPSIS:
    help.test log [--bool|-b] [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				boolOpt(),
				intOpt(),
				floatOpt(),
				ssOpt(),
				iiOpt(),
				mOpt(),
			}, []string{}),
			`SYNOPSIS:
    help.test log [--bool|-b] [--float <float64>] [--ii <int>]... [--int <int>]
                  [-m <key=value>]... [--ss <string>]... [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				boolOpt().SetRequired(""),
				intOpt().SetRequired(""),
				floatOpt().SetRequired(""),
				ssOpt().SetRequired(""),
				iiOpt().SetRequired(""),
				mOpt().SetRequired(""),
			}, []string{}),
			`SYNOPSIS:
    help.test log --bool|-b --float <float64> <--ii <int>>... --int <int>
                  <-m <key=value>>... <--ss <string>>... [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				boolOpt().SetRequired(""),
				intOpt().SetRequired(""),
				floatOpt().SetRequired(""),
				ssOpt().SetRequired(""),
				iiOpt().SetRequired(""),
				mOpt().SetRequired(""),
			}, []string{"log", "show"}),
			`SYNOPSIS:
    help.test log --bool|-b --float <float64> <--ii <int>>... --int <int>
                  <-m <key=value>>... <--ss <string>>... <command> [<args>]
`},
		{"Synopsis", Synopsis(scriptName, "log", "",
			[]*option.Option{
				boolOpt().SetRequired(""),
				intOpt().SetRequired(""),
				floatOpt().SetRequired(""),
				ssOpt().SetRequired(""),
				iiOpt().SetRequired(""),
				mOpt().SetRequired(""),
				func() *option.Option { m := map[string]string{}; return option.New("z", option.StringMapType, &m) }().SetRequired(""),
			}, []string{"log", "show"}),
			`SYNOPSIS:
    help.test log --bool|-b --float <float64> <--ii <int>>... --int <int>
                  <-m <key=value>>... <--ss <string>>... <-z <key=value>>...
                  <command> [<args>]
`},
		{"OptionList nil", OptionList(nil), ""},
		{"OptionList empty", OptionList([]*option.Option{}), ""},
		{"OptionList default str", OptionList([]*option.Option{
			boolOpt().SetDefaultStr("false"),
			intOpt().SetDefaultStr("0"),
			floatOpt().SetDefaultStr("0.0"),
			ssOpt().SetDefaultStr("[]"),
			iiOpt().SetDefaultStr("[]"),
			mOpt().SetDefaultStr("{}"),
		}), `OPTIONS:
    --bool|-b            (default: false)

    --float <float64>    (default: 0.0)

    --ii <int>           (default: [])

    --int <int>          (default: 0)

    -m <key=value>       (default: {})

    --ss <string>        (default: [])

`},
		{"OptionList required", OptionList([]*option.Option{
			boolOpt().SetRequired(""),
			intOpt().SetRequired(""),
			floatOpt().SetRequired(""),
			ssOpt().SetRequired(""),
			iiOpt().SetRequired(""),
			mOpt().SetRequired(""),
		}),
			`REQUIRED PARAMETERS:
    --bool|-b

    --float <float64>

    --ii <int>

    --int <int>

    -m <key=value>

    --ss <string>

`},
		{"OptionList multi line", OptionList([]*option.Option{
			boolOpt().SetDefaultStr("false").SetDescription("bool"),
			intOpt().SetDefaultStr("0").SetDescription("int\nmultiline description"),
			floatOpt().SetDefaultStr("0.0").SetDescription("float"),
			ssOpt().SetDefaultStr("[]").SetDescription("string repeat"),
			iiOpt().SetDefaultStr("[]").SetDescription("int repeat"),
			mOpt().SetDefaultStr("{}").SetDescription("map"),
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
			boolOpt().SetDefaultStr("false").SetDescription("bool").SetRequired(""),
			intOpt().SetDefaultStr("0").SetDescription("int\nmultiline description"),
			floatOpt().SetDefaultStr("0.0").SetDescription("float").SetRequired(""),
			func() *option.Option {
				ss := []string{}
				return option.New("string-repeat", option.StringRepeatType, &ss)
			}().SetDefaultStr("[]").SetDescription("string repeat").SetHelpArgName("my_value"),
			iiOpt().SetDefaultStr("[]").SetDescription("int repeat").SetRequired(""),
			mOpt().SetDefaultStr("{}").SetDescription("map"),
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
		{"OptionList", OptionList([]*option.Option{
			boolOpt().SetDefaultStr("false").SetDescription("bool").SetRequired("").SetEnvVar("BOOL"),
			intOpt().SetDefaultStr("0").SetDescription("int\nmultiline description").SetEnvVar("INT"),
			floatOpt().SetDefaultStr("0.0").SetDescription("float").SetRequired("").SetEnvVar("FLOAT"),
			func() *option.Option {
				ss := []string{}
				return option.New("string-repeat", option.StringRepeatType, &ss)
			}().SetDefaultStr("[]").SetDescription("string repeat").SetHelpArgName("my_value").SetEnvVar("STRING_REPEAT"),
			iiOpt().SetDefaultStr("[]").SetDescription("int repeat").SetRequired("").SetEnvVar("II"),
			mOpt().SetDefaultStr("{}").SetDescription("map").SetEnvVar("M"),
		}), `REQUIRED PARAMETERS:
    --bool|-b                     bool (env: BOOL)

    --float <float64>             float (env: FLOAT)

    --ii <int>                    int repeat (env: II)

OPTIONS:
    --int <int>                   int
                                  multiline description (default: 0, env: INT)

    -m <key=value>                map (default: {}, env: M)

    --string-repeat <my_value>    string repeat (default: [], env: STRING_REPEAT)

`},
		{"CommandList", CommandList(nil), ""},
		{"CommandList", CommandList(map[string]string{}), ""},
		{"CommandList", CommandList(
			map[string]string{"log": "log output", "show": "show output", "multi": "multiline\ndescription\nthat is long"},
		), `COMMANDS:
    log      log output
    multi    multiline
             description
             that is long
    show     show output
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

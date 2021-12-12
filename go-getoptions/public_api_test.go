package getoptions_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	// "github.com/DavidGamba/go-getoptions"
	"github.com/DavidGamba/go-getoptions/go-getoptions"
	"github.com/DavidGamba/go-getoptions/option"
	"github.com/DavidGamba/go-getoptions/text"
)

func TestDefinitionPanics(t *testing.T) {
	recoverFn := func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Errorf("definition did not panic")
		}
	}
	t.Run("Option double defined", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.Bool("flag", false)
		opt.Bool("flag", false)
	})
	t.Run("Option double defined by alias", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.Bool("flag", false)
		opt.Bool("fleg", false, opt.Alias("flag"))
	})
	t.Run("Alias double defined", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f"))
		opt.Bool("fleg", false, opt.Alias("f"))
	})
	t.Run("Option double defined across commands", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.Bool("flag", false)
		cmd := opt.NewCommand("cmd", "")
		cmd.Bool("flag", false)
	})
	t.Run("Option double defined across commands by alias", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.Bool("flag", false)
		cmd := opt.NewCommand("cmd", "")
		cmd.Bool("fleg", false, opt.Alias("flag"))
	})
	t.Run("Alias double defined across commands", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f"))
		cmd := opt.NewCommand("cmd", "")
		cmd.Bool("fleg", false, opt.Alias("f"))
	})
	t.Run("Command double defined", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.NewCommand("cmd", "")
		opt.NewCommand("cmd", "")
	})
	t.Run("Option name is empty", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().Bool("", false)
	})
	t.Run("Command name is empty", func(t *testing.T) {
		defer recoverFn()
		opt := getoptions.New()
		opt.NewCommand("", "")
	})
}

func TestOptionWrongMinMax(t *testing.T) {
	recoverFn := func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Errorf("wrong min/max definition did not panic")
		}
	}

	t.Run("StringSlice min < 1", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().StringSlice("ss", 0, 1)
	})
	t.Run("IntSlice min < 1", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().IntSlice("ss", 0, 1)
	})

	t.Run("StringSlice max < 1", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().StringSlice("ss", 1, 0)
	})
	t.Run("IntSlice max < 1", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().IntSlice("ss", 1, 0)
	})

	t.Run("StringSlice min > max", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().StringSlice("ss", 2, 1)
	})
	t.Run("IntSlice min > max", func(t *testing.T) {
		defer recoverFn()
		getoptions.New().IntSlice("ss", 2, 1)
	})
}

func TestRequired(t *testing.T) {
	t.Run("error raised when called", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Required())
		_, err := opt.Parse([]string{"--flag"})
		if err != nil {
			t.Errorf("Required option called but error raised")
		}
	})
	t.Run("error not raised", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Required())
		_, err := opt.Parse([]string{})
		if err == nil {
			t.Errorf("Required option missing didn't raise error")
		}
		if err != nil && !errors.Is(err, option.ErrorMissingRequiredOption) {
			t.Errorf("Error type didn't match")
		}
		if err != nil && err.Error() != "Missing required parameter 'flag'" {
			t.Errorf("Error string didn't match expected value")
		}
	})
	t.Run("custom message", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Required("please provide 'flag'"))
		_, err := opt.Parse([]string{})
		if err == nil {
			t.Errorf("Required option missing didn't raise error")
		}
		if err != nil && !errors.Is(err, option.ErrorMissingRequiredOption) {
			t.Errorf("Error type didn't match")
		}
		if err != nil && err.Error() != "please provide 'flag'" {
			t.Errorf("Error string didn't match expected value")
		}
	})
}

func TestUnknownOptionModes(t *testing.T) {
	t.Run("default fail", func(t *testing.T) {
		opt := getoptions.New()
		_, err := opt.Parse([]string{"--flags"})
		if err == nil {
			t.Errorf("Unknown option 'flags' didn't raise error")
		}
		if err != nil && err.Error() != "Unknown option 'flags'" {
			t.Errorf("Error string didn't match expected value: %s\n", err)
		}
	})
	t.Run("explicit fail", func(t *testing.T) {
		opt := getoptions.New()
		opt.SetUnknownMode(getoptions.Fail)
		_, err := opt.Parse([]string{"--flags"})
		if err == nil {
			t.Errorf("Unknown option 'flags' didn't raise error")
		}
		if err != nil && err.Error() != "Unknown option 'flags'" {
			t.Errorf("Error string didn't match expected value: %s\n", err)
		}
	})
	t.Run("warn", func(t *testing.T) {
		buf := new(bytes.Buffer)
		opt := getoptions.New()
		getoptions.Writer = buf
		opt.SetUnknownMode(getoptions.Warn)
		remaining, err := opt.Parse([]string{"--flags", "--flegs"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if buf.String() != fmt.Sprintf("WARNING: Unknown option '%s'\nWARNING: Unknown option '%s'\n", "flags", "flegs") {
			t.Errorf("Warning message didn't match expected value: %s", buf.String())
		}
		if !reflect.DeepEqual(remaining, []string{"--flags", "--flegs"}) {
			t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags", "--flegs"})
		}
	})
	t.Run("first unknown argument as a passthrough", func(t *testing.T) {
		buf := new(bytes.Buffer)
		opt := getoptions.New()
		getoptions.Writer = buf
		opt.SetUnknownMode(getoptions.Pass)
		remaining, err := opt.Parse([]string{"--flags"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if buf.String() != "" {
			t.Errorf("output didn't match expected value: %s", buf.String())
		}
		if !reflect.DeepEqual(remaining, []string{"--flags"}) {
			t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags"})
		}
	})
	t.Run("first unknown argument as a passthrough with a known one after", func(t *testing.T) {
		buf := new(bytes.Buffer)
		opt := getoptions.New()
		getoptions.Writer = buf
		opt.Bool("known", false)
		opt.Bool("another", false)
		opt.SetUnknownMode(getoptions.Pass)
		remaining, err := opt.Parse([]string{"--flags", "--known", "--another", "--unknown", "--unknown-2", "--unknown-3", "--unknown-4"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if buf.String() != "" {
			t.Errorf("output didn't match expected value: %s", buf.String())
		}
		expected := []string{"--flags", "--unknown", "--unknown-2", "--unknown-3", "--unknown-4"}
		if !reflect.DeepEqual(remaining, expected) {
			t.Errorf("remaining didn't have expected value: %v != %v", remaining, expected)
		}
		if !opt.Called("known") && !opt.Called("another") {
			t.Errorf("known or another were not called")
		}
	})
}

func TestOptionals(t *testing.T) {
	t.Run("missing argument for non optional", func(t *testing.T) {
		opt := getoptions.New()
		opt.String("string", "default")
		_, err := opt.Parse([]string{"--string"})
		if err == nil {
			t.Errorf("Missing argument for option 'string' didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
			t.Errorf("Error string didn't match expected value")
		}
	})

	t.Run("missing argument string", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringOptional("string", "default", opt.Alias("alias"))
		_, err := opt.Parse([]string{"--string"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("string") != "default" {
			t.Errorf("Default value not set for 'string'")
		}
	})

	t.Run("missing argument int", func(t *testing.T) {
		opt := getoptions.New()
		opt.IntOptional("int", 123, opt.Alias("alias"))
		_, err := opt.Parse([]string{"--int"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("int") != 123 {
			t.Errorf("Default value not set for 'int'")
		}
	})

	t.Run("missing argument float", func(t *testing.T) {
		opt := getoptions.New()
		opt.Float64Optional("float", 123.123, opt.Alias("alias"))
		_, err := opt.Parse([]string{"--float"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("float") != 123.123 {
			t.Errorf("Default value not set for 'int'")
		}
	})

	t.Run("missing argument, next argument is option", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringOptional("string", "default")
		opt.IntOptional("int", 123)
		opt.Float64Optional("float", 123.123)
		opt.Bool("flag", false)
		_, err := opt.Parse([]string{"--string", "--int", "--float", "--flag"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("string") != "default" {
			t.Errorf("Default value not set for 'string'")
		}
		if opt.Value("int") != 123 {
			t.Errorf("Default value not set for 'int'")
		}
		if opt.Value("float") != 123.123 {
			t.Errorf("Default value not set for 'float'")
		}
	})

	t.Run("argument given", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringOptional("string", "default")
		opt.IntOptional("int", 123)
		opt.Float64Optional("float", 123.123)
		_, err := opt.Parse([]string{"--string=arg", "--int=456", "--float=456.456"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("string") != "arg" {
			t.Errorf("string optional didn't take argument")
		}
		if opt.Value("int") != 456 {
			t.Errorf("int optional didn't take argument")
		}
		if opt.Value("float") != 456.456 {
			t.Errorf("float optional didn't take argument")
		}
	})
	t.Run("argument given", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringOptional("string", "default")
		opt.IntOptional("int", 123)
		opt.Float64Optional("float", 123.123)
		_, err := opt.Parse([]string{"--string", "arg", "--int", "456", "--float", "456.456"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("string") != "arg" {
			t.Errorf("string Optional didn't take argument")
		}
		if opt.Value("int") != 456 {
			t.Errorf("int Optional didn't take argument")
		}
		if opt.Value("float") != 456.456 {
			t.Errorf("float optional didn't take argument")
		}
	})

	t.Run("varOptional", func(t *testing.T) {
		var result string
		var i int
		var f float64
		opt := getoptions.New()
		opt.StringVarOptional(&result, "string", "default")
		opt.IntVarOptional(&i, "int", 123)
		opt.Float64VarOptional(&f, "float", 123.123)
		_, err := opt.Parse([]string{"--string=arg", "--int=456", "--float=456.456"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if result != "arg" {
			t.Errorf("StringVarOptional didn't take argument")
		}
		if i != 456 {
			t.Errorf("IntVarOptional didn't take argument")
		}
		if f != 456.456 {
			t.Errorf("FloatVarOptional optional didn't take argument")
		}
	})

	t.Run("varOptional alone", func(t *testing.T) {
		result := ""
		opt := getoptions.New()
		opt.StringVarOptional(&result, "string", "default")
		_, err := opt.Parse([]string{"--string=arg"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if result != "arg" {
			t.Errorf("StringVarOptional didn't take argument")
		}
	})

	t.Run("varOptional alone", func(t *testing.T) {
		i := 0
		opt := getoptions.New()
		opt.IntVarOptional(&i, "int", 123)
		_, err := opt.Parse([]string{"--int=456"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if i != 456 {
			t.Errorf("IntVarOptional didn't take argument")
		}
	})

	t.Run("varOptional alone", func(t *testing.T) {
		f := 0.0
		opt := getoptions.New()
		opt.Float64VarOptional(&f, "float", 123.123)
		_, err := opt.Parse([]string{"--float=456.456"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if f != 456.456 {
			t.Errorf("FloatVarOptional didn't take argument")
		}
	})

	t.Run("varOptional", func(t *testing.T) {
		result := ""
		opt := getoptions.New()
		opt.StringVarOptional(&result, "string", "default")
		_, err := opt.Parse([]string{"--string"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if result != "default" {
			t.Errorf("Default value not set for 'string'")
		}
	})

	t.Run("varOptional", func(t *testing.T) {
		i := 0
		opt := getoptions.New()
		opt.IntVarOptional(&i, "int", 123)
		_, err := opt.Parse([]string{"--int"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if i != 123 {
			t.Errorf("Default value not set for 'int'")
		}
	})

	t.Run("Cast errors", func(t *testing.T) {
		opt := getoptions.New()
		opt.IntOptional("int", 0)
		_, err := opt.Parse([]string{"--int=hello"})
		if err == nil {
			t.Errorf("Int cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("Cast errors", func(t *testing.T) {
		opt := getoptions.New()
		opt.IntOptional("int", 0)
		_, err := opt.Parse([]string{"--int", "hello"})
		if err == nil {
			t.Errorf("Int cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("Cast errors", func(t *testing.T) {
		opt := getoptions.New()
		opt.Float64Optional("float", 0.0)
		_, err := opt.Parse([]string{"--float=hello"})
		if err == nil {
			t.Errorf("Float cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToFloat64, "float", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})
}

func TestGetOptBool(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false)
		_, err := opt.Parse([]string{"--flag"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Value("flag") != true {
			t.Errorf("Wrong value: %v != %v", opt.Value("flag"), true)
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("v", false)
		opt.Bool("V", false)
		_, err := opt.Parse([]string{"-v"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !opt.Called("v") {
			t.Errorf("v didn't have expected value %v", false)
		}
		if opt.Called("V") {
			t.Errorf("V didn't have expected value %v", true)
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("v", false)
		opt.Bool("V", false)
		_, err := opt.Parse([]string{"-V"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !opt.Called("V") {
			t.Errorf("V didn't have expected value %v", false)
		}
		if opt.Called("v") {
			t.Errorf("v didn't have expected value %v", true)
		}
	})
}

func TestCalled(t *testing.T) {
	opt := getoptions.New()
	opt.Bool("hello", false)
	opt.Bool("happy", false)
	opt.Bool("world", false)
	opt.String("string", "")
	opt.String("string2", "")
	opt.Int("int", 0)
	opt.Int("int2", 0)
	opt.Float64("float", 123.123)
	opt.Float64("float2", 0.0)
	_, err := opt.Parse([]string{"--hello", "--world", "--string2", "str", "--int2", "123", "--float2", "456.456"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !opt.Called("hello") {
		t.Errorf("hello didn't have expected value %v", false)
	}
	if opt.Called("happy") {
		t.Errorf("happy didn't have expected value %v", true)
	}
	if !opt.Called("world") {
		t.Errorf("world didn't have expected value %v", false)
	}
	if opt.Called("string") {
		t.Errorf("string didn't have expected value %v", true)
	}
	if !opt.Called("string2") {
		t.Errorf("string2 didn't have expected value %v", false)
	}
	if opt.Called("int") {
		t.Errorf("int didn't have expected value %v", true)
	}
	if !opt.Called("int2") {
		t.Errorf("int2 didn't have expected value %v", false)
	}
	if opt.Called("float") {
		t.Errorf("int didn't have expected value %v", true)
	}
	if !opt.Called("float2") {
		t.Errorf("float2 didn't have expected value %v", false)
	}
	if opt.Called("unknown") {
		t.Errorf("unknown didn't have expected value %v", false)
	}
}

func TestCalledAs(t *testing.T) {
	t.Run("flag", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f", "hello"))
		_, err := opt.Parse([]string{"--flag"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.CalledAs("flag") != "flag" {
			t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "flag")
		}
	})

	t.Run("hello", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f", "hello"))
		_, err := opt.Parse([]string{"--hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.CalledAs("flag") != "hello" {
			t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "hello")
		}
	})

	t.Run("abbreviation", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f", "hello"))
		_, err := opt.Parse([]string{"--h"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.CalledAs("flag") != "hello" {
			t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "hello")
		}
	})

	t.Run("empty", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f", "hello"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.CalledAs("flag") != "" {
			t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "")
		}
	})

	t.Run("wrong name", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f", "hello"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.CalledAs("x") != "" {
			t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("x"), "")
		}
	})

	t.Run("all aliases, last one wins", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringSlice("list", 1, 1, opt.Alias("array", "slice"))
		_, err := opt.Parse([]string{"--list=list", "--array=array", "--slice=slice"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.CalledAs("list") != "slice" {
			t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("list"), "slice")
		}
	})
}

func TestEndOfParsing(t *testing.T) {
	opt := getoptions.New()
	opt.Bool("hello", false)
	opt.Bool("world", false)
	remaining, err := opt.Parse([]string{"hola", "--hello", "--", "mundo", "--world"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if !reflect.DeepEqual(remaining, []string{"hola", "mundo", "--world"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"hola", "mundo", "--world"})
	}
}

func TestGetOptAliases(t *testing.T) {
	setup := func() *getoptions.GetOpt {
		opt := getoptions.New()
		opt.Bool("flag", false, opt.Alias("f", "h"))
		return opt
	}

	cases := []struct {
		opt    *getoptions.GetOpt
		option string
		input  []string
		value  bool
	}{
		{setup(),
			"flag",
			[]string{"--flag"},
			true,
		},
		{setup(),
			"flag",
			[]string{"-f"},
			true,
		},
		{setup(),
			"flag",
			[]string{"-h"},
			true,
		},
		// TODO: Add flag to allow for this.
		{setup(),
			"flag",
			[]string{"--fl"},
			true,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	t.Run("ambiguous", func(t *testing.T) {
		opt := getoptions.New()
		opt.Bool("flag", false)
		opt.Bool("fleg", false)
		_, err := opt.Parse([]string{"--fl"})
		if err == nil {
			t.Errorf("Ambiguous argument 'fl' didn't raise unknown option error")
		}
		expected := fmt.Sprintf(text.ErrorAmbiguousArgument, "--fl", []string{"flag", "fleg"})
		if err != nil && err.Error() != expected {
			t.Errorf("Error string didn't match. expected: '%s', got: '%s'", expected, err)
		}
	})

	t.Run("ensure there is no panic when alias matches the beginning of preexisting option", func(t *testing.T) {
		// Bug: Startup panic when alias matches the beginning of preexisting option
		// https://github.com/DavidGamba/go-getoptions/issues/1
		opt := getoptions.New()
		opt.Bool("fleg", false)
		opt.Bool("flag", false, opt.Alias("f"))
		_, err := opt.Parse([]string{"--f"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if opt.Called("fleg") {
			t.Errorf("fleg should not have been called")
		}
		if !opt.Called("flag") {
			t.Errorf("flag not called")
		}
	})

	t.Run("second alias used", func(t *testing.T) {
		opt := getoptions.New()
		opt.Int("flag", 0, opt.Alias("f", "h"))
		_, err := opt.Parse([]string{"--h"})
		if err == nil {
			t.Errorf("Int didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "h") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})
}

func TestGetOptString(t *testing.T) {
	setup := func() *getoptions.GetOpt {
		opt := getoptions.New()
		opt.String("string", "", opt.Alias("alias"))
		return opt
	}

	cases := []struct {
		opt    *getoptions.GetOpt
		option string
		input  []string
		value  string
	}{
		{setup(),
			"string",
			[]string{"--string=hello"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--string=hello", "world"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--string", "hello"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--alias", "hello"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "world"},
			"hello",
		},
		// String should only accept an option looking string as an argument when passed after =
		{setup(),
			"string",
			[]string{"--string=--hello", "world"},
			"--hello",
		},
		// TODO: Set up a flag to decide whether or not to err on this
		// To have the definition of string overridden. This should probably fail since it is most likely not what the user intends.
		{setup(),
			"string",
			[]string{"--string", "hello", "--string", "world"},
			"world",
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	t.Run("fail when arg not provided", func(t *testing.T) {
		opt := getoptions.New()
		opt.String("string", "")
		_, err := opt.Parse([]string{"--string", "--hello"})
		if err == nil {
			t.Errorf("Passing option where argument expected didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "string") {
			t.Errorf("Error string didn't match expected value")
		}
	})
}

func TestGetOptInt(t *testing.T) {
	setup := func() *getoptions.GetOpt {
		opt := getoptions.New()
		opt.Int("int", 0, opt.Alias("alias"))
		return opt
	}

	cases := []struct {
		opt    *getoptions.GetOpt
		option string
		input  []string
		value  int
	}{
		{setup(),
			"int",
			[]string{"--int=-123"},
			-123,
		},
		{setup(),
			"int",
			[]string{"--int=-123", "world"},
			-123,
		},
		{setup(),
			"int",
			[]string{"--int", "123"},
			123,
		},
		{setup(),
			"int",
			[]string{"--alias", "123"},
			123,
		},
		{setup(),
			"int",
			[]string{"--int", "123", "world"},
			123,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	t.Run("missing argument", func(t *testing.T) {
		opt := getoptions.New()
		opt.Int("int", 0)
		_, err := opt.Parse([]string{"--int"})
		if err == nil {
			t.Errorf("Int didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "int") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("cast error", func(t *testing.T) {
		opt := getoptions.New()
		opt.Int("int", 0)
		_, err := opt.Parse([]string{"--int=hello"})
		if err == nil {
			t.Errorf("Int cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("cast error", func(t *testing.T) {
		opt := getoptions.New()
		opt.Int("int", 0)
		_, err := opt.Parse([]string{"--int", "hello"})
		if err == nil {
			t.Errorf("Int cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		opt := getoptions.New()
		opt.Int("int", 0)
		_, err := opt.Parse([]string{"--int", "-123"})
		if err == nil {
			t.Errorf("Passing option where argument expected didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "int") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})
}

func TestGetOptFloat64(t *testing.T) {
	setup := func() *getoptions.GetOpt {
		opt := getoptions.New()
		opt.Float64("float", 0, opt.Alias("alias"))
		return opt
	}

	cases := []struct {
		opt    *getoptions.GetOpt
		option string
		input  []string
		value  float64
	}{
		{setup(),
			"float",
			[]string{"--float=-1.23"},
			-1.23,
		},
		{setup(),
			"float",
			[]string{"--float=-1.23", "world"},
			-1.23,
		},
		{setup(),
			"float",
			[]string{"--float", "1.23"},
			1.23,
		},
		{setup(),
			"float",
			[]string{"--alias", "1.23"},
			1.23,
		},
		{setup(),
			"float",
			[]string{"--float", "1.23", "world"},
			1.23,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	t.Run("Missing Argument errors", func(t *testing.T) {
		opt := getoptions.New()
		opt.Float64("float", 0)
		_, err := opt.Parse([]string{"--float"})
		if err == nil {
			t.Errorf("Float64 didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "float") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("Cast errors", func(t *testing.T) {
		opt := getoptions.New()
		opt.Float64("float", 0)
		_, err := opt.Parse([]string{"--float=hello"})
		if err == nil {
			t.Errorf("Float cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToFloat64, "float", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("Cast errors", func(t *testing.T) {
		opt := getoptions.New()
		opt.Float64("float", 0)
		_, err := opt.Parse([]string{"--float", "hello"})
		if err == nil {
			t.Errorf("Int cast didn't raise errors")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToFloat64, "float", "hello") {
			t.Errorf("Error string didn't match expected value '%s'", err)
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		opt := getoptions.New()
		opt.Float64("float", 0)
		_, err := opt.Parse([]string{"--float", "-123"})
		if err == nil {
			t.Errorf("Passing option where argument expected didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "float") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})
}

// TODO: Allow passing : as the map divider
func TestGetOptStringMap(t *testing.T) {
	setup := func() *getoptions.GetOpt {
		opt := getoptions.New()
		opt.StringMap("string", 1, 3, opt.Alias("alias"))
		opt.String("opt", "")
		return opt
	}

	cases := []struct {
		name   string
		opt    *getoptions.GetOpt
		option string
		input  []string
		value  map[string]string
	}{
		{"arg inline with option",
			setup(),
			"string",
			[]string{"--string=hello=world"},
			map[string]string{"hello": "world"},
		},
		{"arg inline with alias",
			setup(),
			"string",
			[]string{"--alias=hello=world"},
			map[string]string{"hello": "world"},
		},
		{"inline arg with following non key value text",
			setup(),
			"string",
			[]string{"--string=hello=happy", "world"},
			map[string]string{"hello": "happy"},
		},
		{"arg with following non key value text",
			setup(),
			"string",
			[]string{"--string", "hello=happy", "world"},
			map[string]string{"hello": "happy"},
		},
		{"arg with following string option",
			setup(),
			"string",
			[]string{"--string", "hello=world", "--opt", "happy"},
			map[string]string{"hello": "world"},
		},
		{"inline arg with leading dashes",
			setup(),
			"string",
			[]string{"--string=--hello=happy", "world"},
			map[string]string{"--hello": "happy"},
		},
		{"multiple calls",
			setup(),
			"string",
			[]string{"--string", "hello=world", "--string", "key=value", "--string", "key2=value2"},
			map[string]string{"hello": "world", "key": "value", "key2": "value2"},
		},
		{"multiple calls using maximum",
			setup(),
			"string",
			[]string{"--string", "hello=world", "key=value", "key2=value2"},
			map[string]string{"hello": "world", "key": "value", "key2": "value2"},
		},
		{"2 args",
			setup(),
			"string",
			[]string{"--string", "hello=happy", "happy=world"},
			map[string]string{"hello": "happy", "happy": "world"},
		},
		{"inline arg plus extra arg",
			setup(),
			"string",
			[]string{"--string=--hello=happy", "happy=world"},
			map[string]string{"--hello": "happy", "happy": "world"},
		},
		{"validate case",
			setup(),
			"string",
			[]string{"--string", "key=value", "Key=value1", "kEy=value2"},
			map[string]string{"key": "value", "Key": "value1", "kEy": "value2"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.opt.Parse(c.input)
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
			}
			if !reflect.DeepEqual(c.opt.Value(c.option), c.value) {
				t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
			}
		})
	}

	t.Run("arg not key value", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringMap("string", 1, 3)
		_, err := opt.Parse([]string{"--string", "hello"})
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentIsNotKeyValue, "string") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})

	t.Run("arg not key value", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringMap("string", 1, 3)
		_, err := opt.Parse([]string{"--string=hello"})
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentIsNotKeyValue, "string") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})

	t.Run("no arg", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringMap("string", 1, 3)
		_, err := opt.Parse([]string{"--string"})
		if err == nil {
			t.Errorf("Missing argument for option 'string' didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})

	t.Run("no arg", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringMap("string", 1, 3)
		_, err := opt.Parse([]string{"--string", "--hello=happy", "world"})
		if err == nil {
			t.Errorf("Missing argument for option 'string' didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "string") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})

	t.Run("no arg, wrong min", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringMap("string", 2, 3)
		_, err := opt.Parse([]string{"--string", "hello=world"})
		if err == nil {
			t.Errorf("Passing less than min didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})

	t.Run("arg wrong min", func(t *testing.T) {
		opt := getoptions.New()
		opt.StringMap("string", 2, 3)
		_, err := opt.Parse([]string{"--string", "hello=world", "happy"})
		if err == nil {
			t.Errorf("Passing less than min didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentIsNotKeyValue, "string") {
			t.Errorf("Error string didn't match expected value: %s", err.Error())
		}
	})

	t.Run("multiple args", func(t *testing.T) {
		opt := getoptions.New()
		sm := opt.StringMap("string", 1, 3)
		_, err := opt.Parse([]string{"--string", "hello=world", "key=value", "key2=value2"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm) {
			t.Errorf("Wrong value: %v != %v", map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm)
		}
		if sm["hello"] != "world" || sm["key"] != "value" || sm["key2"] != "value2" {
			t.Errorf("Wrong value: %v", sm)
		}
	})

	t.Run("multiple args", func(t *testing.T) {
		var sm map[string]string
		opt := getoptions.New()
		opt.StringMapVar(&sm, "string", 1, 3)
		_, err := opt.Parse([]string{"--string", "hello=world", "key=value", "key2=value2"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm) {
			t.Errorf("Wrong value: %v != %v", map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm)
		}
		if sm["hello"] != "world" || sm["key"] != "value" || sm["key2"] != "value2" {
			t.Errorf("Wrong value: %v", sm)
		}
	})

	t.Run("ignore case", func(t *testing.T) {
		opt := getoptions.New()
		opt.SetMapKeysToLower()
		sm := opt.StringMap("string", 1, 3)
		_, err := opt.Parse([]string{"--string", "Key1=value1", "kEy2=value2", "keY3=value3"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}, sm) {
			t.Errorf("Wrong value: %v != %v", map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}, sm)
		}
	})
}

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

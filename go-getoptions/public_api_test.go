package getoptions_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/DavidGamba/go-getoptions/go-getoptions"
	"github.com/DavidGamba/go-getoptions/option"
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
}

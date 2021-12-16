package getoptions

import (
	"os"
	"strings"

	"github.com/DavidGamba/go-getoptions/option"
)

// ModifyFn - Function signature for functions that modify an option.
type ModifyFn func(parent *GetOpt, option *option.Option)

// ModifyFn has to include the parent information because We want alias to be a
// global option. That is, that the user can call the top level opt.Alias from
// an option that belongs to a command or a subcommand. The problem with that
// is that if the ModifyFn signature doesn't provide information about the
// current parent we loose information about where the alias belongs to.
//
// The other complication with aliases becomes validation. Ideally, due to the
// tree nature of the command/option definition, you might want to define the
// same option with the same alias for two commands and they could do different
// things. That means that, without parent information, to write validation for
// aliases one has to navigate all leafs of the tree and validate that
// duplicates don't exist and limit functionality.

// Alias - Adds aliases to an option.
func (gopt *GetOpt) Alias(alias ...string) ModifyFn {
	// We want alias to be a global option. That is, that the user can call
	// the top level opt.Alias from an option that belongs to a command or a subcommand.
	return func(parent *GetOpt, opt *option.Option) {
		opt.SetAlias(alias...)
		for _, a := range alias {
			parent.programTree.AddChildOption(a, opt)
		}
	}
}

// Description - Add a description to an option for use in automated help.
func (gopt *GetOpt) Description(msg string) ModifyFn {
	return func(parent *GetOpt, opt *option.Option) {
		opt.Description = msg
	}
}

func (gopt *GetOpt) Required(msg ...string) ModifyFn {
	var errTxt string
	if len(msg) >= 1 {
		errTxt = msg[0]
	}
	return func(parent *GetOpt, opt *option.Option) {
		opt.SetRequired(errTxt)
	}
}

// GetEnv - Will read an environment variable if set.
// Precedence higher to lower: CLI option, environment variable, option default.
//
// Currently, only `opt.Bool`, `opt.BoolVar`, `opt.String`, and `opt.StringVar` are supported.
//
// When an environment variable that matches the variable from opt.GetEnv is
// set, opt.GetEnv will set opt.Called(name) to true and will set
// opt.CalledAs(name) to the name of the environment variable used.
// In other words, when an option is required (opt.Required is set) opt.GetEnv
// satisfies that requirement.
//
// When using `opt.GetEnv` with `opt.Bool` or `opt.BoolVar`, only the words
// "true" or "false" are valid.  They can be provided in any casing, for
// example: "true", "True" or "TRUE".
//
// NOTE: Non supported option types behave with a No-Op when `opt.GetEnv` is defined.
func (gopt *GetOpt) GetEnv(name string) ModifyFn {
	return func(parent *GetOpt, opt *option.Option) {
		opt.SetEnvVar(name)
		value := os.Getenv(name)
		if value != "" {
			switch opt.OptType {
			case option.BoolType:
				v := strings.ToLower(value)
				if v == "true" || v == "false" {
					opt.Save(v)
					opt.SetCalled(name)
				}
			case option.StringType, option.IntType, option.Float64Type:
				opt.Save(value)
				opt.SetCalled(name)
			}
		}
	}
}

func (gopt *GetOpt) ArgName(name string) ModifyFn {
	return func(parent *GetOpt, opt *option.Option) {
		opt.SetHelpArgName(name)
	}
}

func (gopt *GetOpt) ValidValues(values ...string) ModifyFn {
	return func(parent *GetOpt, opt *option.Option) {
		opt.ValidValues = append(opt.ValidValues, values...)
		opt.SuggestedValues = opt.ValidValues
	}
}

// Called - Indicates if the option was passed on the command line.
// If the `name` is an option that wasn't declared it will return false.
func (gopt *GetOpt) Called(name string) bool {
	if name == "" {
		// Don't panic at this point since the user can only reproduce this by
		// executing every branch of their code.
		return false
	}
	if v, ok := gopt.programTree.ChildOptions[name]; ok {
		return v.Called
	}
	return false
}

// CalledAs - Returns the alias used to call the option.
// Empty string otherwise.
//
// If the `name` is an option that wasn't declared it will return an empty string.
//
// For options that can be called multiple times, the last alias used is returned.
func (gopt *GetOpt) CalledAs(name string) string {
	if name == "" {
		// Don't panic at this point since the user can only reproduce this by
		// executing every branch of their code.
		return ""
	}
	if v, ok := gopt.programTree.ChildOptions[name]; ok {
		return v.UsedAlias
	}
	return ""
}

func (gopt *GetOpt) Bool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.BoolVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.BoolType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) String(name, def string, fns ...ModifyFn) *string {
	gopt.StringVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) StringVar(p *string, name, def string, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.StringType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) StringOptional(name, def string, fns ...ModifyFn) *string {
	gopt.StringVarOptional(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) StringVarOptional(p *string, name, def string, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.StringOptionalType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) StringSlice(name string, min, max int, fns ...ModifyFn) *[]string {
	s := []string{}
	gopt.StringSliceVar(&s, name, min, max, fns...)
	return &s
}

func (gopt *GetOpt) StringSliceVar(p *[]string, name string, min, max int, fns ...ModifyFn) {
	n := option.New(name, option.StringRepeatType, p)
	n.MinArgs = min
	n.MaxArgs = max
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
	n.Synopsis()
}

func (gopt *GetOpt) Int(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) IntVar(p *int, name string, def int, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.IntType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) IntOptional(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVarOptional(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.IntOptionalType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) IntSlice(name string, min, max int, fns ...ModifyFn) *[]int {
	s := []int{}
	gopt.IntSliceVar(&s, name, min, max, fns...)
	return &s
}

func (gopt *GetOpt) IntSliceVar(p *[]int, name string, min, max int, fns ...ModifyFn) {
	n := option.New(name, option.IntRepeatType, p)
	n.MinArgs = min
	n.MaxArgs = max
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
	n.Synopsis()
}

// Increment - When called multiple times it increments the int counter defined by this option.
func (gopt *GetOpt) Increment(name string, def int, fns ...ModifyFn) *int {
	gopt.IncrementVar(&def, name, def, fns...)
	return &def
}

// IncrementVar - When called multiple times it increments the provided int.
func (gopt *GetOpt) IncrementVar(p *int, name string, def int, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.Increment, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) Float64(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64Var(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.Float64Type, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) Float64Optional(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64VarOptional(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) Float64VarOptional(p *float64, name string, def float64, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.Float64OptionalType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// // TODO: Before publishing this complete the opt save part for it
// func (gopt *GetOpt) Float64Slice(name string, min, max int, fns ...ModifyFn) *[]float64 {
// 	s := []float64{}
// 	gopt.Float64SliceVar(&s, name, min, max, fns...)
// 	return &s
// }
//
// // TODO: Before publishing this complete the opt save part for it
// func (gopt *GetOpt) Float64SliceVar(p *[]float64, name string, min, max int, fns ...ModifyFn) {
// 	n := option.New(name, option.Float64RepeatType, p)
// 	n.MinArgs = min
// 	n.MaxArgs = max
// 	gopt.programTree.AddChildOption(name, n)
// n.Synopsis()
// }

func (gopt *GetOpt) StringMap(name string, min, max int, fns ...ModifyFn) map[string]string {
	m := map[string]string{}
	gopt.StringMapVar(&m, name, min, max, fns...)
	return m
}

func (gopt *GetOpt) StringMapVar(m *map[string]string, name string, min, max int, fns ...ModifyFn) {
	// check that the map has been initialized
	if *m == nil {
		*m = make(map[string]string)
	}
	n := option.New(name, option.StringMapType, m)
	n.MinArgs = min
	n.MaxArgs = max
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
	n.Synopsis()
}

// SetMapKeysToLower - StringMap keys captured from StringMap are lower case.
// For example:
//
//     command --opt key=value
//
// And:
//
//     command --opt KEY=value
//
// Would both return `map[string]string{"key":"value"}`.
func (gopt *GetOpt) SetMapKeysToLower() *GetOpt {
	gopt.programTree.mapKeysToLower = true
	return gopt
}

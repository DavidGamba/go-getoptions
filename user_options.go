package getoptions

import (
	"os"
	"strings"

	"github.com/DavidGamba/go-getoptions/internal/option"
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

// Required - Automatically return an error if the option is not called.
// Optionally provide a custom error message, a default error message will be used otherwise.
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
			case option.StringType,
				option.IntType,
				option.Float64Type,
				option.StringOptionalType,
				option.IntOptionalType,
				option.Float64OptionalType:

				opt.Save(value)
				opt.SetCalled(name)
			}
		}
	}
}

// ArgName - Add an argument name to an option for use in automated help.
// For example, by default a string option will have a default synopsis as follows:
//
//     --host <string>
//
// If ArgName("hostname") is used, the synopsis will read:
//
//     --host <hostname>
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

// Value - Returns the value of the given option.
//
// Type assertions are required in cases where the compiler can't determine the type by context.
// For example: `opt.Value("flag").(bool)`.
func (gopt *GetOpt) Value(name string) interface{} {
	if v, ok := gopt.programTree.ChildOptions[name]; ok {
		return v.Value()
	}
	return nil
}

// Bool - define a `bool` option and its aliases.
// It returns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) Bool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.BoolVar(&def, name, def, fns...)
	return &def
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.BoolType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) String(name, def string, fns ...ModifyFn) *string {
	gopt.StringVar(&def, name, def, fns...)
	return &def
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) StringVar(p *string, name, def string, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.StringType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringOptional(name, def string, fns ...ModifyFn) *string {
	gopt.StringVarOptional(&def, name, def, fns...)
	return &def
}

// StringVarOptional - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// StringVarOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringVarOptional(p *string, name, def string, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.StringOptionalType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// StringSlice - define a `[]string` option and its aliases.
//
// StringSlice will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
//
// Additionally, StringSlice will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]string{"1", "2", "3"}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
func (gopt *GetOpt) StringSlice(name string, min, max int, fns ...ModifyFn) *[]string {
	s := []string{}
	gopt.StringSliceVar(&s, name, min, max, fns...)
	return &s
}

// StringSliceVar - define a `[]string` option and its aliases.
//
// StringSliceVar will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
//
// Additionally, StringSliceVar will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]string{"1", "2", "3"}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
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

// Int - define an `int` option and its aliases.
func (gopt *GetOpt) Int(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVar(&def, name, def, fns...)
	return &def
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) IntVar(p *int, name string, def int, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.IntType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntOptional(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVarOptional(&def, name, def, fns...)
	return &def
}

// IntVarOptional - define a `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.IntOptionalType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// IntSlice - define a `[]int` option and its aliases.
//
// IntSlice will accept multiple calls to the same option and append them
// to the `[]int`.
// For example, when called with `--intRpt 1 --intRpt 2`, the value is `[]int{1, 2}`.
//
// Additionally, IntSlice will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]int{1, 2, 3}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
//
// Finally, positive integer ranges are allowed.
// For example, Instead of writing: `csv --columns 1 2 3` or
// `csv --columns 1 --columns 2 --columns 3`
// The input could be: `csv --columns 1..3`.
func (gopt *GetOpt) IntSlice(name string, min, max int, fns ...ModifyFn) *[]int {
	s := []int{}
	gopt.IntSliceVar(&s, name, min, max, fns...)
	return &s
}

// IntSliceVar - define a `[]int` option and its aliases.
//
// IntSliceVar will accept multiple calls to the same option and append them
// to the `[]int`.
// For example, when called with `--intRpt 1 --intRpt 2`, the value is `[]int{1, 2}`.
//
// Additionally, IntSliceVar will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]int{1, 2, 3}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
//
// Finally, positive integer ranges are allowed.
// For example, Instead of writing: `csv --columns 1 2 3` or
// `csv --columns 1 --columns 2 --columns 3`
// The input could be: `csv --columns 1..3`.
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
	n := option.New(name, option.IncrementType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// Float64 - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64Var(&def, name, def, fns...)
	return &def
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.Float64Type, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

// Float64Optional - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64Optional(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64VarOptional(&def, name, def, fns...)
	return &def
}

// Float64VarOptional - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64VarOptional(p *float64, name string, def float64, fns ...ModifyFn) {
	*p = def
	n := option.New(name, option.Float64OptionalType, p)
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

func (gopt *GetOpt) Float64Slice(name string, min, max int, fns ...ModifyFn) *[]float64 {
	s := []float64{}
	gopt.Float64SliceVar(&s, name, min, max, fns...)
	return &s
}

func (gopt *GetOpt) Float64SliceVar(p *[]float64, name string, min, max int, fns ...ModifyFn) {
	n := option.New(name, option.Float64RepeatType, p)
	n.MinArgs = min
	n.MaxArgs = max
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
	n.Synopsis()
}

// StringMap - define a `map[string]string` option and its aliases.
//
// StringMap will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
//
// Additionally, StringMap will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strMap k=v k2=v2 k3=v3`,
// the value is `map[string]string{"k":"v", "k2": "v2", "k3": "v3"}`.
// It could also be called with `--strMap k=v --strMap k2=v2 --strMap k3=v3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strMap k=v k2=v2 --strMap k3=v3`
func (gopt *GetOpt) StringMap(name string, min, max int, fns ...ModifyFn) map[string]string {
	m := map[string]string{}
	gopt.StringMapVar(&m, name, min, max, fns...)
	return m
}

// StringMapVar - define a `map[string]string` option and its aliases.
//
// StringMapVar will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
//
// Additionally, StringMapVar will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strMap k=v k2=v2 k3=v3`,
// the value is `map[string]string{"k":"v", "k2": "v2", "k3": "v3"}`.
// It could also be called with `--strMap k=v --strMap k2=v2 --strMap k3=v3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strMap k=v k2=v2 --strMap k3=v3`
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

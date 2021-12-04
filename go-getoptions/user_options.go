package getoptions

import (
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

func (gopt *GetOpt) GetEnv(alias ...string) ModifyFn {
	return func(parent *GetOpt, opt *option.Option) {
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
	gopt.programTree.AddChildOption(name, n)
	for _, fn := range fns {
		fn(gopt, n)
	}
}

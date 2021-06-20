package getoptions

import (
	"github.com/DavidGamba/go-getoptions/option"
)

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

func (gopt *GetOpt) Bool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.BoolVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
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
	n := option.New(name, option.StringType, p)
	gopt.programTree.AddChildOption(name, n)
}

func (gopt *GetOpt) StringOptional(name, def string, fns ...ModifyFn) *string {
	gopt.StringVarOptional(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) StringVarOptional(p *string, name, def string, fns ...ModifyFn) {
	n := option.New(name, option.StringType, p)
	gopt.programTree.AddChildOption(name, n)
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
}

func (gopt *GetOpt) Int(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) IntVar(p *int, name string, def int, fns ...ModifyFn) {
	n := option.New(name, option.IntType, p)
	gopt.programTree.AddChildOption(name, n)
}

func (gopt *GetOpt) IntOptional(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVarOptional(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, fns ...ModifyFn) {
	n := option.New(name, option.IntType, p)
	gopt.programTree.AddChildOption(name, n)
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
}

func (gopt *GetOpt) Float64(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64Var(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, fns ...ModifyFn) {
	n := option.New(name, option.Float64Type, p)
	gopt.programTree.AddChildOption(name, n)
}

func (gopt *GetOpt) Float64Optional(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64VarOptional(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) Float64VarOptional(p *float64, name string, def float64, fns ...ModifyFn) {
	n := option.New(name, option.Float64Type, p)
	gopt.programTree.AddChildOption(name, n)
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
}

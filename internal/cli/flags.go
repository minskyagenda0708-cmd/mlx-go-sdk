package cli

import (
	"flag"
	"strconv"
)

// optionalBoolFlag tracks whether a bool flag was explicitly provided.
type optionalBoolFlag struct {
	value bool
	set   bool
}

func newOptionalBoolFlag(fs *flag.FlagSet, name, usage string) *optionalBoolFlag {
	f := &optionalBoolFlag{}
	fs.Var(f, name, usage)
	return f
}

func (f *optionalBoolFlag) String() string {
	if f == nil {
		return ""
	}
	return strconv.FormatBool(f.value)
}

func (f *optionalBoolFlag) Set(raw string) error {
	if raw == "" {
		f.value = true
		f.set = true
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return err
	}
	f.value = value
	f.set = true
	return nil
}

func (f *optionalBoolFlag) IsBoolFlag() bool {
	return true
}

func (f *optionalBoolFlag) ValueOr(fallback bool) bool {
	if f == nil || !f.set {
		return fallback
	}
	return f.value
}

func (f *optionalBoolFlag) BoolPtr() *bool {
	if f == nil || !f.set {
		return nil
	}
	value := f.value
	return &value
}

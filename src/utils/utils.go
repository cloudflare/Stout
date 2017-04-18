package utils

import (
	"errors"
	"flag"
	"fmt"

	"github.com/urfave/cli"
)

type titleFlag struct {
	Title  string
	Hidden bool
}

func (t titleFlag) Apply(*flag.FlagSet) {}
func (t titleFlag) String() string      { return "\n" + t.Title }
func (t titleFlag) GetName() string     { return "" }

func TitleFlag(title string) cli.Flag {
	return titleFlag{Title: title}
}

func ErrorMerge(str string, err error) error {
	return errors.New(str + " " + err.Error())
}

func PanicsToErrors(debugMode bool, f func() error) (err error) {
	if !debugMode {
		defer func() {
			if r := recover(); r != nil {
				var ok bool
				err, ok = r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
			}
		}()
	}

	return f()
}

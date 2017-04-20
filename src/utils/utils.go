package utils

import (
	"errors"
	"fmt"
)

// Catch errors and panic if there is an error
func PanicIf(err error) {
	if err != nil {
		panic(err)
	}
}

// Catch errors and panic if there is an error
func Must(val interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}
	return val
}
func MustString(val string, err error) string {
	PanicIf(err)
	return val
}
func MustInt(val int, err error) int {
	PanicIf(err)
	return val
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

// Decode INI files with a syntax similar to JSON decoding
package ini

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

type Unmatched struct {
	lineNum int
	line    string
}

type IniError struct {
	lineNum int
	line    string
	error   string
}

// decodeState represents the state while decoding a INI value.
type decodeState struct {
	lineNum    int
	line       string
	scanner    *bufio.Scanner
	savedError error
	unmatched  []Unmatched
}

type property struct {
	tag      string
	value    reflect.Value
	children propertyMap
	isArray  bool
	//array         []interface{}
	isInitialized bool
}

type propertyMap map[string]property

//------------------------------------------------------------------

// NewStack returns a new stack.
func NewPropMapStack() *PropMapStack {
	return &PropMapStack{}
}

// Stack is a basic LIFO stack that resizes as needed.
type PropMapStack struct {
	items []propertyMap
	count int
}

// Push adds an iterm to the top of the stack
func (s *PropMapStack) Push(item propertyMap) {
	s.items = append(s.items[:s.count], item)
	s.count++
}

// Pop removes the top item (LIFO) from the stack
func (s *PropMapStack) Pop() propertyMap {
	if s.count == 0 {
		return nil
	}

	s.count--
	return s.items[s.count]
}

// Peek returns item at top of stack without removing it
func (s *PropMapStack) Peek() propertyMap {
	if s.count == 0 {
		return nil
	}

	return s.items[s.count-1]
}

// Empty returns true when stack is empty, false otherwise
func (s *PropMapStack) Empty() bool {
	return s.count == 0
}

// Size returns the number of items in the stack
func (s *PropMapStack) Size() int {
	return s.count
}

/*
 * Unmarshal parses the INI-encoded data and stores the result
 * in the value pointed to by v.
 */
func Unmarshal(data []byte, v interface{}) error {
	var d decodeState
	d.init(data)
	return d.unmarshal(v)
}

/*
 * String interfacer for Unmatched
 */
func (u Unmatched) String() string {
	return fmt.Sprintf("%d %s", u.lineNum, u.line)
}

/*
 * Conform to Error Interfacer
 */
func (e *IniError) Error() string {
	if e.lineNum > 0 {
		return fmt.Sprintf("%s on line %d: \"%s\"", e.error, e.lineNum, e.line)
	} else {
		return e.error
	}
}

/*
 * Stringer interface for property
 */
func (p property) String() string {
	return fmt.Sprintf("<property %s, isArray:%t>", p.tag, p.isArray)
}

/*
 * Convenience function to prep for decoding byte array.
 */
func (d *decodeState) init(data []byte) *decodeState {

	d.lineNum = 0
	d.line = ""
	d.scanner = bufio.NewScanner(bytes.NewReader(data))
	d.savedError = nil

	return d
}

/*
 * saveError saves the first err it is called with,
 * for reporting at the end of the unmarshal.
 */
func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = err
	}
}

/*
 * Recursive function to map data types in the describing structs
 * to string markers (tags) in the INI file.
 */
func (d *decodeState) generateMap(m propertyMap, v reflect.Value) {

	if v.Type().Kind() == reflect.Ptr {
		d.generateMap(m, v.Elem())
	} else if v.Kind() == reflect.Struct {
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {

			sf := typ.Field(i)
			f := v.Field(i)
			kind := f.Type().Kind()

			tag := sf.Tag.Get("ini")
			if len(tag) == 0 {
				tag = sf.Name
			}
			tag = strings.TrimSpace(strings.ToLower(tag))

			st := property{tag, f, make(propertyMap), kind == reflect.Slice, true}

			// some structures are just for organizing data
			if tag != "-" {
				m[tag] = st
			}

			if kind == reflect.Struct {
				if tag == "-" {
					d.generateMap(m, f)
				} else {
					// little namespacing here so property names can
					// be the same under different sections
					//fmt.Printf("Struct tag: %s, type: %s\n", tag, f.Type())
					d.generateMap(st.children, f)
				}
			} else if kind == reflect.Slice {
				d.generateMap(st.children, reflect.New(f.Type().Elem()))
			}
		}
	}
}

/*
 * Iterates line-by-line through INI file setting values into a struct.
 */
func (d *decodeState) unmarshal(x interface{}) error {

	var topMap propertyMap
	topMap = make(propertyMap)

	d.generateMap(topMap, reflect.ValueOf(x))

	propStack := NewPropMapStack()
	propStack.Push(topMap)

	// for every line in file
	for d.scanner.Scan() {

		if d.savedError != nil {
			break // breaks on first error
		}

		d.line = d.scanner.Text()
		d.lineNum++

		line := strings.TrimSpace(d.line)

		if len(line) < 1 || line[0] == ';' || line[0] == '#' {
			continue // skip comments
		}

		// Two types of lines:
		//   1. NAME=VALUE   (at least one equal sign - breaks on first)
		//   2. [HEADER]     (no equals sign, square brackets NOT required)
		matches := strings.SplitN(line, "=", 2)
		matched := false
		pn := ""
		pv := ""

		if len(matches) == 2 {
			// NAME=VALUE
			pn = strings.ToLower(strings.TrimSpace(matches[0]))
			pv = strings.TrimSpace(matches[1])
			prop := propStack.Peek()[pn]

			if prop.isInitialized {
				if prop.isArray {
					value := reflect.New(prop.value.Type().Elem())
					d.setValue(reflect.Indirect(value), pv)
					appendValue(prop.value, value)

				} else {
					d.setValue(prop.value, pv)
				}

				matched = true
			}

			// What if property is umatched - keep popping the stack
			// until a potential map is found or stay within current section?
			// Think answer is pop.
			// NOPE
			// Section could have unmatched name=value if user doesn't
			// care about certain values - only stack crawling happens
			// during numMatches==1 time?
			// This means if there is > 1 section, there needs to be
			// section breaks for everything

		} else {
			// [Header] section
			pn = strings.ToLower(strings.TrimSpace(matches[0]))

			for propStack.Size() > 0 {
				prop := propStack.Peek()[pn]
				if prop.isInitialized {
					propStack.Push(prop.children)
					matched = true
					break
				} else if propStack.Size() > 1 {
					_ = propStack.Pop()
				} else {
					break
				}
			}
		}

		if !matched {
			d.unmatched = append(d.unmatched, Unmatched{d.lineNum, d.line})
		}
	}

	return d.savedError
}

func (d *decodeState) unmarshal2(x interface{}) error {

	var sectionMap propertyMap = make(propertyMap)
	var tempMap propertyMap = make(propertyMap)

	var section, nextSection property
	var inSection, nextHasSection bool = false, false
	var tempValue reflect.Value // "temp" is for filling in array of structs
	var numTempValue int

	d.generateMap(sectionMap, reflect.ValueOf(x))

	for d.scanner.Scan() {
		if d.savedError != nil {
			break
		}

		d.line = d.scanner.Text()
		d.lineNum++

		//fmt.Printf("%03d: %s\n", d.lineNum, d.line)

		line := strings.ToLower(strings.TrimSpace(d.line))

		if len(line) < 1 || line[0] == ';' || line[0] == '#' {
			continue // skip comments
		}

		// [Sections] could appear at any time (square brackets not required)
		// When in a section, also look in children map
		nextSection, nextHasSection = sectionMap[line]
		if nextHasSection {
			if numTempValue > 0 && section.isArray {
				appendValue(section.value, tempValue)
			}

			section = nextSection
			inSection = true

			if section.isArray {
				tempValue = reflect.New(section.value.Type().Elem())
				d.generateMap(tempMap, tempValue)
			}

			numTempValue = 0
			continue
		}

		// unrecognized section - exit out of current section
		if line[0] == '[' && line[len(line)-1] == ']' {
			inSection = false
			continue
		}

		matches := strings.SplitN(d.line, "=", 2)
		matched := false

		// potential property=value
		if len(matches) == 2 {
			n := strings.ToLower(strings.TrimSpace(matches[0]))
			s := strings.TrimSpace(matches[1])

			if inSection {
				// child property, within a section
				childProperty, hasProp := section.children[n]

				if hasProp {
					if section.isArray {
						tempProperty := tempMap[n]
						numTempValue++
						d.setValue(tempProperty.value, s)
					} else {
						d.setValue(childProperty.value, s)
					}

					matched = true
				}
			}

			if !matched {
				// top level property
				topLevelProperty, hasProp := sectionMap[n]
				if hasProp {
					// just encountered a top level property - switch out of section mode
					inSection = false
					matched = true
					d.setValue(topLevelProperty.value, s)
				}
			}
		}

		if !matched {
			d.unmatched = append(d.unmatched, Unmatched{d.lineNum, d.line})
		}
	}

	if numTempValue > 0 {
		appendValue(section.value, tempValue)
	}

	return d.savedError
}

func appendValue(arr, val reflect.Value) {
	arr.Set(reflect.Append(arr, reflect.Indirect(val)))
}

// Set Value with given string
func (d *decodeState) setValue(v reflect.Value, s string) {
	//fmt.Printf("SET(kind:%s, %s)\n", v.Kind(), s)

	switch v.Kind() {

	case reflect.String:
		v.SetString(s)

	case reflect.Bool:
		v.SetBool(boolValue(s))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v.OverflowInt(n) {
			d.saveError(&IniError{d.lineNum, d.line, "Invalid int"})
			return
		}
		v.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil || v.OverflowUint(n) {
			d.saveError(&IniError{d.lineNum, d.line, "Invalid uint"})
			return
		}
		v.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil || v.OverflowFloat(n) {
			d.saveError(&IniError{d.lineNum, d.line, "Invalid float"})
			return
		}
		v.SetFloat(n)

	case reflect.Slice:
		d.sliceValue(v, s)

	default:
		d.saveError(&IniError{d.lineNum, d.line, fmt.Sprintf("Can't set value of type %s", v.Kind())})
	}

}

func (d *decodeState) sliceValue(v reflect.Value, s string) {
	//fmt.Printf(":SLICE(%s, %s)\n", v.Kind(), s)

	switch v.Type().Elem().Kind() {

	case reflect.String:
		v.Set(reflect.Append(v, reflect.ValueOf(s)))

	case reflect.Bool:
		v.Set(reflect.Append(v, reflect.ValueOf(boolValue(s))))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Hardcoding of []int temporarily
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			d.saveError(&IniError{d.lineNum, d.line, "Invalid int"})
			return
		}

		n1 := reflect.ValueOf(n)
		n2 := n1.Convert(v.Type().Elem())

		v.Set(reflect.Append(v, n2))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			d.saveError(&IniError{d.lineNum, d.line, "Invalid uint"})
			return
		}

		n1 := reflect.ValueOf(n)
		n2 := n1.Convert(v.Type().Elem())

		v.Set(reflect.Append(v, n2))

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			d.saveError(&IniError{d.lineNum, d.line, "Invalid float"})
			return
		}

		n1 := reflect.ValueOf(n)
		n2 := n1.Convert(v.Type().Elem())

		v.Set(reflect.Append(v, n2))

	default:
		d.saveError(&IniError{d.lineNum, d.line, fmt.Sprintf("Can't set value in array of type %s",
			v.Type().Elem().Kind())})
	}

}

// Returns true for truthy values like t/true/y/yes/1, false otherwise
func boolValue(s string) bool {
	v := false
	switch strings.ToLower(s) {
	case "t", "true", "y", "yes", "1":
		v = true
	}

	return v
}

// A Decoder reads and decodes INI object from an input stream.
type Decoder struct {
	r io.Reader
	d decodeState
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the INI file and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about
// the conversion of an INI into a Go value.
func (dec *Decoder) Decode(v interface{}) error {

	buf, readErr := ioutil.ReadAll(dec.r)
	if readErr != nil {
		return readErr
	}
	// Don't save err from unmarshal into dec.err:
	// the connection is still usable since we read a complete JSON
	// object from it before the error happened.
	dec.d.init(buf)
	err := dec.d.unmarshal(v)

	return err
}

// UnparsedLines returns an array of strings where each string is an
// unparsed line from the file.
func (dec *Decoder) Unmatched() []Unmatched {
	return dec.d.unmatched
}

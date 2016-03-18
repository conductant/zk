package flag

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Allocate a flagset, bind it to val and return the flag set.
func GetFlagSet(name string, val interface{}) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.PanicOnError)
	RegisterFlags(name, val, fs)
	fs.Usage = func() {
		fs.PrintDefaults()
	}
	return fs
}

// Register fields in the given struct that have the tag `flag:"name,desc"`.
// Nested structs are supported as long as the field is a struct value field and not pointer to a struct.
// Exception to this is the use of StringList which needs to be a pointer.  The StringList type implements
// the Set and String methods required by the flag package and is dynamically allocated when registering its flag.
// See the test case for example.
func RegisterFlags(name string, val interface{}, fs *flag.FlagSet) {
	t := reflect.TypeOf(val).Elem()
	v := reflect.Indirect(reflect.ValueOf(val)) // the actual value of val
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			RegisterFlags(name+"."+field.Name, v.Field(i).Addr().Interface(), fs)
			continue
		}

		// See https://golang.org/ref/spec#Uniqueness_of_identifiers
		exported := field.PkgPath == ""
		if exported {

			tag := field.Tag
			spec := tag.Get("flag")
			if spec == "" {
				continue
			}

			// Bind the flag based on the tag spec
			f, d := "", ""
			p := strings.Split(spec, ",")
			if len(p) == 1 {
				// Just one field, use it as description
				f = fmt.Sprintf("%s.%s", name, strings.ToLower(field.Name))
				d = strings.Trim(p[0], " ")
			} else {
				// More than one, the first is the name of the flag
				f = strings.Trim(p[0], " ")
				d = strings.Trim(p[1], " ")
			}

			fv := v.Field(i).Interface()
			if v.Field(i).CanAddr() {
				ptr := v.Field(i).Addr().Interface() // The pointer value

				switch fv := fv.(type) {
				case bool:
					fs.BoolVar(ptr.(*bool), f, fv, d)
				case string:
					fs.StringVar(ptr.(*string), f, fv, d)
				case uint:
					fs.UintVar(ptr.(*uint), f, fv, d)
				case uint64:
					fs.Uint64Var(ptr.(*uint64), f, fv, d)
				case int64:
					fs.Int64Var(ptr.(*int64), f, fv, d)
				case int:
					fs.IntVar(ptr.(*int), f, fv, d)
				case float64:
					fs.Float64Var(ptr.(*float64), f, fv, d)
				case time.Duration:
					fs.DurationVar(ptr.(*time.Duration), f, fv, d)
				case []time.Duration:
					if len(fv) == 0 {
						// Special case where we allocate an empty list - otherwise it's default.
						v.Field(i).Set(reflect.ValueOf([]time.Duration{}))
					}
					fs.Var(&durationListProxy{list: ptr.(*[]time.Duration)}, f, d)
				default:
					// We only register if the field is a concrete vale and not a pointer
					// since we don't automatically allocate zero value structs to fill the field slot.
					switch field.Type.Kind() {

					case reflect.String:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: stringFromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Bool:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: boolFromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Float64:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: float64FromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Int:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: intFromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Int64:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: int64FromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Uint:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: uintFromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Uint64:
						fs.Var(&aliasProxy{
							fieldType:  field.Type,
							ptr:        ptr,
							fromString: uint64FromString,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}, f, d)
					case reflect.Struct:
						RegisterFlags(f, ptr, fs)
					case reflect.Slice:
						et := field.Type.Elem()
						proxy := &sliceProxy{
							fieldType: field.Type,
							elemType:  et,
							slice:     ptr,
							defaults:  reflect.ValueOf(fv).Len() > 0,
							toString: func(v interface{}) string {
								return fmt.Sprint("%v", v)
							},
						}
						fs.Var(proxy, f, d)
						switch {
						// Checking for string is placed here first because other types are
						// convertible to string as well.
						case reflect.TypeOf(string("")).ConvertibleTo(et):
							proxy.fromString = stringFromString
						case reflect.TypeOf(bool(true)).ConvertibleTo(et):
							proxy.fromString = boolFromString
						case reflect.TypeOf(float64(1.)).ConvertibleTo(et):
							proxy.fromString = float64FromString
						case reflect.TypeOf(int(1)).ConvertibleTo(et):
							proxy.fromString = intFromString
						case reflect.TypeOf(int64(1)).ConvertibleTo(et):
							proxy.fromString = int64FromString
						case reflect.TypeOf(uint(1)).ConvertibleTo(et):
							proxy.fromString = uintFromString
						case reflect.TypeOf(uint64(1)).ConvertibleTo(et):
							proxy.fromString = uint64FromString
						case reflect.TypeOf(time.Second).AssignableTo(et):
							proxy.fromString = durationFromString
						}
					}
				}
			}
		}
	}
}

func stringFromString(s string) (interface{}, error) {
	return s, nil
}

func boolFromString(s string) (interface{}, error) {
	value, err := strconv.ParseBool(s)
	if err != nil {
		return false, err
	}
	return value, nil
}

func float64FromString(s string) (interface{}, error) {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return float64(0), err
	}
	return value, nil
}

func intFromString(s string) (interface{}, error) {
	value, err := strconv.Atoi(s)
	if err != nil {
		return int(0), err
	}
	return value, nil
}

func int64FromString(s string) (interface{}, error) {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return int64(0), err
	}
	return value, nil
}

func uintFromString(s string) (interface{}, error) {
	value, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return uint(0), err
	}
	return value, nil
}

func uint64FromString(s string) (interface{}, error) {
	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return uint64(0), err
	}
	return value, nil
}

func durationFromString(s string) (interface{}, error) {
	value, err := time.ParseDuration(s)
	if err != nil {
		return false, err
	}
	return value, nil
}

// For a list of types that are convertible to string
type aliasProxy struct {
	fieldType  reflect.Type
	fromString func(string) (interface{}, error) // conversion from string
	toString   func(interface{}) string          // to string
	ptr        interface{}                       // the Pointer to the slice
}

func (this *aliasProxy) Set(value string) error {
	v, err := this.fromString(value)
	if err != nil {
		return err
	}
	newValue := reflect.ValueOf(reflect.ValueOf(v).Convert(this.fieldType).Interface())
	reflect.ValueOf(this.ptr).Elem().Set(newValue)
	return nil
}
func (this *aliasProxy) String() string {
	return this.toString(reflect.ValueOf(this.ptr).Elem().Interface())
}

// For a list of types that are convertible to string
type sliceProxy struct {
	fieldType  reflect.Type
	elemType   reflect.Type                      // the element type
	fromString func(string) (interface{}, error) // conversion from string
	toString   func(interface{}) string          // to string
	slice      interface{}                       // the Pointer to the slice
	defaults   bool                              // set to true on first time Set is called.
}

func (this *sliceProxy) Set(value string) error {
	v, err := this.fromString(value)
	if err != nil {
		return err
	}
	newElement := reflect.ValueOf(reflect.ValueOf(v).Convert(this.elemType).Interface())
	if this.defaults {
		reflect.ValueOf(this.slice).Elem().Set(reflect.Zero(this.fieldType))
		this.defaults = false
	}
	reflect.ValueOf(this.slice).Elem().Set(reflect.Append(reflect.ValueOf(this.slice).Elem(), newElement))
	return nil
}
func (this *sliceProxy) String() string {
	list := []string{}
	for i := 0; i < reflect.ValueOf(this.slice).Elem().Len(); i++ {
		str := this.toString(reflect.ValueOf(this.slice).Elem().Index(i).Interface())
		list = append(list, str)
	}
	return strings.Join(list, ",")
}

// Supports default values.  This means that if the slice was initialized with value, setting
// via flag will wipe out the existing value.
type durationListProxy struct {
	list *[]time.Duration
	set  bool // set to true on first time Set is called.
}

func (this *durationListProxy) Set(str string) error {
	value, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	if this.set {
		*this.list = append(*this.list, value)
	} else {
		// false means we have default value, now wipe it out
		*this.list = []time.Duration{value}
		this.set = true
	}
	return nil
}
func (this *durationListProxy) String() string {
	list := make([]string, len(*this.list))
	for i, v := range *this.list {
		list[i] = v.String()
	}
	return strings.Join(list, ",")
}

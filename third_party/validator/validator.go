package validator

import (
    "fmt"
    "reflect"
    "strings"
)

type Validate struct{}

func New() *Validate { return &Validate{} }

func (v *Validate) Struct(s interface{}) error {
    val := reflect.ValueOf(s)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }
    typ := val.Type()
    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        sf := typ.Field(i)
        tag := sf.Tag.Get("validate")
        if strings.Contains(tag, "required") {
            if isZero(field) {
                return fmt.Errorf("%s is required", sf.Name)
            }
        }
    }
    return nil
}

func isZero(v reflect.Value) bool {
    switch v.Kind() {
    case reflect.String:
        return v.Len() == 0
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return v.Int() == 0
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return v.Uint() == 0
    case reflect.Bool:
        return !v.Bool()
    case reflect.Slice, reflect.Map, reflect.Array:
        return v.Len() == 0
    case reflect.Struct:
        // assume non-zero
        return false
    default:
        return v.IsZero()
    }
}

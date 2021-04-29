package extractor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/query-go"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
)

var (
	yamlMapItemType  = reflect.TypeOf(yaml.MapItem{})
	yamlMapSliceType = reflect.TypeOf(yaml.MapSlice{})
)

// Key returns a new key extractor.
func Key(key string) query.Extractor {
	return &keyExtractor{key}
}

type keyExtractor struct {
	key string
}

// Extract implements query.Extractor interface.
func (e *keyExtractor) Extract(v reflect.Value) (reflect.Value, bool) {
	if v.IsValid() {
		if i, ok := v.Interface().(query.KeyExtractor); ok {
			x, ok := i.ExtractByKey(e.key)
			return reflect.ValueOf(x), ok
		}
	}
	return e.extract(v)
}

func (e *keyExtractor) extract(v reflect.Value) (reflect.Value, bool) {
	v = reflectutil.Elem(v)
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			k := reflectutil.Elem(k)
			if k.String() == e.key {
				return v.MapIndex(k), true
			}
		}
	case reflect.Struct:
		switch v.Type() {
		case yamlMapItemType:
			if fmt.Sprint(v.FieldByName("Key").Interface()) == e.key {
				return v.FieldByName("Value"), true
			}
		default:
			inlines := []int{}
			for i := 0; i < v.Type().NumField(); i++ {
				if !v.Field(i).CanInterface() {
					// Ignore unexported fields to avoid conflict with
					// other exported fields and "yaml" struct field tags.
					continue
				}
				field := v.Type().FieldByIndex([]int{i})
				name := strings.ToLower(field.Name)
				if tag, ok := field.Tag.Lookup("yaml"); ok {
					strs := strings.Split(tag, ",")
					for _, opt := range strs[1:] {
						if opt == "inline" {
							inlines = append(inlines, i)
						}
					}
					name = strs[0]
				}
				if name == e.key {
					return v.FieldByIndex([]int{i}), true
				}
			}
			for _, i := range inlines {
				if val, ok := e.Extract(v.Field(i)); ok {
					return val, true
				}
			}
		}
	case reflect.Slice:
		if v.Type() == yamlMapSliceType {
			for i := 0; i < v.Len(); i++ {
				elem := v.Index(i)
				if !elem.IsValid() {
					continue
				}
				if val, ok := e.Extract(elem); ok {
					return val, true
				}
			}
		}
	default:
	}
	return reflect.Value{}, false
}

// String implements query.Extractor interface.
func (e *keyExtractor) String() string {
	return "." + e.key
}

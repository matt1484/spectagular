package spectagular

import (
	"reflect"
	"time"
)

// StructTagOptionUnmarshaler is an interface used to convert a string value extracted
// from a field's struct tag options and convert it to its expected value.
type StructTagOptionUnmarshaler interface {
	UnmarshalTagOption(field reflect.StructField, value string) (reflect.Value, error)
}

// nameResolver is used to parse tags that use the first value as a "name"
// and default to the field name (i.e. json, yaml, etc.)
type nameResolver struct {
	resolver StructTagOptionUnmarshaler
}

func (n *nameResolver) UnmarshalTagOption(field reflect.StructField, value string) (reflect.Value, error) {
	if value == EmptyTag {
		return n.resolver.UnmarshalTagOption(field, field.Name)
	}
	return n.resolver.UnmarshalTagOption(field, value)
}

// boolResolver is used to parse tags of boolean values. if the key is present it is set to true
type boolResolver struct {
	key string
}

func (b *boolResolver) UnmarshalTagOption(field reflect.StructField, value string) (reflect.Value, error) {
	if value == b.key {
		return reflect.ValueOf(true), nil
	}
	return convertToValue(value, reflect.Bool)
}

// pointerResolver resolves a value and returns a pointer to it
type pointerResolver struct {
	resolver       StructTagOptionUnmarshaler
	underlyingType reflect.Type
}

func (p *pointerResolver) UnmarshalTagOption(field reflect.StructField, valueStr string) (reflect.Value, error) {
	v, err := p.resolver.UnmarshalTagOption(field, valueStr)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	value := reflect.New(p.underlyingType.Elem())
	value.Elem().Set(v.Convert(p.underlyingType.Elem()))
	return value, err
}

// arrayResolver is used to parse anything as an array
type sliceResolver struct {
	resolver       StructTagOptionUnmarshaler
	underlyingType reflect.Type
}

func (s *sliceResolver) UnmarshalTagOption(field reflect.StructField, tag string) (reflect.Value, error) {
	valueStr := ""
	value := reflect.MakeSlice(reflect.SliceOf(s.underlyingType), 0, 0)
	if len(tag) > 0 {
		if tag[0] == ',' {
			tag = "," + tag
		}
		if tag[len(tag)-1] == ',' {
			tag += ","
		}
	}
	var err error
	for tag != EmptyTag {
		if tag[0] == ',' {
			tag = tag[1:]
		}
		tag, valueStr, err = getNextTagValue(tag)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		val, err := s.resolver.UnmarshalTagOption(field, valueStr)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		value = reflect.Append(value, val)
	}
	return value, nil
}

// durationResolver is used to parse a duration string
type durationResolver struct{}

func (d *durationResolver) UnmarshalTagOption(field reflect.StructField, value string) (reflect.Value, error) {
	dur, err := time.ParseDuration(value)
	return reflect.ValueOf(dur), err
}

// defaultResolver is used to parse any other values
type defaultResolver struct {
	kind reflect.Kind
}

func (d *defaultResolver) UnmarshalTagOption(field reflect.StructField, value string) (reflect.Value, error) {
	return convertToValue(value, d.kind)
}

func getResolver(fType reflect.Type, name string) StructTagOptionUnmarshaler {
	if name == NameTag {
		return &nameResolver{
			resolver: getResolver(fType, ""),
		}
	}
	if fType.Implements(reflect.TypeOf((*StructTagOptionUnmarshaler)(nil)).Elem()) {
		return reflect.New(fType).Interface().(StructTagOptionUnmarshaler)
	}
	if fType == reflect.TypeOf(*new(time.Duration)) {
		return &durationResolver{}
	}
	if fType.Kind() == reflect.Slice {
		return &sliceResolver{
			resolver:       getResolver(fType.Elem(), name),
			underlyingType: fType.Elem(),
		}
	}
	if fType.Kind() == reflect.Pointer {
		return &pointerResolver{
			resolver:       getResolver(fType.Elem(), name),
			underlyingType: fType,
		}
	}
	if fType.Kind() == reflect.Bool {
		return &boolResolver{
			key: name,
		}
	}
	return &defaultResolver{
		kind: fType.Kind(),
	}
}

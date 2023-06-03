package spectagular

import (
	"fmt"
	"reflect"
	"time"
)

type TagValueResolver interface {
	ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error)
}

// nameResolver is used to parse tags that use the first value as a "name"
// and default to the field name (i.e. json, yaml, etc.)
type nameResolver struct {
	resolver TagValueResolver
}

func (n *nameResolver) ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error) {
	if value == EmptyTag {
		return n.resolver.ResolveTagValue(field, field.Name)
	}
	return n.resolver.ResolveTagValue(field, value)
}

// boolResolver is used to parse tags of boolean values. if the key is present it is set to true
type boolResolver struct {
	key string
}

func (b *boolResolver) ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error) {
	if value == b.key {
		return reflect.ValueOf(true), nil
	}
	return convertToValue(value, reflect.Bool)
}

// pointerResolver resolves a value and returns a pointer to it
type pointerResolver struct {
	resolver       TagValueResolver
	underlyingType reflect.Type
}

func (p *pointerResolver) ResolveTagValue(field reflect.StructField, valueStr string) (reflect.Value, error) {
	v, err := p.resolver.ResolveTagValue(field, valueStr)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	value := reflect.New(p.underlyingType.Elem())
	value.Elem().Set(v.Convert(p.underlyingType.Elem()))
	return value, err
}

// arrayResolver is used to parse anything as an array
type sliceResolver struct {
	resolver       TagValueResolver
	underlyingType reflect.Type
}

func (s *sliceResolver) ResolveTagValue(field reflect.StructField, tag string) (reflect.Value, error) {
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
	for tag != EmptyTag {
		if tag[0] == ',' {
			tag = tag[1:]
		}
		tag, valueStr = getNextTagValue(tag)
		val, err := s.resolver.ResolveTagValue(field, valueStr)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		value = reflect.Append(value, val)
	}
	return value, nil
}

// durationResolver is used to parse a duration string
type durationResolver struct{}

func (d *durationResolver) ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error) {
	fmt.Println(value)
	dur, err := time.ParseDuration(value)
	return reflect.ValueOf(dur), err
}

// defaultResolver is used to parse any other values
type defaultResolver struct {
	kind reflect.Kind
}

func (d *defaultResolver) ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error) {
	return convertToValue(value, d.kind)
}

func getResolver(fType reflect.Type, name string) TagValueResolver {
	if name == NameTag {
		return &nameResolver{
			resolver: getResolver(fType, ""),
		}
	}
	if fType.Implements(reflect.TypeOf((*TagValueResolver)(nil)).Elem()) {
		return reflect.New(fType).Interface().(TagValueResolver)
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

package spectagular

import "reflect"

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

// defaultResolver is used to parse any other values
type defaultResolver struct {
	kind reflect.Kind
}

func (d *defaultResolver) ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error) {
	return convertToValue(value, d.kind)
}

func getResolver(fType reflect.Type, name string, isArray bool) TagValueResolver {
	if name == NameTag {
		return &nameResolver{
			resolver: getResolver(fType, "", isArray),
		}
	}
	if fType.Implements(reflect.TypeOf((*TagValueResolver)(nil)).Elem()) {
		return reflect.New(fType).Interface().(TagValueResolver)
	}
	if fType.Kind() == reflect.Pointer {
		return &pointerResolver{
			resolver:       getResolver(fType.Elem(), name, isArray),
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

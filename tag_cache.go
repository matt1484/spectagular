package spectagular

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	EmptyTag     = ""
	SkipTag      = "-"
	BlankTag     = " "
	StructTagTag = "structtag"
	RequiredTag  = "required"
	NameTag      = "$name"
)

var (
	keyValueRegex       = regexp.MustCompile(`^(?:(\w+)=)?(.+)`)
	untilNextCommaRegex = regexp.MustCompile(`^([^,]*),?`)
	untilNextQuoteRegex = regexp.MustCompile(`^([^']*)'`)
)

func convertToValue(value string, kind reflect.Kind) (reflect.Value, error) {
	switch kind {
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		return reflect.ValueOf(v), err
	case reflect.String:
		return reflect.ValueOf(value), nil
	case reflect.Int8:
		v, err := strconv.ParseInt(value, 10, 8)
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(int8))), err
	case reflect.Int16:
		v, err := strconv.ParseInt(value, 10, 16)
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(int16))), err
	case reflect.Int32:
		v, err := strconv.ParseInt(value, 10, 32)
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(int32))), err
	case reflect.Int, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if kind == reflect.Int64 {
			return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(int64))), err
		}
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(int))), err
	case reflect.Uint8:
		v, err := strconv.ParseUint(value, 10, 8)
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(uint8))), err
	case reflect.Uint16:
		v, err := strconv.ParseUint(value, 10, 16)
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(uint16))), err
	case reflect.Uint32:
		v, err := strconv.ParseUint(value, 10, 32)
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(uint32))), err
	case reflect.Uint, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, 64)
		if kind == reflect.Uint64 {
			return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(uint64))), err
		}
		return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(uint))), err
	case reflect.Float32, reflect.Float64:
		var v float64
		var err error
		if kind == reflect.Float32 {
			v, err = strconv.ParseFloat(value, 32)
			return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(float32))), err
		}
		v, err = strconv.ParseFloat(value, 64)
		return reflect.ValueOf(v), err
	case reflect.Complex64, reflect.Complex128:
		var v complex128
		var err error
		if kind == reflect.Complex64 {
			v, err = strconv.ParseComplex(value, 64)
			return reflect.ValueOf(v).Convert(reflect.TypeOf(*new(complex64))), err
		}
		v, err = strconv.ParseComplex(value, 128)
		return reflect.ValueOf(v), err
	}
	return reflect.ValueOf(nil), errors.New("should not have made it here lol")
}

type FieldTag[V any] struct {
	FieldName  string
	FieldIndex int
	Value      V
}

type TagOption struct {
	Key   string
	Value string
}

type StructTag struct {
	Name       string
	Required   bool
	FieldIndex int
	Resolver   TagValueResolver
}

type FieldTagCache[T any] struct {
	tagNames     []string
	typeToTags   map[reflect.Type][]FieldTag[T]
	structTagMap map[string]StructTag
	hasName      bool
}

func NewFieldTagCache[T any](tagName string, aliases ...string) (*FieldTagCache[T], error) {
	defType := reflect.TypeOf(*new(T))
	switch defType.Kind() {
	case reflect.Struct:
		break
	case reflect.Pointer:
		defType = defType.Elem()
		if defType.Elem().Kind() == reflect.Struct {
			break
		}
		fallthrough
	default:
		return nil, errors.New("FieldTagCache needs a struct type for initialization")
	}
	hasName := false
	structTagMap := make(map[string]StructTag)
	for i := 0; i < defType.NumField(); i++ {
		field := defType.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		var isArray bool
		fieldKind := field.Type.Kind()
		if fieldKind == reflect.Array {
			isArray = true
			fieldKind = field.Type.Elem().Kind()
			continue
		}
		switch fieldKind {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Invalid, reflect.Struct, reflect.Map, reflect.UnsafePointer:
			continue
		}
		tags := field.Tag.Get(StructTagTag)
		structTag := StructTag{FieldIndex: i}
		for n, o := range append(strings.Split(tags, ","), strings.ToLower(field.Name)) {
			if n == 0 {
				if o != "-" {
					structTag.Name = o
				}
			} else {
				if o == RequiredTag {
					structTag.Required = true
				}
			}
		}
		if structTag.Name != EmptyTag && structTag.Name != BlankTag && structTag.Name != SkipTag {
			if structTag.Name == NameTag {
				hasName = true
			}
			structTag.Resolver = getResolver(field.Type, structTag.Name, isArray)
			structTagMap[structTag.Name] = structTag
		}
	}
	return &FieldTagCache[T]{
		tagNames:     append([]string{tagName}, aliases...),
		typeToTags:   make(map[reflect.Type][]FieldTag[T]),
		structTagMap: structTagMap,
		hasName:      hasName,
	}, nil
}

func (t *FieldTagCache[T]) Add(rType reflect.Type) error {
	kind := rType.Kind()
	if kind == reflect.Pointer || kind == reflect.Array {
		kind = rType.Elem().Kind()
		rType = rType.Elem()
	}
	if kind != reflect.Struct {
		return errors.New("FieldTagCache cannot parse non struct types")
	}

	var field reflect.StructField
	var tag string
	var key string
	var valueStr string
	var err error
	fieldTags := make([]FieldTag[T], 0)
	for i := 0; i < rType.NumField(); i++ {
		field = rType.Field(i)
		tag = field.Tag.Get(t.tagNames[0])
		field = rType.Field(i)
		if field.PkgPath != "" || field.Anonymous {
			continue
		}
		value := new(T)
		ft := FieldTag[T]{
			FieldName:  field.Name,
			FieldIndex: i,
		}
		ftv := reflect.Indirect(reflect.ValueOf(value))
		var v reflect.Value
		for i := 0; ; i++ {
			valueStr = ""
			kv := keyValueRegex.FindStringSubmatchIndex(tag)
			if kv == nil {
				break
			}
			keyStart, keyEnd, valueStart, valueEnd := kv[2], kv[3], kv[4], kv[5]
			if keyEnd > 0 {
				key = tag[keyStart:keyEnd]
			} else {
				key = ""
			}
			if valueEnd > 0 {
				tag = tag[valueStart:valueEnd]
				if tag[0] == '\'' {
					tag = tag[1:]
					for {
						kv = untilNextQuoteRegex.FindStringSubmatchIndex(tag)
						valueStr += tag[kv[2]:kv[3]]
						if kv != nil && kv[3] > 0 && kv[3] > kv[2] && tag[kv[3]-1] == '\\' {
							valueStr += "'"
							tag = tag[kv[1]:]
						} else {
							break
						}
					}
					if kv != nil {
						tag = tag[kv[1]:]
					}
					// } else if value[valueStart] == '[' {
					// TODO: handle array?
				} else {
					kv = untilNextCommaRegex.FindStringSubmatchIndex(tag)
					valueStart, valueEnd = kv[2], kv[3]
					valueStr = strings.Replace(tag[valueStart:valueEnd], `\'`, `'`, -1)
					tag = tag[kv[1]:]
				}
				if i == 0 && t.hasName {
					key = NameTag
				} else if key == "" {
					key = valueStr
				}
				if st, ok := t.structTagMap[key]; ok {
					v, err = st.Resolver.ResolveTagValue(field, valueStr)
					if err != nil {
						if st.Required {
							// may potentially want to allow for a not-found error to be checked or something?
							return err
						}
					} else {
						ftv.Field(st.FieldIndex).Set(v)
					}
				}
			} else {
				break
			}
		}
		ft.Value = *value
		fieldTags = append(fieldTags, ft)
	}
	t.typeToTags[rType] = fieldTags
	return nil
}

func (t *FieldTagCache[T]) Get(rType reflect.Type) ([]FieldTag[T], bool) {
	tags, ok := t.typeToTags[rType]
	return tags, ok
}

func (t *FieldTagCache[T]) GetOrAdd(rType reflect.Type) ([]FieldTag[T], error) {
	tags, ok := t.typeToTags[rType]
	if !ok {
		err := t.Add(rType)
		return t.typeToTags[rType], err
	}
	return tags, nil
}

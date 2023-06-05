package spectagular

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	// EmptyTag is used to denote a tag with nothing in it
	EmptyTag = ""
	// SkipTag is used to denote that the current field should not be used for parsing
	SkipTag = "-"
	// StructTagTag is the tag used by this package that defines the options for a struct tag field
	StructTagTag = "structtag"
	// RequiredTag is used to denote a this struct tag field is required
	RequiredTag = "required"
	// NameTag is used to denote the first field or the name of the field if empty
	// (i.e. how its used for encoding/json, encoding/yaml, etc.).
	NameTag = "$name"
)

var (
	keyValueRegex         = regexp.MustCompile(`^(?:(\w+)=)?(.+)`)
	untilNextCommaRegex   = regexp.MustCompile(`^([^,]*),?`)
	untilNextQuoteRegex   = regexp.MustCompile(`^([^']*)'`)
	untilNextBracketRegex = regexp.MustCompile(`^([^\]]*)]`)
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
	return reflect.ValueOf(nil), errors.New("unable to convert string to kind: " + kind.String())
}

// FieldTag[V any] is the parsed struct tag value for the struct field with the
// corresponding name and index of said field as provided by "reflect".
type FieldTag[V any] struct {
	// FieldName is the name of the field that these tags apply too. It is included
	// since most of the time when you are parsing struct tags you need to know
	// some limited information about the field.
	FieldName string
	// FieldIndex is the index of the field that these tags apply too. It is included
	// since most of the time when you are parsing struct tags you need to know
	// some limited information about the field.
	FieldIndex int
	// Value is the parsed value of the struct tags for a field in a struct.
	Value V
}

// StructTagOption is the definition of an option for a defined struct tag type. An example being how
// encoding/json has "name", "omitempty", and "string" as options.
type StructTagOption struct {
	Name       string
	Required   bool
	FieldIndex int
	Resolver   StructTagOptionUnmarshaler
}

// StructTagCache[T any] is a cache for parsed struct tags. It is used to parse a struct's tag defined
// by type T and store them as mapping of the struct's type to []FieldTag[T] for easy lookup later.
// While tags could be parsed as needed, this struct is designed for workflows like encoding/json
// where the same type may need its struct tags parsed more than once.
type StructTagCache[T any] struct {
	tagName      string
	typeToTags   map[reflect.Type][]FieldTag[T]
	structTagMap map[string]StructTagOption
	hasName      bool
	requiredTags []string
}

// NewFieldTagCache[T any] initializes a StructTagCache for type T.
func NewFieldTagCache[T any](tagName string) (*StructTagCache[T], error) {
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
	structTagMap := make(map[string]StructTagOption)
	requiredTags := make([]string, 0)
	for i := 0; i < defType.NumField(); i++ {
		field := defType.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		tags := field.Tag.Get(StructTagTag)
		structTag := StructTagOption{FieldIndex: i}
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
		if structTag.Name != EmptyTag && structTag.Name != SkipTag {
			fieldKind := field.Type.Kind()
			if fieldKind == reflect.Slice {
				// just check for a 1d array, multidimensional arrays are not ideal for structtags imo
				// and just wont be supported unless users decide to create their own resolvers
				fieldKind = field.Type.Elem().Kind()
			}
			switch fieldKind {
			case reflect.Slice, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Invalid, reflect.Map, reflect.UnsafePointer:
				// im unwilling to try to support the above types, so only solution is to create a custom resolver
				// over a "raw" string value
				if !field.Type.Implements(reflect.TypeOf((*StructTagOptionUnmarshaler)(nil)).Elem()) {
					return nil, fmt.Errorf("unsupported type for struct tag: %s", field.Type)
				}
			}
			if structTag.Name == NameTag {
				hasName = true
			}
			structTag.Resolver = getResolver(field.Type, structTag.Name)
			if _, ok := structTagMap[structTag.Name]; ok {
				return nil, errors.New("tag '" + structTag.Name + "' is in use by multiple fields")
			}
			structTagMap[structTag.Name] = structTag
			if structTag.Required {
				requiredTags = append(requiredTags, structTag.Name)
			}
		}
	}
	return &StructTagCache[T]{
		tagName:      tagName,
		typeToTags:   make(map[reflect.Type][]FieldTag[T]),
		structTagMap: structTagMap,
		hasName:      hasName,
		requiredTags: requiredTags,
	}, nil
}

func getNextTagValue(tag string) (string, string, error) {
	valueStr := ""
	var kv []int
	if tag != EmptyTag && tag[0] == '\'' {
		tag = tag[1:]
		for {
			kv = untilNextQuoteRegex.FindStringSubmatchIndex(tag)
			if kv == nil {
				return "", "", errors.New("missing end quote on quoted string")
			}
			valueStr += tag[kv[2]:kv[3]]
			if kv[3] > 0 && kv[3] > kv[2] && tag[kv[3]-1] == '\\' {
				valueStr = valueStr[:len(valueStr)-1] + "'"
				tag = tag[kv[1]:]
			} else {
				break
			}
		}
		if kv != nil {
			tag = tag[kv[1]:]
		}
	} else {
		kv = untilNextCommaRegex.FindStringSubmatchIndex(tag)
		valueStart, valueEnd := kv[2], kv[3]
		valueStr = strings.Replace(tag[valueStart:valueEnd], `\'`, `'`, -1)
		tag = tag[kv[1]:]
	}
	return tag, valueStr, nil
}

// Add parses the struct tags from the type given and adds them to the internal cache while
// returning any validation errors found.
func (t *StructTagCache[T]) Add(rType reflect.Type) error {
	kind := rType.Kind()
	if kind == reflect.Pointer || kind == reflect.Array {
		kind = rType.Elem().Kind()
		rType = rType.Elem()
	}
	if kind != reflect.Struct {
		return errors.New("FieldTagCache cannot cache non struct types")
	}

	var field reflect.StructField
	var tag string
	var key string
	var valueStr string
	var err error
	fieldTags := make([]FieldTag[T], 0)
	requiredTags := make([]string, 0)
	for i := 0; i < rType.NumField(); i++ {
		field = rType.Field(i)
		tag = field.Tag.Get(t.tagName)
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
				if tag[0] == '[' {
					tag = tag[1:]
					for {
						kv = untilNextBracketRegex.FindStringSubmatchIndex(tag)
						if kv == nil {
							return errors.New("missing end quote on quoted string")
						}
						valueStr += tag[kv[2]:kv[3]]
						if kv[3] > 0 && kv[3] > kv[2] && tag[kv[3]-1] == '\\' {
							valueStr = valueStr[:len(valueStr)-1] + "]"
							tag = tag[kv[1]:]
						} else {
							break
						}
					}
					if kv != nil {
						tag = tag[kv[1]:]
					}
				} else {
					tag, valueStr, err = getNextTagValue(tag)
					if err != nil {
						return err
					}
				}
				if i == 0 && t.hasName {
					key = NameTag
				} else if key == "" {
					key = valueStr
				}
				if st, ok := t.structTagMap[key]; ok {
					v, err = st.Resolver.UnmarshalTagOption(field, valueStr)
					if err != nil {
						if st.Required {
							// may potentially want to allow for a not-found error to be checked or something?
							return err
						}
					} else {
						if !v.CanConvert(ftv.Field(st.FieldIndex).Type()) {
							return fmt.Errorf("unable to convert value of '%s' to type '%s' for field '%s'", ftv.Type().Field(st.FieldIndex).Name, ftv.Field(st.FieldIndex).Type(), field.Name)
						}
						ftv.Field(st.FieldIndex).Set(v.Convert(ftv.Field(st.FieldIndex).Type()))
						if st.Required {
							requiredTags = append(requiredTags, st.Name)
						}
					}
				}
			} else {
				break
			}
		}
		ft.Value = *value
		fieldTags = append(fieldTags, ft)
	}
	if len(requiredTags) != len(t.requiredTags) {
		requiredMap := make(map[string]struct{})
		for _, r := range t.requiredTags {
			requiredMap[r] = struct{}{}
		}
		for _, r := range requiredTags {
			delete(requiredMap, r)
		}
		requiredTags := make([]string, 0)
		for r := range requiredMap {
			requiredTags = append(requiredTags, r)
		}
		return fmt.Errorf("missing required tag fields: %s", requiredTags)
	}
	t.typeToTags[rType] = fieldTags
	return nil
}

// Get returns a []FieldTag for a type if it is found in the cache.
func (t *StructTagCache[T]) Get(rType reflect.Type) ([]FieldTag[T], bool) {
	tags, ok := t.typeToTags[rType]
	return tags, ok
}

// GetOrAdd returns a []FieldTag for a type if it is found in the cache and adds/returns it
// otherwise.
func (t *StructTagCache[T]) GetOrAdd(rType reflect.Type) ([]FieldTag[T], error) {
	tags, ok := t.typeToTags[rType]
	if !ok {
		err := t.Add(rType)
		return t.typeToTags[rType], err
	}
	return tags, nil
}

// ParseTagsForType[T any] parses the struct tags for a given type and converts them to type T.
func ParseTagsForType[T any](tagName string, rType reflect.Type) ([]FieldTag[T], error) {
	cache, err := NewFieldTagCache[T](tagName)
	if err != nil {
		return nil, err
	}
	return cache.GetOrAdd(rType)
}

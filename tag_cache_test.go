package spectagular_test

import (
	"reflect"
	"testing"

	"github.com/matt1484/spectagular"
)

func TestNewTagCache(t *testing.T) {
	type NewTest struct {
		S string `structtag:"name"`
	}
	cache, err := spectagular.NewFieldTagCache[NewTest]("test")
	if cache == nil || err != nil {
		t.Error("TestNewTagCache: failed struct validation", err.Error())
	}
}

func TestNewTagCacheInvalid(t *testing.T) {
	cache, err := spectagular.NewFieldTagCache[string]("test")
	if cache != nil || err == nil {
		t.Error("TestNewTagCacheInvalid: failed struct validation")
	}
	type Bad struct {
		F  int `structtag:"name"`
		F2 int `structtag:"name"`
	}
	badCache, err := spectagular.NewFieldTagCache[Bad]("test")
	if badCache != nil || err == nil {
		t.Error("TestNewTagCacheInvalid: failed duplicate name test")
	}
}

func TestQuotedTags(t *testing.T) {
	type TestQuotedTag struct {
		String string `structtag:"s"`
	}
	type TestQuotedStruct struct {
		Int     int `test:"s='123'"`
		Spaces  int `test:"s='with spaces'"`
		Empty   int `test:"s=''"`
		Escaped int `test:"s='has\\'quotes\\''"`
	}
	cache, _ := spectagular.NewFieldTagCache[TestQuotedTag]("test")
	tags, err := cache.GetOrAdd(reflect.TypeOf(TestQuotedStruct{}))
	if err != nil {
		t.Error("TestQuotedTags: failed quoted tags validation", err.Error())
	}
	for _, tag := range tags {
		switch tag.FieldName {
		case "Int":
			if tag.Value.String != "123" {
				t.Error("TestQuotedTags: wrong parsed value:", tag.Value.String, ", expected:", "123")
			}
		case "Spaces":
			if tag.Value.String != "with spaces" {
				t.Error("TestQuotedTags: wrong parsed value:", tag.Value.String, ", expected:", "with spaces")
			}
		case "Empty":
			if tag.Value.String != "" {
				t.Error("TestQuotedTags: wrong parsed value:", tag.Value.String, ", expected:", "")
			}
		case "Escaped":
			if tag.Value.String != "has'quotes'" {
				t.Error("TestQuotedTags: wrong parsed value:", tag.Value.String, ", expected:", "has'quotes'")
			}
		}
	}
}

func TestSpecialTags(t *testing.T) {
	type TestSpecialTag struct {
		Name     string  `structtag:"$name"`
		Required string  `structtag:"r,required"`
		Pointer  *string `structtag:"p"`
	}
	type TestSpecialStructValid struct {
		Valid int `test:"name,r='r',p=p"`
	}
	cache, _ := spectagular.NewFieldTagCache[TestSpecialTag]("test")
	tags, err := cache.GetOrAdd(reflect.TypeOf(TestSpecialStructValid{}))
	if err != nil {
		t.Error("TestSpecialTags: failed special tags validation", err.Error())
	}
	if tags[0].Value.Name != "name" {
		t.Error("TestSpecialTags: wrong parsed value:", tags[0].Value.Name, ", expected:", "name")
	}
	if tags[0].Value.Required != "r" {
		t.Error("TestSpecialTags: wrong parsed value:", tags[0].Value.Name, ", expected:", "r")
	}
	if *tags[0].Value.Pointer != "p" {
		t.Error("TestSpecialTags: wrong parsed value:", *tags[0].Value.Pointer, ", expected:", "p")
	}
	type TestSpecialStructInvalid struct {
		Valid int `test:"name"`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestSpecialStructInvalid{}))
	if err == nil || tags != nil {
		t.Error("TestSpecialTags: failed special tags validation with invalid tags")
	}
}

type CustomType struct {
	C string
}

func (c *CustomType) ResolveTagValue(field reflect.StructField, value string) (reflect.Value, error) {
	return reflect.ValueOf(CustomType{C: value}), nil
}

func TestTypeConversion(t *testing.T) {
	type TestTagTypes struct {
		String     string     `structtag:"s"`
		Bool       bool       `structtag:"b"`
		Int        int        `structtag:"i"`
		Int8       int8       `structtag:"i8"`
		Int16      int16      `structtag:"i16"`
		Int32      int32      `structtag:"i32"`
		Int64      int64      `structtag:"i64"`
		Uint       uint       `structtag:"u"`
		Uint8      uint8      `structtag:"u8"`
		Uint16     uint16     `structtag:"u16"`
		Uint32     uint32     `structtag:"u32"`
		Uint64     uint64     `structtag:"u64"`
		Float32    float32    `structtag:"f32"`
		Float64    float64    `structtag:"f64"`
		Complex64  complex64  `structtag:"c64"`
		Complex128 complex128 `structtag:"c128"`
		CustomType CustomType `structtag:"ct"`
		StringList []string   `structtag:"sa"`
		IntList    []int      `structtag:"ia"`
	}
	cache, _ := spectagular.NewFieldTagCache[TestTagTypes]("test")
	// only going to test valid string representations
	// testing everything would be the same as testing strconv
	// and I just need to prove that strconv gets called anyways
	type TestStringValid struct {
		Empty  int `test:"s="`
		String int `test:"s=a string"`
	}
	tags, err := cache.GetOrAdd(reflect.TypeOf(TestStringValid{}))
	if err != nil || tags == nil {
		t.Error("TestTypeConversion: failed string tags validation", err.Error())
	}
	type TestBoolValid struct {
		True    int `test:"b"`
		False   int `test:"b=false"`
		NotTrue int `test:"b=not true"`
		Empty   int `test:""`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestBoolValid{}))
	if err != nil || tags == nil {
		t.Error("TestTypeConversion: failed bool tags validation", err.Error())
	}
	type TestOtherValid struct {
		Ints       int `test:"i=-1,i8=2,i16=3,i32=4,i64=5"`
		Uints      int `test:"ui=1,ui8=2,ui16=3,ui32=4,ui64=5"`
		Floats     int `test:"f32=-1.0,f64=2"`
		Complex64  int `test:"c64=-1,c128=2+3i"`
		CustomType int `test:"ct=a value"`
		Arrays     int `test:"sa=['quoted spaces',not quoted spaces,],ia=[-1,2]"`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestOtherValid{}))
	if err != nil || tags == nil {
		t.Error("TestTypeConversion: failed other tags validation", err.Error())
	}
}

package spectagular_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/matt1484/spectagular"
)

type assertType interface {
	string |
		bool |
		int |
		int8 |
		int16 |
		int32 |
		int64 |
		uint |
		uint8 |
		uint16 |
		uint32 |
		uint64 |
		float32 |
		float64 |
		complex64 |
		complex128 |
		time.Duration
}

func assertEqual[T assertType](t *testing.T, actual, expected T, message string) {
	if actual != expected {
		t.Error(message, "actual:", actual, ", expected:", expected)
	}
}

func assertNotEqual[T assertType](t *testing.T, actual, expected T, message string) {
	if actual == expected {
		t.Error(message, "actual:", actual, ", expected:", expected)
	}
}

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
			assertEqual(t, tag.Value.String, "123", "TestQuotedTags: wrong parsed value:")
		case "Spaces":
			assertEqual(t, tag.Value.String, "with spaces", "TestQuotedTags: wrong parsed value:")
		case "Empty":
			assertEqual(t, tag.Value.String, "", "TestQuotedTags: wrong parsed value:")
		case "Escaped":
			assertEqual(t, tag.Value.String, "has'quotes'", "TestQuotedTags: wrong parsed value:")
		}
	}
	type TestQuotedInvalid struct {
		Invalid int `test:"s='test string"`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestQuotedInvalid{}))
	if err == nil {
		t.Error("TestQuotedTags: failed quoted tags invalidation")
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
	assertEqual(t, tags[0].Value.Name, "name", "TestSpecialTags: wrong parsed value:")
	assertEqual(t, tags[0].Value.Required, "r", "TestSpecialTags: wrong parsed value:")
	assertEqual(t, *tags[0].Value.Pointer, "p", "TestSpecialTags: wrong parsed value:")
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

func (c CustomType) UnmarshalTagOption(field reflect.StructField, value string) (reflect.Value, error) {
	return reflect.ValueOf(CustomType{C: value}), nil
}

func TestTypeConversion(t *testing.T) {
	type TestTagTypes struct {
		String     string        `structtag:"s"`
		Bool       bool          `structtag:"b"`
		Int        int           `structtag:"i"`
		Int8       int8          `structtag:"i8"`
		Int16      int16         `structtag:"i16"`
		Int32      int32         `structtag:"i32"`
		Int64      int64         `structtag:"i64"`
		Uint       uint          `structtag:"u"`
		Uint8      uint8         `structtag:"u8"`
		Uint16     uint16        `structtag:"u16"`
		Uint32     uint32        `structtag:"u32"`
		Uint64     uint64        `structtag:"u64"`
		Float32    float32       `structtag:"f32"`
		Float64    float64       `structtag:"f64"`
		Complex64  complex64     `structtag:"c64"`
		Complex128 complex128    `structtag:"c128"`
		CustomType CustomType    `structtag:"ct"`
		StringList []string      `structtag:"sa"`
		IntList    []int         `structtag:"ia"`
		Duration   time.Duration `structtag:"d"`
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
	// only testing the field name/index here since it should be the same process for each
	assertEqual(t, tags[0].FieldIndex, 0, "TestTypeConversion: wrong field index:")
	assertEqual(t, tags[0].FieldName, "Empty", "TestTypeConversion: wrong field name:")
	assertEqual(t, tags[0].Value.String, "", "TestTypeConversion: wrong parsed string value:")
	assertEqual(t, tags[1].FieldIndex, 1, "TestTypeConversion: wrong field index:")
	assertEqual(t, tags[1].FieldName, "String", "TestTypeConversion: wrong field name:")
	assertEqual(t, tags[1].Value.String, "a string", "TestTypeConversion: wrong parsed string value:")
	type TestBoolValid struct {
		True         int `test:"b"`
		False        int `test:"b=false"`
		NotTrue      int `test:"b=not true"`
		Empty        int `test:""`
		TrueExplicit int `test:"b=true"`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestBoolValid{}))
	if err != nil || tags == nil {
		t.Error("TestTypeConversion: failed bool tags validation", err.Error())
	}
	assertEqual(t, tags[0].Value.Bool, true, "TestTypeConversion: wrong parsed bool value:")
	assertEqual(t, tags[1].Value.Bool, false, "TestTypeConversion: wrong parsed bool value:")
	assertEqual(t, tags[2].Value.Bool, false, "TestTypeConversion: wrong parsed bool value:")
	assertEqual(t, tags[3].Value.Bool, false, "TestTypeConversion: wrong parsed bool value:")
	assertEqual(t, tags[4].Value.Bool, true, "TestTypeConversion: wrong parsed bool value:")
	type TestOtherValid struct {
		Ints       int `test:"i=-1,i8=2,i16=3,i32=4,i64=5"`
		Uints      int `test:"u=1,u8=2,u16=3,u32=4,u64=5"`
		Floats     int `test:"f32=-1.0,f64=2"`
		Complex64  int `test:"c64=-1,c128=2+3i"`
		CustomType int `test:"ct=a value"`
		Arrays     int `test:"sa=['quoted spaces',not quoted spaces,],ia=[-1,2]"`
		Duration   int `test:"d=5h"`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestOtherValid{}))
	if err != nil || tags == nil {
		t.Error("TestTypeConversion: failed other tags validation", err.Error())
	}
	assertEqual(t, tags[0].Value.Int, -1, "TestTypeConversion: wrong parsed int value:")
	assertEqual(t, tags[0].Value.Int8, 2, "TestTypeConversion: wrong parsed int value:")
	assertEqual(t, tags[0].Value.Int16, 3, "TestTypeConversion: wrong parsed int value:")
	assertEqual(t, tags[0].Value.Int32, 4, "TestTypeConversion: wrong parsed int value:")
	assertEqual(t, tags[0].Value.Int64, 5, "TestTypeConversion: wrong parsed int value:")
	assertEqual(t, tags[1].Value.Uint, 1, "TestTypeConversion: wrong parsed uint value:")
	assertEqual(t, tags[1].Value.Uint8, 2, "TestTypeConversion: wrong parsed uint value:")
	assertEqual(t, tags[1].Value.Uint16, 3, "TestTypeConversion: wrong parsed uint value:")
	assertEqual(t, tags[1].Value.Uint32, 4, "TestTypeConversion: wrong parsed uint value:")
	assertEqual(t, tags[2].Value.Float32, -1.0, "TestTypeConversion: wrong parsed float value:")
	assertEqual(t, tags[2].Value.Float64, 2.0, "TestTypeConversion: wrong parsed float value:")
	assertEqual(t, tags[3].Value.Complex64, -1, "TestTypeConversion: wrong parsed complex value:")
	assertEqual(t, tags[3].Value.Complex128, 2+3i, "TestTypeConversion: wrong parsed complex value:")
	assertEqual(t, tags[4].Value.CustomType.C, "a value", "TestTypeConversion: wrong parsed custom value:")
	assertEqual(t, tags[5].Value.StringList[0], "quoted spaces", "TestTypeConversion: wrong parsed array value:")
	assertEqual(t, tags[5].Value.StringList[1], "not quoted spaces", "TestTypeConversion: wrong parsed array value:")
	assertEqual(t, tags[5].Value.StringList[2], "", "TestTypeConversion: wrong parsed array value:")
	assertEqual(t, tags[5].Value.IntList[0], -1, "TestTypeConversion: wrong parsed array value:")
	assertEqual(t, tags[5].Value.IntList[1], 2, "TestTypeConversion: wrong parsed array value:")
	assertEqual(t, tags[6].Value.Duration, 5*time.Hour, "TestTypeConversion: wrong parsed duration value:")
	type TestInvalidArray struct {
		Arrays int `test:"sa=["`
	}
	tags, err = cache.GetOrAdd(reflect.TypeOf(TestInvalidArray{}))
	if err == nil {
		t.Error("TestTypeConversion: failed invalid array validation")
	}
}

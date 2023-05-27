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

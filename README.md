# :sparkles: `spectagular`
A simple go library for parsing and managing struct tags as structs using generics and nothing but the standard library.

In general it's not very often you need to parse struct tags, but if you do this library is designed to help by automatically converting struct tags to their expected value as well as cache/validate said values. A good example of how this could be useful is a package like the `encoding/json` package which uses a well defined set of struct tag options that could be described with the following struct:

```go
type JSONStructTag struct {
    // here $name indicates that it is either the first option or the name of the field if empty
    Name      string `structtag:"$name"`
    OmitEmpty bool   `structtag:"omitempty"`
    String    bool   `structtag:"string"`
}
```
By using this package you could then parse any struct with `json` tags like so:

```go
// an example of a typical struct with json tags
type Person struct {
    Name string `json:",omitempty"`
    Age  int    `json:"age"`
}

spectagular.ParseTagsForType(reflect.TypeOf(&Person{}))
// this will return a []spectagular.FieldTag that is equivalent to:
/*
[]FieldTag{{ 
    FieldName: "Name",
    FieldIndex: 0,
    Value: JSONStructTag{ Name: "Name", OmitEmpty: true, String: true },
}, {
    FieldName: "Age",
    FieldIndex: 1,
    Value: JSONStructTag{ Name: "age", OmitEmpty: false, String: true },
}}
*/
```

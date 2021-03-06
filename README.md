# convertor
convertor convert source struct to destination struct, 
but the field tree of this two struct need to be same, 
or there is a convert function to deal with the different sub tree
you can register convert function by call RegisterConvertFunc
```go
RegisterConvertFunc(func(a string,b *int64) (err error) {
    *b, err = strconv.ParseInt(a, 10, 64)
    return
})
```
In addition, there are some rules to convert struct to field tree:
- Convertor use field name as default name of a field, but a convertor tag value can cover it.
- Embed anonymous struct field will be flatten by default.
- If the Embed anonymous field has a convertor tag, it will not be flatten.
- If a direct field name is conflict with a field name of embed anonymous struct after flatten, the direct field is prior, this is the default behavior of golang.
- If a direct field name is conflict with another direct field's convertor tag, it will return error, you should explicitly ignore a field by convertor tag.
- A field with convertor tag - will be ignored.
- A struct field with convertor tag + will be flatten.
- If two type is assignable, it will use reflect.Value.Set to assign direct.
- Different int type or float type can convert, but it can't convert between int and float type, you can use a convert func to deal with it.

Not support list:
- Not support over two level pointer.
- Not support Map and Array type
- Not support circle struct rely, it will return error, for example: struct A has a field struct B, and struct B has a field struct A

Example:
```go
type TypeB struct {
    FieldBB string
    FieldCC int
}

type TypeC struct {
    XX float32
}

type TypeA struct {
    TypeB // embed anonymous struct, flatten
    FieldBB string // cover TypeB's FieldBB field
    CCC     struct {
        FieldDD float64
        aa      int
    } `convertor:"CCCC"`
    DDD      string `convertor:"-"` // ignore field
    bb       TypeC // unexport
    FieldEEE *struct { // normal struct
        EE int
        FF struct {
            GG int
        }
    }
    TypeC   `convertor:"FF"` // embed anonymous struct with convertor tag, don't flatten
    dd      string
    Flatten struct { // flatten
        FlattenField string
    } `convertor:"+"`
}
```
Field tree:
```
TypeA
  CCCC
    FieldDD
  FF
    XX
  FieldBB
  FieldCC
  FieldEEE
    EE
    FF
      GG
  FlattenField
```
If another struct has same field tree with TypeA will be convertible to TypeA or convertible from TypeA.

## example code
```go
func ExampleConvert() {
	type InnerType struct {
		FieldA string
		FieldB int
	}
	type Product struct {
		Code int    `convertor:"ProductCode"`
		Name string `convertor:"ProductName"`
	}
	type Anonymous struct {
		AnonFieldA int64
		AnonFieldB string
	}
	type TypeA struct {
		AA           InnerType
		P            Product                      `convertor:"+"` // flatten struct
		Anonymous                                 // flatten struct
		AnonFieldB   string                       // cover Anonymous's Field AnonFieldB
		IgnoreFieldA string                       `convertor:"-"`
		Product      `convertor:"NamedAnonymous"` // don't flatten struct
		ignoreField  string                       // unexport field will ignore
	}
	var b = struct {
		AA             string
		ProductCode    int
		ProductName    *string
		AnonFieldA     int32
		AnonFieldB     string
		IgnoreFieldB   string `convertor:"-"`
		NamedAnonymous Product
	}{}
	// convert func
	convertor.RegisterConvertFunc(
		func(src InnerType, dest *string) error {
			data, err := json.Marshal(src)
			if err != nil {
				return err
			}
			*dest = string(data)
			return nil
		},
	)
	if err := convertor.Convert(TypeA{
		AA: InnerType{
			FieldA: "field a",
			FieldB: 2,
		},
		P: Product{
			Code: 1000,
			Name: "P",
		},
		Anonymous: Anonymous{
			AnonFieldA: 2000,
			AnonFieldB: "inner AnonFieldB",
		},
		AnonFieldB:   "outer AnonFieldB",
		IgnoreFieldA: "xxx",
		Product: Product{
			Code: 3000,
			Name: "NamedAnonymous",
		},
		ignoreField: "unexport ignore field",
	}, &b); err != nil {
		panic(err)
	}
	fmt.Printf("b.AA: %v\n", b.AA)
	fmt.Printf("b.ProductCode: %v\n", b.ProductCode)
	fmt.Printf("b.ProductName: %v\n", *b.ProductName)
	fmt.Printf("b.AnonFieldA: %v\n", b.AnonFieldA)
	fmt.Printf("b.AnonFieldB: %v\n", b.AnonFieldB)
	fmt.Printf("b.NamedAnonymous: %+v\n", b.NamedAnonymous)
	// Output:
	// b.AA: {"FieldA":"field a","FieldB":2}
	// b.ProductCode: 1000
	// b.ProductName: P
	// b.AnonFieldA: 2000
	// b.AnonFieldB: outer AnonFieldB
	// b.NamedAnonymous: {Code:3000 Name:NamedAnonymous}
}

// don't return error if destination struct fields not exist in source struct
func ExampleSrcNotExistFieldIgnoreConvertor() {
	type TypeA struct {
		FieldA string
	}
	type TypeB struct {
		FieldA string
		FieldB *int
	}
	var b TypeB
	if err := convertor.SrcNotExistFieldIgnoreConvertor.Convert(TypeA{FieldA: "fieldA"}, &b); err != nil {
		panic(err)
	}
	fmt.Printf("b.FieldA: %v\n", b.FieldA)
	fmt.Printf("b.FieldB: %v\n", b.FieldB)
	// Output:
	// b.FieldA: fieldA
	// b.FieldB: <nil>
}

// don't return error if source struct fields not exist in destination struct
func ExampleDestNotExistFieldIgnoreConvertor() {
	type TypeA struct {
		FieldA string
		FieldB int
	}
	type TypeB struct {
		FieldA string
	}
	var b TypeB
	if err := convertor.DestNotExistFieldIgnoreConvertor.Convert(TypeA{FieldA: "fieldA", FieldB: 10}, &b); err != nil {
		panic(err)
	}
	fmt.Printf("b.FieldA: %v\n", b.FieldA)
	// Output:
	// b.FieldA: fieldA
}

```

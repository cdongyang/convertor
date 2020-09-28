package convertor_test

import (
	"encoding/json"
	"fmt"

	"github.com/cdongyang/convertor"
)

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

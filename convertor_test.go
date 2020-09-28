package convertor

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestGetCacheStruct(t *testing.T) {
	type TypeB struct {
		FieldBB string
		FieldCC int
	}

	type TypeC struct {
		XX float32
	}

	type TypeA struct {
		TypeB
		FieldBB string
		CCC     struct {
			FieldDD float64
			aa      int
		} `convertor:"CCCC"`
		DDD      string `convertor:"-"`
		bb       TypeC
		FieldEEE *struct {
			EE int
			FF struct {
				GG int
			}
		}
		TypeC   `convertor:"FF"`
		dd      string
		Flatten struct {
			FlattenField string
		} `convertor:"+"`
	}
	s := getCacheStruct(reflect.TypeOf(TypeA{}), nil)
	type result struct {
		Name   string
		Fields []result
	}
	res := []result{
		{Name: "CCCC", Fields: []result{{Name: "FieldDD"}}},
		{Name: "FF", Fields: []result{{Name: "XX"}}},
		{Name: "FieldBB"},
		{Name: "FieldCC"},
		{Name: "FieldEEE", Fields: []result{{Name: "EE"}, {Name: "FF", Fields: []result{{Name: "GG"}}}}},
		{Name: "FlattenField"},
	}
	var checkEqual func(s *typeStruct, res []result) bool
	checkEqual = func(s *typeStruct, res []result) bool {
		if len(s.fields) != len(res) {
			return false
		}
		for i, field := range s.fields {
			if field.Name != res[i].Name {
				t.Log(field.Name, res[i].Name)
				return false
			}
			s := getCacheStruct(field.Type, nil)
			if s != notStructType || len(res[i].Fields) > 0 {
				if !checkEqual(s, res[i].Fields) {
					t.Log(field, s, res[i].Name, res[i].Fields)
					return false
				}
			}
		}
		return true
	}
	if !checkEqual(s, res) {
		t.Fatal(*s, res)
	}
	type TypeD struct {
		FieldBB string
	}
	type TypeE struct {
		TypeD
		TypeB
	}
	s = getCacheStruct(reflect.TypeOf(TypeE{}), nil)
	assert.Equal(t, s.err, fmt.Errorf("ambiguous field FieldBB"))

	type TypeF struct {
		FieldD string
		FieldE string `convertor:"FieldD"`
	}
	s = getCacheStruct(reflect.TypeOf(TypeF{}), nil)
	assert.Equal(t, s.err, fmt.Errorf("conflict field name and tag: FieldD"))
	type TypeAA struct {
		FieldA  *TypeAA
		FieldBB string
		*TypeAA
	}
	s = getCacheStruct(reflect.TypeOf(TypeAA{}), nil)
	assert.Equal(t, s.err, fmt.Errorf("circle struct rely: %s", reflect.TypeOf(&TypeAA{})))
}

func TestConvert(t *testing.T) {
	type TypeD struct {
		FieldDD int64
	}
	type TypeB struct {
		FieldBB string
		FieldDD int
	}
	type TypeC struct {
		FieldBB string
		TypeD
	}
	var a = TypeB{FieldBB: "aaa", FieldDD: 10}
	var b = &TypeC{}
	err := Convert(a, b)
	assert.Nil(t, err)
	assert.Equal(t, a.FieldBB, b.FieldBB)
	assert.EqualValues(t, a.FieldDD, b.FieldDD)
}

func TestAllConvertRule(t *testing.T) {
	type TypeB struct {
		FieldBB string
		FieldCC int
	}

	type TypeC struct {
		XX float32
	}

	type TypeD struct {
		DD float64
		aa int
	}

	type TypeE struct {
		EE int
		FF struct {
			GG uint32
		}
	}

	type TypeA struct {
		TypeB
		FieldBB       string
		FieldC        TypeD  `convertor:"CCCC"`
		DDD           string `convertor:"-"`
		bb            TypeC
		EEE           *TypeE
		TypeC         `convertor:"FF"`
		dd            string
		FlattenField  string
		Slice         []TypeC
		PtrSlice      []*TypeC
		ValToPtrSlice []TypeC
		NilSlice      []TypeC
	}

	type TypeAA struct {
		*TypeB
		CCCC struct {
			DD float32
		}
		EEE struct {
			EE uint16
			FF *struct {
				GG int
			}
		}
		FFF struct {
			XX float64
		} `convertor:"FF"`
		Flatten struct {
			FlattenField string
		} `convertor:"+"`
		Slice []struct {
			XX float32
		}
		PtrSlice []*struct {
			XX float32
		}
		ValToPtrSlice []*struct {
			XX float32
		}
		NilSlice []struct {
			XX float32
		}
	}
	var a = TypeA{
		TypeB:   TypeB{FieldBB: "second level BB", FieldCC: 1234},
		FieldBB: "first level BB",
		FieldC:  TypeD{DD: 1.35},
		DDD:     "first level DDD",
		EEE:     &TypeE{EE: 10},
		TypeC: TypeC{
			XX: 1.432,
		},
		FlattenField:  "flatten field",
		Slice:         []TypeC{{XX: 1.2}, {XX: 1.3}},
		PtrSlice:      []*TypeC{nil, {XX: 1.2}},
		ValToPtrSlice: []TypeC{{XX: 1.2}, {}},
		NilSlice:      nil,
	}
	var b = &TypeAA{}
	ass := assert.New(t)
	err := Convert(a, b)
	ass.Nil(err)
	var equal3 = func(a, b, c interface{}) {
		ass.EqualValues(a, b)
		ass.EqualValues(a, c)
	}
	equal3(a.FieldBB, b.FieldBB, "first level BB")
	equal3(a.FieldCC, b.FieldCC, 1234)
	equal3(a.FieldC.DD, b.CCCC.DD, 1.35)
	equal3(a.EEE.EE, b.EEE.EE, 10)
	equal3(a.EEE.FF.GG, b.EEE.FF.GG, 0)
	ass.EqualValues(b.FFF.XX, 1)
	equal3(a.FlattenField, b.Flatten.FlattenField, "flatten field")
	equal3(len(a.Slice), len(b.Slice), 2)
	equal3(a.Slice[0].XX, b.Slice[0].XX, float32(1.2))
	equal3(a.Slice[1].XX, b.Slice[1].XX, float32(1.3))
	equal3(len(a.PtrSlice), len(b.PtrSlice), 2)
	if !(a.PtrSlice[0] == nil && b.PtrSlice[0] == nil) {
		t.Fatal(a.PtrSlice, b.PtrSlice)
	}
	equal3(a.PtrSlice[1].XX, b.PtrSlice[1].XX, float32(1.2))
	equal3(len(a.ValToPtrSlice), len(b.ValToPtrSlice), 2)
	equal3(a.ValToPtrSlice[0].XX, b.ValToPtrSlice[0].XX, float32(1.2))
	equal3(a.ValToPtrSlice[1].XX, b.ValToPtrSlice[1].XX, float32(0))
	if !(a.NilSlice == nil && b.NilSlice == nil) {
		t.Fatal(a.NilSlice, b.NilSlice)
	}
}

func TestConvertFunc(t *testing.T) {
	type TypeA struct {
		FieldA string
	}
	type TypeB struct {
		FieldB int64
	}
	type TypeCC struct {
		Field TypeA
	}
	type TypeDD struct {
		Field TypeB
	}
	type TypeEE struct {
		Field  TypeA
		Field1 TypeB
	}
	type TypeFF struct {
		Field1 TypeB
	}
	err := Convert(TypeCC{}, &TypeDD{})
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "dest has no field to receive src field FieldA(string)")
	err = Convert(TypeDD{}, &TypeEE{})
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "src has no field FieldA(string) convert to dest")
	err = Convert(TypeEE{}, &TypeCC{})
	assert.Equal(t, err.Error(), "dest has no field to receive src field Field1(convertor.TypeB)")
	err = Convert(TypeCC{}, &TypeEE{})
	assert.Equal(t, err.Error(), "src has no field Field1(convertor.TypeB) convert to dest")
	err = DestNotExistFieldIgnoreConvertor.Convert(TypeEE{}, &TypeFF{})
	assert.Nil(t, err)
	err = SrcNotExistFieldIgnoreConvertor.Convert(TypeFF{}, &TypeEE{})
	assert.Nil(t, err)
	err = DestNotExistFieldIgnoreConvertor.Convert(TypeEE{}, &TypeCC{})
	assert.Nil(t, err)
	err = SrcNotExistFieldIgnoreConvertor.Convert(TypeCC{}, &TypeEE{})
	assert.Nil(t, err)
	RegisterConvertFunc(func(a TypeA, b *TypeB) (err error) {
		b.FieldB, err = strconv.ParseInt(a.FieldA, 10, 64)
		return
	})
	var d = &TypeDD{}
	err = Convert(TypeCC{Field: TypeA{FieldA: "100"}}, d)
	assert.Nil(t, err)
	assert.EqualValues(t, d.Field.FieldB, 100)
	d = nil
	err = Convert(TypeCC{}, d)
	assert.Equal(t, err, ErrNilDestination)
	err = Convert(TypeCC{}, TypeDD{})
	assert.Equal(t, err, ErrDestinationNotPointer)
	i := new(int)
	err = Convert("aaa", i)
	assert.Equal(t, err, fmt.Errorf("type %s is not convertiable to type %s", reflect.TypeOf(""), reflect.TypeOf(i)))
}

func TestOption(t *testing.T) {
	type SrcType struct {
		Field string
	}
	type DestType struct {
		Field1 []byte
	}
	convertor, err := NewConvertor(
		OptionConvertFunc(func(src SrcType, dest *DestType) error {
			dest.Field1 = []byte(src.Field)
			return nil
		}),
	)
	assert.Nil(t, err)
	dest := &DestType{}
	err = convertor.Convert(SrcType{"Src Field"}, dest)
	assert.Nil(t, err)
	assert.Equal(t, dest.Field1, []byte("Src Field"))
}

type People struct {
	firstName string
	lastName  string
}

func (p People) FullName() string {
	return p.firstName + " " + p.lastName
}

type Peopler interface {
	FullName() string
}

func TestConvertInterface(t *testing.T) {
	type TypeA struct {
		P People
	}
	type TypeB struct {
		P Peopler
	}
	b := &TypeB{}
	err := Convert(TypeA{P: People{firstName: "aaa", lastName: "bbb"}}, b)
	assert.Nil(t, err)
	assert.Equal(t, b.P.FullName(), "aaa bbb")
	a := &TypeA{}
	err = Convert(*b, a)
	assert.Equal(t, err, fmt.Errorf("type convertor.Peopler is not convertiable to type *convertor.People"))
}

func TestRegisterConvertorFuncError(t *testing.T) {
	assert.Equal(t, BadConvertFuncNotFunc, registerConvertFunc(nil, 1))
	assert.Equal(t, BadConvertFuncInCount, registerConvertFunc(nil, func() {}))
	assert.Equal(t, BadConvertFuncSrcTypeIsPointer, registerConvertFunc(nil, func(src *int, dest *float64) error { return nil }))
	assert.Equal(t, BadConvertFuncDestTypeNotPointer, registerConvertFunc(nil, func(src int, dest float64) error { return nil }))
	assert.Equal(t, BadConvertFuncOut, registerConvertFunc(nil, func(src int, dest *float64) {}))
}

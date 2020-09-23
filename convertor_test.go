package convertor

import (
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
	s := getCacheStruct(reflect.TypeOf(TypeA{}))
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
			s := getCacheStruct(field.Type)
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
	s = getCacheStruct(reflect.TypeOf(TypeE{}))
	assert.Equal(t, s.err, ErrAmbiguousField)

	type TypeF struct {
		FieldD string
		FieldE string `convertor:"FieldD"`
	}
	s = getCacheStruct(reflect.TypeOf(TypeF{}))
	assert.Equal(t, s.err, ErrConflictFieldNameAndTag)
	type TypeAA struct {
		FieldA  *TypeAA
		FieldBB string
		*TypeAA // self loop Anonymous will be ignored
	}
	s = getCacheStruct(reflect.TypeOf(TypeAA{}))
	t.Log(s)
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
			GG int
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
		Array         []TypeC
		PtrArray      []*TypeC
		ValToPtrArray []TypeC
		NilArray      []TypeC
	}

	type TypeAA struct {
		*TypeB
		CCCC struct {
			DD float32
		}
		EEE struct {
			EE int
			FF *struct {
				GG int
			}
		}
		FFF struct {
			XX int
		} `convertor:"FF"`
		Flatten struct {
			FlattenField string
		} `convertor:"+"`
		Array []struct {
			XX float32
		}
		PtrArray []*struct {
			XX float32
		}
		ValToPtrArray []*struct {
			XX float32
		}
		NilArray []struct {
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
		Array:         []TypeC{{XX: 1.2}, {XX: 1.3}},
		PtrArray:      []*TypeC{nil, {XX: 1.2}},
		ValToPtrArray: []TypeC{{XX: 1.2}, {}},
		NilArray:      nil,
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
	equal3(len(a.Array), len(b.Array), 2)
	equal3(a.Array[0].XX, b.Array[0].XX, float32(1.2))
	equal3(a.Array[1].XX, b.Array[1].XX, float32(1.3))
	equal3(len(a.PtrArray), len(b.PtrArray), 2)
	if !(a.PtrArray[0] == nil && b.PtrArray[0] == nil) {
		t.Fatal(a.PtrArray, b.PtrArray)
	}
	equal3(a.PtrArray[1].XX, b.PtrArray[1].XX, float32(1.2))
	equal3(len(a.ValToPtrArray), len(b.ValToPtrArray), 2)
	equal3(a.ValToPtrArray[0].XX, b.ValToPtrArray[0].XX, float32(1.2))
	equal3(a.ValToPtrArray[1].XX, b.ValToPtrArray[1].XX, float32(0))
	if !(a.NilArray == nil && b.NilArray == nil) {
		t.Fatal(a.NilArray, b.NilArray)
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
	err := Convert(TypeCC{}, &TypeDD{})
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "type convertor.TypeA has not same fields name with type *convertor.TypeB")
	err = Convert(TypeDD{}, &TypeEE{})
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "type convertor.TypeDD has not same fields len with type *convertor.TypeEE")
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
	assert.Equal(t, err, ErrNotConvertible)
}

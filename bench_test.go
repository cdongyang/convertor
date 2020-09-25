package convertor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cdongyang/convertor"
)

func BenchmarkConvertSlice(b *testing.B) {
	type SliceInnerType struct {
		FieldA int
		FieldB float64
	}
	type SliceTypeA struct {
		Slice []SliceInnerType
	}
	type SliceTypeB struct {
		Slice []struct {
			FieldA int64
			FieldB *float64
		}
	}

	var aa = SliceTypeA{
		Slice: []SliceInnerType{
			{
				FieldA: 10,
				FieldB: 1.2,
			},
			{
				FieldA: 20,
				FieldB: 1.3,
			},
		},
	}
	var bb = &SliceTypeB{}
	var ass = func(b *testing.B) {
		assert.Equal(b, len(aa.Slice), len(bb.Slice))
		for i := 0; i < len(aa.Slice); i++ {
			assert.EqualValues(b, aa.Slice[i].FieldA, bb.Slice[i].FieldA)
			assert.EqualValues(b, aa.Slice[i].FieldB, *bb.Slice[i].FieldB)
		}
	}
	b.Run("Convert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			*bb = SliceTypeB{}
			if err := convertor.Convert(&aa, bb); err != nil {
				b.Fatal(err)
			}
		}
		ass(b)
	})
	b.Run("ConvertNative", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			*bb = SliceTypeB{
				Slice: make([]struct {
					FieldA int64
					FieldB *float64
				}, len(aa.Slice)),
			}
			for i, elem := range aa.Slice {
				bb.Slice[i].FieldA = int64(elem.FieldA)
				bb.Slice[i].FieldB = new(float64)
				*bb.Slice[i].FieldB = elem.FieldB
			}
		}
		ass(b)
	})
}

func BenchmarkConvertStruct(b *testing.B) {
	type TypeA struct {
		FieldA int
		FieldB float32
	}

	type TypeB struct {
		FieldA int
		FieldB float32
	}

	var aa = TypeA{
		FieldA: 10,
		FieldB: 1.2,
	}
	var bb = &TypeB{}
	b.Run("Convert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			*bb = TypeB{}
			if err := convertor.Convert(&aa, bb); err != nil {
				b.Fatal(err)
			}
		}
		assert.Equal(b, aa.FieldA, bb.FieldA)
		assert.Equal(b, aa.FieldB, bb.FieldB)
	})
	b.Run("ConvertFunc", func(b *testing.B) {
		customConvertor, err := convertor.NewConvertor(
			convertor.OptionConvertFunc(func(a TypeA, b *TypeB) error {
				*b = TypeB{FieldA: a.FieldA + 1, FieldB: 1.3}
				return nil
			}),
		)
		assert.Nil(b, err)
		for i := 0; i < b.N; i++ {
			*bb = TypeB{}
			if err := customConvertor.Convert(&aa, bb); err != nil {
				b.Fatal(err)
			}
		}
		assert.Equal(b, bb.FieldA, 11)
		assert.Equal(b, bb.FieldB, float32(1.3))
	})
	b.Run("ConvertNative", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			*bb = TypeB(aa)
		}
		assert.Equal(b, aa.FieldA, bb.FieldA)
		assert.Equal(b, aa.FieldB, bb.FieldB)
	})
}

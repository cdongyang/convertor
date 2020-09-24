package convertor

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

const (
	convertorTag = "convertor"
)

type convertFuncsType map[[2]reflect.Type]reflect.Value

var (
	cacheFields  sync.Map
	convertFuncs = convertFuncsType{} // global convert func
	errType      = reflect.TypeOf((*error)(nil)).Elem()
)

// RegisterConvertFunc register convert function like func (src SrcType, dest DestType) error
// DestType must be pointer
// concurrent unsafe, just register in main func, and it will panic if it's a bad convert func
func RegisterConvertFunc(f interface{}) {
	if err := registerConvertFunc(convertFuncs, f); err != nil {
		panic(err)
	}
}

func registerConvertFunc(convertFuncs convertFuncsType, f interface{}) error {
	val := reflect.ValueOf(f)
	if val.Type().Kind() != reflect.Func {
		return errors.New("bad convertor func")
	}
	if val.Type().NumIn() != 2 {
		return errors.New("bad convertor func in count")
	}
	if val.Type().In(1).Kind() != reflect.Ptr {
		return errors.New("convertor func dest type should be pointer")
	}
	if val.Type().NumOut() != 1 || !isErrorType(val.Type().Out(0)) {
		return errors.New("bad convertor func out")
	}
	convertFuncs[[2]reflect.Type{val.Type().In(0), val.Type().In(1)}] = val
	return nil
}

func isErrorType(typ reflect.Type) bool {
	return typ.Implements(errType)
}

/*
TO consider:
interface assign
*/
type typeStruct struct {
	err    error
	fields []typeField
}

type typeField struct {
	Type       reflect.Type
	Name       string
	Idx        int
	NextIdx    int
	NextStruct *typeStruct
}

var (
	ErrAmbiguousField          = errors.New("ambiguous field")
	ErrConflictFieldNameAndTag = errors.New("conflict field name and tag")
	ErrNotConvertible          = errors.New("not convertible")
	ErrDestinationNotPointer   = errors.New("destination value is not pointer")
	ErrNilDestination          = errors.New("nil destination")
	ErrCircleStructRely        = errors.New("circle struct rely")
	notStructType              = &typeStruct{}
)

func getCacheStruct(typ reflect.Type, typePath map[reflect.Type]bool) (finalTypeStruct *typeStruct) {
	if val, ok := cacheFields.Load(typ); ok {
		return val.(*typeStruct)
	}
	if typePath == nil {
		typePath = map[reflect.Type]bool{}
	}
	if typePath[typ] { // prevent cirle struct rely
		return &typeStruct{
			err: ErrCircleStructRely,
		}
	}
	typePath[typ] = true
	originType := typ
	defer func() {
		sort.Slice(finalTypeStruct.fields, func(i, j int) bool {
			return finalTypeStruct.fields[i].Name < finalTypeStruct.fields[j].Name
		})
		cacheFields.Store(originType, finalTypeStruct)
		typePath[typ] = false
	}()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return notStructType
	}
	finalFields := make([]typeField, 0, typ.NumField())
	nameMap := map[string]bool{}
	var anonymousStructField []reflect.StructField
	var anonymousStructFieldIndex []int
	tmpVal := reflect.New(typ).Elem()
	for i := 0; i < typ.NumField(); i++ {
		if !tmpVal.Field(i).CanSet() { // unexport
			continue
		}
		field := typ.Field(i)
		tf := typeField{
			Type:    field.Type,
			Name:    field.Name,
			Idx:     i,
			NextIdx: -1,
		}
		// use convertor tag to cover field name
		tag, ok := field.Tag.Lookup(convertorTag)
		if ok {
			if tag == "-" { // ignore field
				continue
			}
			tf.Name = tag
		}
		if (field.Anonymous && !ok) || (ok && tag == "+") { // anonymous field has not tag, flatten later
			anonymousStructField = append(anonymousStructField, field)
			anonymousStructFieldIndex = append(anonymousStructFieldIndex, i)
			continue
		}
		finalFields = append(finalFields, tf)
		if nameMap[tf.Name] {
			return &typeStruct{
				err: ErrConflictFieldNameAndTag,
			}
		}
		nameMap[tf.Name] = true
	}
	var allAnonFields []typeField
	for i, field := range anonymousStructField {
		ftStruct := getCacheStruct(field.Type, typePath)
		if ftStruct.err != nil {
			return ftStruct
		}
		if inFields(ftStruct.fields, allAnonFields) { // two anonymous field has same sub field
			return &typeStruct{
				err: ErrAmbiguousField,
			}
		}
		allAnonFields = append(allAnonFields, ftStruct.fields...)
		for _, subField := range ftStruct.fields {
			if !nameMap[subField.Name] {
				nameMap[subField.Name] = true
				finalFields = append(finalFields, typeField{
					Type:       subField.Type,
					Name:       subField.Name,
					Idx:        anonymousStructFieldIndex[i],
					NextStruct: ftStruct,
					NextIdx:    subField.Idx,
				})
			}
		}
	}
	return &typeStruct{
		fields: finalFields,
	}
}

func inFields(sub, full []typeField) bool {
	for _, subField := range sub {
		for _, fullField := range full {
			if subField.Name == fullField.Name {
				return true
			}
		}
	}
	return false
}

type Options struct {
	convertFuncs            convertFuncsType
	srcNotExistFieldIgnore  bool
	destNotExistFieldIgnore bool
}

type Option func(*Options) error

type convertor struct {
	opts Options
}

type Convertor interface {
	Convert(src, dest interface{}) error
}

func OptionConvertFunc(f interface{}) Option {
	return func(opts *Options) error {
		return registerConvertFunc(opts.convertFuncs, f)
	}
}

func OptionSrcNotExistFieldIgnore() Option {
	return func(opts *Options) error {
		opts.srcNotExistFieldIgnore = true
		return nil
	}
}

func OptionDestNotExistFieldIgnore() Option {
	return func(opts *Options) error {
		opts.destNotExistFieldIgnore = true
		return nil
	}
}

func NewConvertor(opts ...Option) (Convertor, error) {
	c := &convertor{}
	c.opts.convertFuncs = convertFuncsType{}
	for _, o := range opts {
		if err := o(&c.opts); err != nil {
			return nil, err
		}
	}
	return c, nil
}

/*
Convert convert struct src to dest
Example:
	type D struct {
		DD int64
	}
	type B struct {
		BB string
		DD int
	}
	type C struct {
		BB string
		D
	}
	var a = B{}
	var b = &C{}
	Convert(a,b)
*/

var DefaultConvertor, _ = NewConvertor()
var SrcNotExistFieldIgnoreConvertor, _ = NewConvertor(OptionSrcNotExistFieldIgnore())
var DestNotExistFieldIgnoreConvertor, _ = NewConvertor(OptionDestNotExistFieldIgnore())

func Convert(src, dest interface{}) error {
	return DefaultConvertor.Convert(src, dest)
}

func (c *convertor) Convert(src, dest interface{}) (err error) {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return ErrDestinationNotPointer
	}
	if destVal.IsNil() {
		return ErrNilDestination
	}
	return c.convert(reflect.ValueOf(src), destVal)
}

func (c *convertor) convert(src, dest reflect.Value) error {
	convertFuncKey := [2]reflect.Type{indirect(src).Type(), dest.Type()}
	convertFunc, ok := c.opts.convertFuncs[convertFuncKey]
	if !ok {
		convertFunc, ok = convertFuncs[convertFuncKey]
	}
	if ok {
		out := convertFunc.Call([]reflect.Value{indirect(src), dest})
		if err, ok := out[0].Interface().(error); ok {
			return err
		}
		return nil
	}
	if indirect(src).Type().AssignableTo(indirect(dest).Type()) {
		indirect(dest).Set(indirect(src))
		return nil
	}
	if indirect(src).Type().ConvertibleTo(indirect(dest).Type()) {
		indirect(dest).Set(indirect(src).Convert(indirect(dest).Type()))
		return nil
	}
	srcStruct := getCacheStruct(src.Type(), nil)
	destStruct := getCacheStruct(dest.Type(), nil)
	if srcStruct.err != nil {
		return srcStruct.err
	}
	if destStruct.err != nil {
		return destStruct.err
	}
	if indirect(src).Kind() == reflect.Slice && indirect(dest).Kind() == reflect.Slice {
		if src.IsNil() {
			return nil
		}
		src = indirect(src)
		dest = indirect(dest)
		dest.Set(reflect.MakeSlice(dest.Type(), src.Len(), src.Cap()))
		for i := 0; i < src.Len(); i++ {
			srcElem := src.Index(i)
			if srcElem.Kind() == reflect.Ptr && srcElem.IsNil() {
				continue
			}
			destElem := dest.Index(i)
			if destElem.Kind() == reflect.Ptr && destElem.IsNil() {
				destElem.Set(reflect.New(destElem.Type().Elem()))
			}
			if destElem.Kind() != reflect.Ptr && destElem.CanAddr() {
				destElem = destElem.Addr()
			}
			if err := c.convert(src.Index(i), destElem); err != nil {
				return err
			}
		}
		return nil
	}
	if indirect(src).Kind() != reflect.Struct || indirect(dest).Kind() != reflect.Struct {
		return ErrNotConvertible
	}
	srcFields := srcStruct.fields
	destFields := destStruct.fields
	var i, j int
	for i < len(srcFields) && j < len(destFields) {
		if srcFields[i].Name != destFields[j].Name {
			var err error
			if srcFields[i].Name < destFields[j].Name {
				if c.opts.destNotExistFieldIgnore {
					i++
					continue
				}
				err = fmt.Errorf("dest has no field to receive src field %s(%v)", srcFields[i].Name, srcFields[i].Type)
			} else {
				if c.opts.srcNotExistFieldIgnore {
					j++
					continue
				}
				err = fmt.Errorf("src has no field %s(%v) convert to dest", destFields[j].Name, destFields[j].Type)
			}
			return err
		}
		val := getValueByPath(src, srcFields[i])
		if val == zeroValue || (val.Kind() == reflect.Ptr && val.IsNil()) {
			i++
			j++
			continue
		}
		if err := c.setValueByPath(dest, val, destFields[j]); err != nil {
			return err
		}
		i++
		j++
	}
	if i < len(srcFields) && !c.opts.destNotExistFieldIgnore {
		return fmt.Errorf("dest has no field to receive src field %s(%v)", srcFields[i].Name, srcFields[i].Type)
	}
	if j < len(destFields) && !c.opts.srcNotExistFieldIgnore {
		return fmt.Errorf("src has no field %s(%v) convert to dest", destFields[j].Name, destFields[j].Type)
	}
	return nil
}

func indirect(val reflect.Value) reflect.Value {
	return reflect.Indirect(val)
}

var zeroValue = reflect.Value{}

func getValueByPath(val reflect.Value, field typeField) reflect.Value {
	for {
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return zeroValue
			}
			val = val.Elem()
		}
		val = val.Field(field.Idx)
		if field.NextStruct == nil {
			break
		}
		field = field.NextStruct.fields[field.NextIdx]
	}
	return val
}

func (c *convertor) setValueByPath(dest, val reflect.Value, field typeField) error {
	for {
		if dest.Kind() == reflect.Ptr {
			if dest.IsNil() {
				dest.Set(reflect.New(dest.Type().Elem()))
			}
			dest = dest.Elem()
		}
		//log.Println(dest.Type(), field.Idx, val.Type(), val.Type().String(), val.Type().PkgPath())
		dest = dest.Field(field.Idx)
		if field.NextStruct == nil {
			break
		}
		field = field.NextStruct.fields[field.NextIdx]
	}
	if dest.Kind() == reflect.Ptr {
		if dest.IsNil() {
			dest.Set(reflect.New(dest.Type().Elem()))
		}
	}
	if dest.Kind() != reflect.Ptr && dest.CanAddr() {
		dest = dest.Addr()
	}
	return c.convert(val, dest)
}

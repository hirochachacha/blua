package reflect

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/hirochachacha/plua/object"
)

// interface{} (Go) -> Value (Lua)
func ValueOf(x interface{}) object.Value {
	switch x := x.(type) {
	case nil:
		return nil
	case bool:
		return object.Boolean(x)
	case int:
		return object.Integer(x)
	case int8:
		return object.Integer(x)
	case int32:
		return object.Integer(x)
	case int64:
		return object.Integer(x)
	case float32:
		return object.Number(x)
	case float64:
		return object.Number(x)
	case string:
		return object.String(x)
	case unsafe.Pointer:
		return object.LightUserdata{Pointer: x}
	case object.Integer:
		return x
	case object.Number:
		return x
	case object.String:
		return x
	case object.Boolean:
		return x
	case object.LightUserdata:
		return x
	case object.GoFunction:
		return x
	case *object.Userdata:
		return x
	case object.Table:
		return x
	case object.Closure:
		return x
	case object.Thread:
		return x
	}

	return valueOfReflect(reflect.ValueOf(x), true)
}

var (
	tGoBool    = reflect.TypeOf(false)
	tGoInt     = reflect.TypeOf(int(0))
	tGoInt8    = reflect.TypeOf(int8(0))
	tGoInt16   = reflect.TypeOf(int16(0))
	tGoInt32   = reflect.TypeOf(int32(0))
	tGoInt64   = reflect.TypeOf(int64(0))
	tGoFloat32 = reflect.TypeOf(float32(0))
	tGoFloat64 = reflect.TypeOf(float64(0))
	tGoString  = reflect.TypeOf("")
	tGoPointer = reflect.TypeOf(unsafe.Pointer(nil))

	tValue = reflect.TypeOf((*object.Value)(nil)).Elem()

	tBoolean       = reflect.TypeOf(object.False)
	tInteger       = reflect.TypeOf(object.Integer(0))
	tNumber        = reflect.TypeOf(object.Number(0))
	tString        = reflect.TypeOf(object.String(""))
	tLightUserdata = reflect.TypeOf(object.LightUserdata{})
	tGoFunction    = reflect.TypeOf(object.GoFunction(nil))
	tUserdataPtr   = reflect.TypeOf((*object.Userdata)(nil))

	tTable   = reflect.TypeOf((*object.Table)(nil)).Elem()
	tClosure = reflect.TypeOf((*object.Closure)(nil)).Elem()
	tThread  = reflect.TypeOf((*object.Thread)(nil)).Elem()
)

var (
	boolMT    object.Table
	intMT     object.Table
	floatMT   object.Table
	stringMT  object.Table
	uintMT    object.Table
	complexMT object.Table
	arrayMT   object.Table
	chanMT    object.Table
	funcMT    object.Table
	ifaceMT   object.Table
	mapMT     object.Table
	ptrMT     object.Table
	sliceMT   object.Table
	structMT  object.Table
)

var (
	boolOnce    sync.Once
	intOnce     sync.Once
	floatOnce   sync.Once
	stringOnce  sync.Once
	uintOnce    sync.Once
	complexOnce sync.Once
	arrayOnce   sync.Once
	chanOnce    sync.Once
	funcOnce    sync.Once
	ifaceOnce   sync.Once
	mapOnce     sync.Once
	ptrOnce     sync.Once
	sliceOnce   sync.Once
	structOnce  sync.Once
)

// reflect.Value (Go) -> Value (Lua)
func valueOfReflect(rval reflect.Value, skipPrimitive bool) object.Value {
	if !skipPrimitive {
		typ := rval.Type()
		if typ == tValue {
			if rval.IsNil() {
				return nil
			}

			rval = rval.Elem()
			typ = rval.Type()
		}

		switch typ {
		case tGoBool:
			return object.Boolean(rval.Bool())
		case tGoInt, tGoInt8, tGoInt16, tGoInt32, tGoInt32:
			return object.Integer(rval.Int())
		case tGoFloat32, tGoFloat64:
			return object.Number(rval.Float())
		case tGoString:
			return object.String(rval.String())
		case tGoPointer:
			return object.LightUserdata{Pointer: unsafe.Pointer(rval.Pointer())}
		case tBoolean:
			return rval.Interface().(object.Boolean)
		case tInteger:
			return rval.Interface().(object.Integer)
		case tNumber:
			return rval.Interface().(object.Number)
		case tString:
			return rval.Interface().(object.String)
		case tLightUserdata:
			return rval.Interface().(object.LightUserdata)
		case tGoFunction:
			return rval.Interface().(object.GoFunction)
		case tUserdataPtr:
			return rval.Interface().(*object.Userdata)
		default:
			switch {
			case typ.Implements(tTable):
				return rval.Interface().(object.Table)
			case typ.Implements(tClosure):
				return rval.Interface().(object.Closure)
			case typ.Implements(tThread):
				return rval.Interface().(object.Thread)
			}
		}
	}

	ud := &object.Userdata{Value: rval}

	switch kind := rval.Kind(); kind {
	case reflect.Bool:
		boolOnce.Do(buildBoolMT)

		ud.Metatable = boolMT
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intOnce.Do(buildIntMT)

		ud.Metatable = intMT
	case reflect.Float32, reflect.Float64:
		floatOnce.Do(buildFloatMT)

		ud.Metatable = floatMT
	case reflect.String:
		stringOnce.Do(buildStringMT)

		ud.Metatable = stringMT
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		uintOnce.Do(buildUintMT)

		ud.Metatable = uintMT
	case reflect.Complex64, reflect.Complex128:
		complexOnce.Do(buildComplexMT)

		ud.Metatable = complexMT
	case reflect.Array:
		arrayOnce.Do(buildArrayMT)

		ud.Metatable = arrayMT
	case reflect.Chan:
		if rval.IsNil() {
			return nil
		}

		chanOnce.Do(buildChanMT)

		ud.Metatable = chanMT
	case reflect.Func:
		if rval.IsNil() {
			return nil
		}

		funcOnce.Do(buildFuncMT)

		ud.Metatable = funcMT
	case reflect.Interface:
		if rval.IsNil() {
			return nil
		}

		ifaceOnce.Do(buildIfaceMT)

		ud.Metatable = ifaceMT
	case reflect.Map:
		if rval.IsNil() {
			return nil
		}

		mapOnce.Do(buildMapMT)

		ud.Metatable = mapMT
	case reflect.Ptr:
		if rval.IsNil() {
			return nil
		}

		ptrOnce.Do(buildPtrMT)

		ud.Metatable = ptrMT
	case reflect.Slice:
		if rval.IsNil() {
			return nil
		}

		sliceOnce.Do(buildSliceMT)

		ud.Metatable = sliceMT
	case reflect.Struct:
		structOnce.Do(buildStructMT)

		ud.Metatable = structMT
	case reflect.UnsafePointer:
		return object.LightUserdata{Pointer: unsafe.Pointer(rval.Pointer())}
	case reflect.Invalid:
		return nil
	default:
		panic("unreachable")
	}

	return ud
}

// Value (Lua) -> reflect.Value (Go)
func toReflectValue(typ reflect.Type, val object.Value) reflect.Value {
	switch val := val.(type) {
	case nil:
		if typ == tValue {
			return reflect.Zero(typ)
		}

		switch typ.Kind() {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
			return reflect.Zero(typ)
		}
	case object.Boolean:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		if typ.Kind() == reflect.Bool {
			return reflect.ValueOf(val).Convert(typ)
		}
	case object.Integer:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		switch typ.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64:
			return reflect.ValueOf(val).Convert(typ)
		case reflect.String:
			return reflect.ValueOf(integerToString(val)).Convert(typ)
		}
	case object.Number:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		switch typ.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if ival, ok := numberToInteger(val); ok {
				return reflect.ValueOf(ival).Convert(typ)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if u, ok := numberToGoUint(val); ok {
				return reflect.ValueOf(u).Convert(typ)
			}
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(val).Convert(typ)
		case reflect.String:
			return reflect.ValueOf(numberToString(val)).Convert(typ)
		}
	case object.String:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		switch typ.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if ival, ok := stringToInteger(val); ok {
				return reflect.ValueOf(ival).Convert(typ)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if u, ok := stringToGoUint(val); ok {
				return reflect.ValueOf(u).Convert(typ)
			}
		case reflect.Float32, reflect.Float64:
			if nval, ok := stringToNumber(val); ok {
				return reflect.ValueOf(nval).Convert(typ)
			}
		case reflect.String:
			return reflect.ValueOf(val).Convert(typ)
		}
	case object.LightUserdata:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		if typ.Kind() == reflect.UnsafePointer {
			return reflect.ValueOf(val.Pointer).Convert(typ)
		}
	case object.GoFunction:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		rval := reflect.ValueOf(val)
		rtyp := rval.Type()

		if rtyp == typ {
			return rval
		}
	case *object.Userdata:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		if rval, ok := val.Value.(reflect.Value); ok {
			rtyp := rval.Type()

			if rtyp == typ {
				return rval
			}
		} else {
			rval := reflect.ValueOf(val.Value)
			rtyp := rval.Type()

			if rtyp == typ {
				return rval
			}
		}
	case object.Table:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		rval := reflect.ValueOf(val)
		rtyp := rval.Type()

		if rtyp == typ {
			return rval
		}
	case object.Closure:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		rval := reflect.ValueOf(val)
		rtyp := rval.Type()

		if rtyp == typ {
			return rval
		}
	case object.Thread:
		if typ == tValue {
			return reflect.ValueOf(val)
		}

		rval := reflect.ValueOf(val)
		rtyp := rval.Type()

		if rtyp == typ {
			return rval
		}
	default:
		panic("unreachable")
	}

	return reflect.ValueOf(nil)
}

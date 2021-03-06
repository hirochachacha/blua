package object

type Table interface {
	Value

	// Sequence APIs
	Len() int
	SetList(base int, src []Value)

	// Map APIs
	Get(key Value) Value
	Set(Key, val Value)
	Del(key Value)
	Next(key Value) (nkey, nval Value, ok bool)

	Metatable() Table
	SetMetatable(mt Table)
}

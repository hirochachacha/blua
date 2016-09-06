package goroutine

import (
	"github.com/hirochachacha/plua/object"
	"github.com/hirochachacha/plua/object/fnutil"
)

func NewChannel(th object.Thread, args ...object.Value) ([]object.Value, *object.RuntimeError) {
	ap := fnutil.NewArgParser(th, args)

	capacity, err := ap.OptGoInt(0, 0)
	if err != nil {
		return nil, err
	}

	if capacity < 0 {
		return nil, ap.ArgError(0, "capacity should not be negative")
	}

	return []object.Value{th.NewChannel(capacity)}, nil
}

func Wrap(th object.Thread, args ...object.Value) ([]object.Value, *object.RuntimeError) {
	ap := fnutil.NewArgParser(th, args)

	fn, err := ap.ToFunction(0)
	if err != nil {
		return nil, err
	}

	th1 := th.NewGoThread()

	th1.LoadFunc(fn)

	retfn := func(_ object.Thread, args1 ...object.Value) ([]object.Value, *object.RuntimeError) {
		rets, err := th1.Resume(args1...)
		if err != nil {
			return nil, err
		}

		return rets, nil
	}

	return []object.Value{object.GoFunction(retfn)}, nil
}

func Open(th object.Thread, args ...object.Value) ([]object.Value, *object.RuntimeError) {
	chanIndex := th.NewTableSize(0, 3)

	chanIndex.Set(object.String("recv"), object.GoFunction(ChanRecv))
	chanIndex.Set(object.String("send"), object.GoFunction(ChanSend))
	chanIndex.Set(object.String("close"), object.GoFunction(ChanClose))

	mt := th.NewTableSize(0, 2)

	mt.Set(object.String("__index"), chanIndex)
	mt.Set(object.String("__pairs"), object.GoFunction(ChanPairs))

	th.SetMetatable(th.NewChannel(0), mt)

	m := th.NewTableSize(0, 2)

	m.Set(object.String("newchannel"), object.GoFunction(NewChannel))
	m.Set(object.String("wrap"), object.GoFunction(Wrap))

	return []object.Value{m}, nil
}

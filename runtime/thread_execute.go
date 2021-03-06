package runtime

import (
	"fmt"

	"github.com/hirochachacha/plua/internal/arith"
	"github.com/hirochachacha/plua/internal/errors"
	"github.com/hirochachacha/plua/internal/version"
	"github.com/hirochachacha/plua/object"
	"github.com/hirochachacha/plua/opcode"
)

func (th *thread) initExecute(args []object.Value) (rets []object.Value, done bool) {
	ctx := th.context

	switch fn := ctx.stack[ctx.ci.base-1].(type) {
	case nil:
		panic("main function isn't loaded yet")
	case object.GoFunction:
		var err *object.RuntimeError

		old := th.stack[1]

		rets, err = th.docallGo(fn, args...)

		if err != nil {
			th.error(err)
		} else {
			ctx.status = object.THREAD_RETURN
		}

		th.stack[1] = old

		done = true
	case object.Closure:
		cl := fn.(*closure)

		ci := &ctx.ciStack[0]
		ci.closure = cl
		ci.top = ci.base + cl.MaxStackSize

		if !ctx.growStack(ci.top) {
			th.error(errors.StackOverflowError())
		}

		if len(args) > cl.NParams {
			if cl.IsVararg {
				ci.varargs = args[cl.NParams:]
			} else {
				ci.varargs = nil
			}
			for r := ci.base - 1 + len(args); r > ci.base-1+cl.NParams; r-- {
				ctx.stack[r] = nil
			}
			args = args[:cl.NParams]
		} else {
			for r := ci.base - 1 + cl.NParams; r > ci.base-1+len(args); r-- {
				ctx.stack[r] = nil
			}
		}
		copy(ctx.stack[ci.base:], args)
	default:
		panic("unreachable")
	}

	return
}

func (th *thread) execute() {
	defer close(th.resume)
	defer close(th.yield)

	args := <-th.resume

	rets, done := th.initExecute(args)
	if !done {
		rets = th.execute0()
	}

	if !th.context.isRoot() {
		panic("unexpected")
	}

	switch th.status {
	case object.THREAD_RETURN:
		th.yield <- rets
	case object.THREAD_ERROR:
	default:
		panic("unexpected")
	}
}

func (th *thread) doExecute(fn object.Value, args []object.Value, isHook bool) (rets []object.Value, err *object.RuntimeError) {
	th.pushContext(basicStackSize, isHook)

	th.loadfn(fn)

	rets, done := th.initExecute(args)
	if !done {
		rets = th.execute0()
	}

	ctx := th.popContext()

	switch ctx.status {
	case object.THREAD_RETURN:
		ctx.closeUpvals(0) // close all upvalues on this context

		return rets, nil
	case object.THREAD_ERROR:
		ctx.closeUpvals(0) // close all upvalues on this context

		return nil, ctx.err
	default:
		panic("unreachable")
	}
}

// execute with current context
func (th *thread) execute0() (rets []object.Value) {
	if th.depth >= version.MAX_VM_RECURSION {
		th.error(errors.StackOverflowError())

		return nil
	}

	ctx := th.context

	ctx.status = object.THREAD_RUNNING

	var inst opcode.Instruction

	ci := ctx.ci

	for {
		inst = ci.Code[ci.pc]

		if err := th.onInstruction(); err != nil {
			th.error(err)

			return nil
		}

		ci.pc++

		switch inst.OpCode() {
		case opcode.MOVE:
			ctx.setRA(inst, ctx.getRB(inst))
		case opcode.LOADK:
			ctx.setRA(inst, ctx.getKBx(inst))
		case opcode.LOADKX:
			extra := ci.Code[ci.pc]
			if extra.OpCode() != opcode.EXTRAARG {
				th.error(errors.InvalidByteCodeError())

				return nil
			}

			ctx.setRA(inst, ctx.getKAx(extra))

			ci.pc++
		case opcode.LOADBOOL:
			ctx.setRA(inst, object.Boolean(inst.B() != 0))
			if inst.C() != 0 {
				ci.pc++
			}
		case opcode.LOADNIL:
			a := inst.A()
			for i := 0; i <= inst.B(); i++ {
				ctx.setR(a+i, nil)
			}
		case opcode.GETUPVAL:
			ctx.setRA(inst, ctx.getUB(inst))
		case opcode.GETTABUP:
			t := ctx.getUB(inst)
			key := ctx.getRKC(inst)

			val, err := arith.CallGettable(th, t, key)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, val)
		case opcode.GETTABLE:
			t := ctx.getRB(inst)
			key := ctx.getRKC(inst)

			val, err := arith.CallGettable(th, t, key)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, val)
		case opcode.SETTABUP:
			t := ctx.getUA(inst)
			key := ctx.getRKB(inst)
			val := ctx.getRKC(inst)

			err := arith.CallSettable(th, t, key, val)
			if err != nil {
				th.error(err)

				return nil
			}
		case opcode.SETUPVAL:
			ctx.setUB(inst, ctx.getRA(inst))
		case opcode.SETTABLE:
			t := ctx.getRA(inst)
			key := ctx.getRKB(inst)
			val := ctx.getRKC(inst)

			err := arith.CallSettable(th, t, key, val)
			if err != nil {
				th.error(err)

				return nil
			}
		case opcode.NEWTABLE:
			asize := opcode.LogToInt(inst.B())
			msize := opcode.LogToInt(inst.C())

			t := newTableSize(asize, msize)

			ctx.setRA(inst, t)
		case opcode.SELF:
			a := inst.A()

			t := ctx.getRB(inst)
			key := ctx.getRKC(inst)

			val, err := arith.CallGettable(th, t, key)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setR(a+1, t)
			ctx.setR(a, val)
		case opcode.ADD:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			sum, err := arith.CallAdd(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, sum)
		case opcode.SUB:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			diff, err := arith.CallSub(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, diff)
		case opcode.MUL:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			prod, err := arith.CallMul(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, prod)
		case opcode.DIV:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			quo, err := arith.CallDiv(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, quo)
		case opcode.IDIV:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			quo, err := arith.CallIdiv(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, quo)
		case opcode.BAND:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			band, err := arith.CallBand(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, band)
		case opcode.BOR:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			bor, err := arith.CallBor(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, bor)
		case opcode.BXOR:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			bxor, err := arith.CallBxor(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, bxor)
		case opcode.SHL:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			shl, err := arith.CallShl(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, shl)
		case opcode.SHR:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			shr, err := arith.CallShr(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, shr)
		case opcode.MOD:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			rem, err := arith.CallMod(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, rem)
		case opcode.POW:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			prod, err := arith.CallPow(th, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, prod)
		case opcode.UNM:
			rb := ctx.getRB(inst)

			unm, err := arith.CallUnm(th, rb)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, unm)
		case opcode.BNOT:
			rb := ctx.getRB(inst)

			bnot, err := arith.CallBnot(th, rb)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, bnot)
		case opcode.NOT:
			rb := ctx.getRB(inst)

			ctx.setRA(inst, arith.Not(rb))
		case opcode.LEN:
			rb := ctx.getRB(inst)

			len, err := arith.CallLen(th, rb)
			if err != nil {
				th.error(err)

				return nil
			}

			ctx.setRA(inst, len)
		case opcode.CONCAT:
			if err := th.concat(inst.A(), inst.B(), inst.C()); err != nil {
				th.error(err)

				return nil
			}
		case opcode.JMP:
			th.dojmp(inst)
		case opcode.EQ:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			b, err := arith.CallEqual(th, inst.A() != 0, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			if b {
				ci.pc++
			} else {
				jmp := ci.Code[ci.pc]

				if jmp.OpCode() != opcode.JMP {
					th.error(errors.InvalidByteCodeError())

					return nil
				}

				ci.pc++

				th.dojmp(jmp)
			}
		case opcode.LT:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			b, err := arith.CallLessThan(th, inst.A() != 0, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			if b {
				ci.pc++
			} else {
				jmp := ci.Code[ci.pc]

				if jmp.OpCode() != opcode.JMP {
					th.error(errors.InvalidByteCodeError())

					return nil
				}

				ci.pc++

				th.dojmp(jmp)
			}
		case opcode.LE:
			rb := ctx.getRKB(inst)
			rc := ctx.getRKC(inst)

			b, err := arith.CallLessThanOrEqualTo(th, inst.A() != 0, rb, rc)
			if err != nil {
				th.error(err)

				return nil
			}

			if b {
				ci.pc++
			} else {
				jmp := ci.Code[ci.pc]

				if jmp.OpCode() != opcode.JMP {
					th.error(errors.InvalidByteCodeError())

					return nil
				}

				ci.pc++

				th.dojmp(jmp)
			}
		case opcode.TEST:
			ra := ctx.getRA(inst)

			if object.ToGoBool(ra) != (inst.C() != 0) {
				ci.pc++
			} else {
				jmp := ci.Code[ci.pc]

				if jmp.OpCode() != opcode.JMP {
					th.error(errors.InvalidByteCodeError())

					return nil
				}

				ci.pc++

				th.dojmp(jmp)
			}
		case opcode.TESTSET:
			rb := ctx.getRB(inst)

			if object.ToGoBool(rb) != (inst.C() != 0) {
				ci.pc++
			} else {
				ctx.setRA(inst, rb)

				jmp := ci.Code[ci.pc]

				if jmp.OpCode() != opcode.JMP {
					th.error(errors.InvalidByteCodeError())

					return nil
				}

				ci.pc++

				th.dojmp(jmp)
			}
		case opcode.CALL:
			a := inst.A()

			nargs := inst.B() - 1
			nrets := inst.C() - 1

			if nargs == -1 {
				nargs = ci.top - ci.base - a - 1
			}

			if err := th.call(a, nargs, nrets); err != nil {
				th.error(err)

				return nil
			}
			ci = ctx.ci
		case opcode.TAILCALL:
			a := inst.A()

			nargs := inst.B() - 1

			if nargs == -1 {
				nargs = ci.top - ci.base - a - 1
			}

			if err := th.tailcall(a, nargs); err != nil {
				th.error(err)

				return nil
			}
			ci = ctx.ci
		case opcode.RETURN:
			a := inst.A()

			nrets := inst.B() - 1

			if nrets == -1 {
				nrets = ci.top - ci.base - a
			}

			if rets, exit := th.returnLua(a, nrets); exit {
				return rets
			}

			ci = ctx.ci
		case opcode.FORLOOP:
			a := inst.A()
			ra := ctx.getR(a)
			ra1 := ctx.getR(a + 1)
			ra2 := ctx.getR(a + 2)

			// forprep already convert val to integer or number.
			// so there are no need to check types.
			if idx, ok := ra.(object.Integer); ok {
				limit := ra1.(object.Integer)
				step := ra2.(object.Integer)
				idx += step
				if 0 < step {
					if idx <= limit {
						ci.pc += inst.SBx()
						ctx.setR(a, idx)
						ctx.setR(a+3, idx)

						break
					}
				} else {
					if idx >= limit {
						ci.pc += inst.SBx()
						ctx.setR(a, idx)
						ctx.setR(a+3, idx)

						break
					}
				}
			} else {
				idx := ra.(object.Number)
				limit := ra1.(object.Number)
				step := ra2.(object.Number)
				idx += step
				if 0 < step {
					if idx <= limit {
						ci.pc += inst.SBx()
						ctx.setR(a, idx)
						ctx.setR(a+3, idx)

						break
					}
				} else {
					if idx >= limit {
						ci.pc += inst.SBx()
						ctx.setR(a, idx)
						ctx.setR(a+3, idx)

						break
					}
				}
			}
		case opcode.FORPREP:
			a := inst.A()
			ra := ctx.getR(a)
			ra1 := ctx.getR(a + 1)
			ra2 := ctx.getR(a + 2)

			if init, ok := ra.(object.Integer); ok {
				if step, ok := ra2.(object.Integer); ok {
					ilimit, ok := object.ToInteger(ra1)
					if !ok {
						nlimit, ok := object.ToNumber(ra1)
						if !ok {
							th.error(errors.ForLoopError("limit"))

							return nil
						}

						switch {
						case nlimit < object.Number(object.MinInteger):
							ilimit = object.MinInteger
							if step >= 0 {
								init = 0
							}
						case nlimit > object.Number(object.MaxInteger):
							ilimit = object.MaxInteger
							if step < 0 {
								init = 0
							}
						default:
							if step < 0 {
								if nlimit < 0 {
									ilimit = object.Integer(nlimit)
								} else {
									ilimit = object.Integer(nlimit + 1)
								}
							} else {
								if nlimit < 0 {
									ilimit = object.Integer(nlimit - 1)
								} else {
									ilimit = object.Integer(nlimit)
								}
							}
						}
					}

					ctx.setR(a, init-step)
					ctx.setR(a+1, ilimit)

					ci.pc += inst.SBx()

					break
				}
			}

			init, ok := object.ToNumber(ra)
			if !ok {
				th.error(errors.ForLoopError("initial"))

				return nil
			}

			limit, ok := object.ToNumber(ra1)
			if !ok {
				th.error(errors.ForLoopError("limit"))

				return nil
			}

			step, ok := object.ToNumber(ra2)
			if !ok {
				th.error(errors.ForLoopError("step"))

				return nil
			}

			ctx.setR(a, init-step)
			ctx.setR(a+1, limit)
			ctx.setR(a+2, step)

			ci.pc += inst.SBx()
		case opcode.TFORCALL:
			a := inst.A()
			nrets := inst.C()

			if err := th.tforcall(a, nrets); err != nil {
				th.error(err)

				return nil
			}

			tloop := ci.Code[ci.pc]

			if tloop.OpCode() != opcode.TFORLOOP {
				th.error(errors.InvalidByteCodeError())

				return nil
			}
		case opcode.TFORLOOP:
			a := inst.A()
			raplus := ctx.getR(a + 1)

			if raplus != nil {
				ctx.setR(a, raplus)

				ci.pc += inst.SBx()

				break
			}
		case opcode.SETLIST:
			a := inst.A()
			length := inst.B()
			if length == 0 {
				length = ci.top - ci.base - a - 1
			}

			c := inst.C()
			if c == 0 {
				extra := ci.Code[ci.pc]
				if extra.OpCode() != opcode.EXTRAARG {
					th.error(errors.InvalidByteCodeError())

					return nil
				}

				ci.pc++

				c = extra.Ax()
			}

			base := (c - 1) * version.LUA_FPF

			t := ctx.getR(a).(object.Table)

			t.SetList(base, ctx.stack[ci.base+a+1:ci.base+a+1+length])
		case opcode.CLOSURE:
			bx := inst.Bx()

			if len(ci.Protos) <= bx {
				th.error(errors.InvalidByteCodeError())

				return nil
			}

			cl := th.makeClosure(bx)

			ctx.setRA(inst, cl)
		case opcode.VARARG:
			a := inst.A()
			nrets := inst.B() - 1

			varargs := ci.varargs
			if nrets != -1 && nrets < len(varargs) {
				varargs = varargs[:nrets]
			}

			top := ci.base + a + len(varargs)

			if !ctx.growStack(top) {
				th.error(errors.StackOverflowError())
			}

			copy(ctx.stack[ci.base+a:], varargs)

			for r := ci.base + a + nrets; r >= top; r-- {
				ctx.stack[r] = nil
			}

			ctx.ci.top = top
		case opcode.EXTRAARG:
			th.error(errors.InvalidByteCodeError())

			return nil
		default:
			th.error(errors.InvalidByteCodeError())

			return nil
		}
	}
}

func (th *thread) dojmp(inst opcode.Instruction) {
	a := inst.A()
	sbx := inst.SBx()
	if a > 0 {
		th.closeUpvals(th.ci.base + a - 1)
	}
	th.ci.pc += sbx
}

func (th *thread) concat(a, b, c int) (err *object.RuntimeError) {
	ctx := th.context
	ci := ctx.ci

	rc := ctx.stack[ci.base+c]
	for r := c - 1; r >= b; r-- {
		rb := ctx.stack[ci.base+r]

		rc, err = arith.CallConcat(th, rb, rc)
		if err != nil {
			return err
		}
	}

	ctx.setR(a, rc)

	return nil
}

func isFunction(val object.Value) bool {
	return object.ToType(val) == object.TFUNCTION
}

func mustFunction(val object.Value) {
	if !isFunction(val) {
		panic(fmt.Sprintf("%v is not a function", val))
	}
}

func mustFunctionOrNil(val object.Value) {
	t := object.ToType(val)
	if t != object.TNIL && t != object.TFUNCTION {
		panic(fmt.Sprintf("%v is not a function", val))
	}
}

func (ctx *context) getR(r int) object.Value {
	return ctx.stack[ctx.ci.base+r]
}

func (ctx *context) setR(r int, val object.Value) {
	ctx.stack[ctx.ci.base+r] = val
}

func (ctx *context) getK(k int) object.Value {
	return ctx.ci.Constants[k]
}

func (ctx *context) getRK(rk int) object.Value {
	if rk&opcode.BitRK != 0 {
		return ctx.getK(rk & ^opcode.BitRK)
	}

	return ctx.getR(rk)
}

func (ctx *context) getU(r int) object.Value {
	return ctx.ci.GetUpvalue(r)
}

func (ctx *context) setU(r int, val object.Value) {
	ctx.ci.SetUpvalue(r, val)
}

func (ctx *context) getRA(inst opcode.Instruction) object.Value {
	return ctx.getR(inst.A())
}

func (ctx *context) getRB(inst opcode.Instruction) object.Value {
	return ctx.getR(inst.B())
}

func (ctx *context) getRC(inst opcode.Instruction) object.Value {
	return ctx.getR(inst.C())
}

func (ctx *context) setRA(inst opcode.Instruction, val object.Value) {
	ctx.setR(inst.A(), val)
}

func (ctx *context) setUB(inst opcode.Instruction, val object.Value) {
	ctx.setU(inst.B(), val)
}

func (ctx *context) getKBx(inst opcode.Instruction) object.Value {
	return ctx.getK(inst.Bx())
}

func (ctx *context) getKAx(inst opcode.Instruction) object.Value {
	return ctx.getK(inst.Ax())
}

func (ctx *context) getRKB(inst opcode.Instruction) object.Value {
	return ctx.getRK(inst.B())
}

func (ctx *context) getRKC(inst opcode.Instruction) object.Value {
	return ctx.getRK(inst.C())
}

func (ctx *context) getUA(inst opcode.Instruction) object.Value {
	return ctx.getU(inst.A())
}

func (ctx *context) getUB(inst opcode.Instruction) object.Value {
	return ctx.getU(inst.B())
}

package runtime

import (
	"github.com/hirochachacha/plua/internal/util"
	"github.com/hirochachacha/plua/internal/version"
	"github.com/hirochachacha/plua/object"
	"github.com/hirochachacha/plua/opcode"
)

func (th *thread) getInfo(level int, what string) *object.DebugInfo {
	if level < 0 {
		return nil
	}

	ctx := th.context

	i := len(ctx.ciStack) - 1 - level
	for i < 0 {
		ctx = ctx.prev
		if ctx == nil {
			return nil
		}
		i += len(ctx.ciStack)
	}

	ci := &ctx.ciStack[i]

	d := &object.DebugInfo{Func: ctx.fn(ci), CallInfo: ci}

	cl := ci.closure

	for _, r := range what {
		switch r {
		case 'S':
			setFuncInfo(d, cl)
		case 'l':
			d.CurrentLine = getCurrentLine(ci)
		case 'u':
			setUpInfo(d, cl)
		case 't':
			d.IsTailCall = ci.isTailCall
		case 'n':
			if ctx.hookState == isHook && i == 0 {
				d.Name = "?"
				d.NameWhat = "hook"
			} else {
				if !ci.isTailCall {
					pctx := ctx

					j := i - 1
					if j < 0 {
						pctx = pctx.prev
						if pctx == nil {
							break
						}
						j += len(pctx.ciStack)
					}

					prev := &pctx.ciStack[j]
					if prev != nil && !prev.isGoFunction() {
						setFuncName(d, prev)
					}
				}
			}
		case 'L':
			if cl != nil {
				lines := th.NewTableSize(0, len(cl.LineInfo))
				for _, line := range cl.LineInfo {
					lines.Set(object.Integer(line), object.True)
				}
				d.Lines = lines
			}
		}
	}

	return d
}

func (th *thread) getInfoByFunc(fn object.Value, what string) *object.DebugInfo {
	var cl *closure

	switch fn := fn.(type) {
	case object.Closure:
		cl = fn.(*closure)
	case object.GoFunction:
	default:
		panic("should be function")
	}

	d := &object.DebugInfo{Func: fn}

	for _, r := range what {
		switch r {
		case 'S':
			setFuncInfo(d, cl)
		case 'l':
			d.CurrentLine = -1
		case 'u':
			setUpInfo(d, cl)
		case 'L':
			if cl != nil {
				lines := th.NewTableSize(0, len(cl.LineInfo))
				for _, line := range cl.LineInfo {
					lines.Set(object.Integer(line), object.True)
				}
				d.Lines = lines
			}
		}
	}

	return d
}

func (th *thread) setLocal(level, n int, val object.Value) (name string) {
	if level < 0 {
		return
	}

	if n == 0 {
		return
	}

	ctx := th.context

	i := len(ctx.ciStack) - 1 - level
	for i < 0 {
		ctx = ctx.prev
		if ctx == nil {
			return
		}
		i += len(ctx.ciStack)
	}

	ci := &ctx.ciStack[i]

	if !ci.isGoFunction() {
		if n < 0 {
			if -n <= len(ci.varargs) {
				name = "(*vararg)"
				ci.varargs[-n-1] = val
			}

			return
		}

		name = getLocalName(ci.Prototype(), ci.pc, n)

		if i+1 < len(ctx.ciStack) {
			next := &ctx.ciStack[i+1]
			if ci.base-1+n <= next.base-1 {
				ctx.stack[ci.base-1+n] = val
				if name == "" {
					name = "(*temporary)"
				}
			}
		} else {
			if ci.base-1+n <= ci.top {
				ctx.stack[ci.base-1+n] = val
				if name == "" {
					name = "(*temporary)"
				}
			}
		}
	}

	return
}

func (th *thread) getLocal(level, n int) (name string, val object.Value) {
	if level < 0 {
		return
	}

	if n == 0 {
		return
	}

	ctx := th.context

	i := len(ctx.ciStack) - 1 - level
	for i < 0 {
		ctx = ctx.prev
		if ctx == nil {
			return
		}
		i += len(ctx.ciStack)
	}

	ci := &ctx.ciStack[i]

	if !ci.isGoFunction() {
		if n < 0 {
			if -n <= len(ci.varargs) {
				name, val = "(*vararg)", ci.varargs[-n-1]
			}

			return
		}

		name = getLocalName(ci.Prototype(), ci.pc, n)

		if i+1 < len(ctx.ciStack) {
			next := &ctx.ciStack[i+1]
			if ci.base-1+n <= next.base-1 {
				val = ctx.stack[ci.base-1+n]
				if name == "" {
					name = "(*temporary)"
				}
			}
		} else {
			if ci.base-1+n <= ci.top {
				val = ctx.stack[ci.base-1+n]
				if name == "" {
					name = "(*temporary)"
				}
			}
		}
	}

	return
}

func getLocalName(p *object.Proto, pc, n int) (name string) {
	for _, locvar := range p.LocVars {
		if pc < locvar.StartPC {
			break
		}

		if pc < locvar.EndPC {
			n--

			if n == 0 {
				return locvar.Name
			}
		}
	}
	return ""
}

func getUpvalName(p *object.Proto, n int) (name string) {
	name = p.Upvalues[n].Name
	if len(name) == 0 {
		name = "?"
	}

	return
}

func getRKName(p *object.Proto, pc, rk int) (name string) {
	if rk&opcode.BitRK != 0 {
		if s, ok := p.Constants[rk & ^opcode.BitRK].(object.String); ok {
			return string(s)
		}
	} else {
		name, nameWhat := getObjectName(p, pc, rk)
		if nameWhat == "constant" {
			return name
		}
	}

	return "?"
}

func setFuncInfo(d *object.DebugInfo, cl *closure) {
	if cl == nil {
		d.Source = "=[Go]"
		d.ShortSource = "[Go]"
		d.LineDefined = -1
		d.LastLineDefined = -1
		d.What = "Go"
	} else {
		if len(cl.Source) == 0 {
			d.Source = "=?"
			d.ShortSource = "?"
		} else {
			d.Source = cl.Source
			d.ShortSource = util.Shorten(cl.Source)
		}
		d.LineDefined = cl.LineDefined
		d.LastLineDefined = cl.LastLineDefined
		if d.LineDefined == 0 {
			d.What = "main"
		} else {
			d.What = "Lua"
		}
	}
}

func getCurrentLine(ci *callInfo) int {
	if ci == nil || ci.isGoFunction() {
		return -1
	}
	if len(ci.LineInfo) == 0 {
		return -1
	}
	if ci.pc == 0 { // see execute0, go stack overflow
		return ci.LineInfo[0]
	}
	return ci.LineInfo[ci.pc-1]
}

func getObjectName(p *object.Proto, pc, reg int) (name, nameWhat string) {
	name = getLocalName(p, pc, reg+1)
	if len(name) != 0 {
		nameWhat = "local"

		return
	}

	pc = getRelativePC(p, pc, reg)

	if pc != -1 { /* could find instruction? */
		inst := p.Code[pc]

		switch inst.OpCode() {
		case opcode.MOVE:
			b := inst.B()
			if b < inst.A() {
				return getObjectName(p, pc, b)
			}
		case opcode.GETTABUP:
			t := inst.B()
			key := inst.C()
			tn := getUpvalName(p, t)
			name = getRKName(p, pc, key)
			if tn == version.LUA_ENV {
				nameWhat = "global"
			} else {
				nameWhat = "field"
			}
		case opcode.GETTABLE:
			t := inst.B()
			key := inst.C()
			tn := getLocalName(p, pc, t+1)
			name = getRKName(p, pc, key)
			if tn == version.LUA_ENV {
				nameWhat = "global"
			} else {
				nameWhat = "field"
			}
		case opcode.GETUPVAL:
			name = getUpvalName(p, inst.B())
			nameWhat = "upvalue"
		case opcode.LOADK:
			bx := inst.Bx()
			if s, ok := p.Constants[bx].(object.String); ok {
				name = string(s)
			}
			nameWhat = "constant"
		case opcode.LOADKX:
			ax := p.Code[pc+1].Ax()
			if s, ok := p.Constants[ax].(object.String); ok {
				name = string(s)
			}
			nameWhat = "constant"
		case opcode.SELF:
			key := inst.C()
			name = getRKName(p, pc, key)
			nameWhat = "method"
		}
	}

	return
}

func getRelativePC(p *object.Proto, lastpc, n int) (relpc int) {
	var jmpdest int

	relpc = -1

	for pc := 0; pc < lastpc; pc++ {
		inst := p.Code[pc]

		a := inst.A()

		switch op := inst.OpCode(); op {
		case opcode.LOADNIL:
			b := inst.B()
			if a <= n && n <= a+b {
				if pc < jmpdest {
					relpc = -1
				} else {
					relpc = pc
				}
			}
		case opcode.TFORCALL:
			if n >= a+2 {
				if pc < jmpdest {
					relpc = -1
				} else {
					relpc = pc
				}
			}
		case opcode.CALL, opcode.TAILCALL:
			if n >= a {
				if pc < jmpdest {
					relpc = -1
				} else {
					relpc = pc
				}
			}
		case opcode.JMP:
			sbx := inst.SBx()
			dest := pc + 1 + sbx
			if pc < dest && dest <= lastpc {
				if dest > jmpdest {
					jmpdest = dest
				}
			}
		default:
			if op.TestAMode() && n == a {
				if pc < jmpdest {
					relpc = -1
				} else {
					relpc = pc
				}
			}
		}
	}

	return
}

func setUpInfo(d *object.DebugInfo, cl *closure) {
	if cl == nil {
		d.NUpvalues = 0
		d.IsVararg = true
		d.NParams = 0
	} else {
		d.NUpvalues = cl.NUpvalues()
		d.IsVararg = cl.IsVararg
		d.NParams = cl.NParams
	}
}

func setFuncName(d *object.DebugInfo, ci *callInfo) {
	var tag object.Value

	inst := ci.Code[ci.pc-1]

	switch inst.OpCode() {
	case opcode.CALL, opcode.TAILCALL:
		d.Name, d.NameWhat = getObjectName(ci.Prototype(), ci.pc-1, inst.A())

		return
	case opcode.TFORCALL:
		d.Name = "for iterator"
		d.NameWhat = "for iterator"

		return
	case opcode.SELF, opcode.GETTABUP, opcode.GETTABLE:
		tag = object.TM_INDEX
	case opcode.SETTABUP, opcode.SETTABLE:
		tag = object.TM_NEWINDEX
	case opcode.ADD:
		tag = object.TM_ADD
	case opcode.SUB:
		tag = object.TM_SUB
	case opcode.MUL:
		tag = object.TM_MUL
	case opcode.MOD:
		tag = object.TM_MOD
	case opcode.POW:
		tag = object.TM_POW
	case opcode.DIV:
		tag = object.TM_DIV
	case opcode.IDIV:
		tag = object.TM_IDIV
	case opcode.BAND:
		tag = object.TM_BAND
	case opcode.BOR:
		tag = object.TM_BOR
	case opcode.BXOR:
		tag = object.TM_BXOR
	case opcode.SHL:
		tag = object.TM_SHL
	case opcode.SHR:
		tag = object.TM_SHR
	case opcode.UNM:
		tag = object.TM_UNM
	case opcode.BNOT:
		tag = object.TM_BNOT
	case opcode.LEN:
		tag = object.TM_LEN
	case opcode.CONCAT:
		tag = object.TM_CONCAT
	case opcode.EQ:
		tag = object.TM_EQ
	case opcode.LT:
		tag = object.TM_LT
	case opcode.LE:
		tag = object.TM_LE
	default:
		return
	}

	d.Name = tag.String()
	d.NameWhat = "metamethod"
}

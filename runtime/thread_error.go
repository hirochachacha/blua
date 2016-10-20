package runtime

import (
	"github.com/hirochachacha/plua/object"
	"github.com/hirochachacha/plua/position"
)

func (th *thread) error(err *object.RuntimeError) {
	if th.status != object.THREAD_ERROR {
		if err.Level > 0 {
			l := err.Level - 1
			for {
				d := th.getInfo(l, "Sl")
				if d == nil {
					break
				}
				err.Traceback = append(err.Traceback, position.Position{
					SourceName: "@" + d.ShortSource,
					Line:       d.CurrentLine,
					Column:     -1,
				})

				l++
			}
		}

		th.status = object.THREAD_ERROR
		th.err = err
	}
}

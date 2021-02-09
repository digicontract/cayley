package gizmo

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

type Console struct {
	sess *Session
	vm   *goja.Runtime
}

func NewConsole(sess *Session) require.ModuleLoader {
	return func(runtime *goja.Runtime, module *goja.Object) {
		c := &Console{
			sess: sess,
			vm:   runtime,
		}

		o := module.Get("exports").(*goja.Object)
		logFn := c.logger("info", c.sess.log.Infof)
		if err := o.Set("log", logFn); err != nil {
			panic(err)
		}

		debugFn := c.logger("debug", c.sess.log.Debugf)
		if err := o.Set("debug", debugFn); err != nil {
			panic(err)
		}

		infoFn := c.logger("info", c.sess.log.Infof)
		if err := o.Set("info", infoFn); err != nil {
			panic(err)
		}

		warnFn := c.logger("warn", c.sess.log.Warnf)
		if err := o.Set("warn", warnFn); err != nil {
			panic(err)
		}

		errorFn := c.logger("error", c.sess.log.Errorf)
		if err := o.Set("error", errorFn); err != nil {
			panic(err)
		}
	}
}

func (c *Console) logger(level string, logf func(string, ...interface{})) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		args := exportArgs(call.Arguments)
		if len(args) == 0 {
			panic(c.vm.ToValue(errArgCount{Got: len(args)}))
		}

		format := false
		if arg, ok := args[0].(string); ok {
			format = strings.IndexAny(arg, "%") != -1
		}

		err := map[string]interface{}{"level": level}
		if format {
			logf(args[0].(string), args[1:]...)
			err["data"] = fmt.Sprintf(args[0].(string), args[1:]...)
			if level == "error" || !c.sess.error(err) {
				panic(c.vm.ToValue(err))
			}
		} else {
			logf("%+v", args)
			err["data"] = args
			if level == "error" || !c.sess.error(err) {
				panic(c.vm.ToValue(err))
			}
		}

		return goja.Undefined()
	}
}

package lua

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vxcontrol/golua/lua"
	"github.com/vxcontrol/luar"
	"go.opentelemetry.io/otel/attribute"

	obs "soldr/internal/observability"
)

// State is context of lua module
type State struct {
	tmpdir string
	closed bool
	L      *lua.State
	logger *logrus.Entry
	ctx    context.Context // ctx will be rotated after await call
}

// is not routine safe and here should use synchronization
// on mutex when it will use in another space
func stateDestructor(s *State) {
	if s.L != nil {
		s.L.Close()
		os.RemoveAll(s.tmpdir)
		s.L = nil
		s.logger.WithContext(s.ctx).Info("the state was destroyed")
		obs.Observer.SpanFromContext(s.ctx).End()
	}
}

// NewState is function which constructed State object
func NewState(files map[string][]byte) (*State, error) {
	if files["main.lua"] == nil {
		return nil, fmt.Errorf("main module not found")
	}

	nfiles := make(map[string][]byte)
	for name, data := range files {
		nfiles[name] = data
	}

	var err error
	var tmpdir string
	if tmpdir, err = ioutil.TempDir("", "vxlua-"); err != nil {
		return nil, err
	}

	pathToPID := filepath.Join(tmpdir, "lock.pid")
	pid := strconv.Itoa(os.Getpid())
	if err = ioutil.WriteFile(pathToPID, []byte(pid), 0640); err != nil {
		return nil, err
	}

	ctx, _ := obs.Observer.NewSpan(context.Background(), obs.SpanKindInternal, "lua_state")
	s := &State{
		ctx:    ctx,
		tmpdir: tmpdir,
		closed: true,
		L:      luar.Init(),
		logger: logrus.WithFields(logrus.Fields{
			"component": "lua_state",
			"tmpdir":    filepath.Base(tmpdir),
		}),
	}
	runtime.SetFinalizer(s, stateDestructor)

	s.getRegisterCalls()(s.L)
	s.getRegisterLoader()(s.L)
	s.getRegisterPanicRecover()(s.L)

	if err = s.loadClibs(nfiles); err != nil {
		return nil, err
	}

	if err = s.loadData(nfiles); err != nil {
		return nil, err
	}

	lfiles := make(map[string]string)
	for name, data := range nfiles {
		lfiles[name] = string(data)
	}
	s.getRegisterFFILoader()(s.L, s.tmpdir)
	luar.GoToLua(s.L, tmpdir)
	s.L.SetGlobal("__tmpdir")
	luar.GoToLua(s.L, lfiles)
	s.L.SetGlobal("__files")

	s.logger.WithContext(s.ctx).Info("the state was created")
	return s, nil
}

func (s *State) loadClibs(files map[string][]byte) error {
	const (
		clibsPrefix  = "clibs/"
		strictPrefix = clibsPrefix + runtime.GOOS + "/" + runtime.GOARCH + "/"
	)

	for name, data := range files {
		if strings.HasPrefix(name, strictPrefix) {
			fname := filepath.Join(s.tmpdir, strings.TrimPrefix(name, strictPrefix))
			if err := writeFile(s.ctx, fname, data, s.logger); err != nil {
				return err
			}
		}
		if strings.HasPrefix(name, clibsPrefix) {
			delete(files, name)
		}
	}

	loadstr := fmt.Sprintf(`package.cpath = package.cpath .. ";%s"`, normPath(s.tmpdir))
	if err := s.L.DoString(loadstr); err != nil {
		s.logger.WithContext(s.ctx).WithError(err).Error("failed to add a new cpath to the lua state")
		return err
	}

	s.logger.WithContext(s.ctx).Debug("the state loaded clibs")
	return nil
}

func (s *State) loadData(files map[string][]byte) error {
	const prefix = "data/"
	for name, data := range files {
		if strings.HasPrefix(name, prefix) {
			fname := filepath.Join(s.tmpdir, name)
			if err := writeFile(s.ctx, fname, data, s.logger); err != nil {
				return err
			}
			delete(files, name)
		}
	}
	s.logger.WithContext(s.ctx).Debug("the state loaded data")
	return nil
}

func writeFile(ctx context.Context, path string, contents []byte, logger *logrus.Entry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		logger.WithContext(ctx).WithError(err).WithField("dir", dir).
			Error("failed to create a directory")
		return fmt.Errorf("failed to create a directory: %w", err)
	}
	if err := ioutil.WriteFile(path, contents, 0640); err != nil {
		logger.WithContext(ctx).WithError(err).WithField("name", path).
			Error("failed to write a file")
		return fmt.Errorf("failed to write a file: %w", err)
	}
	return nil
}

func (s *State) getModuleLoader() func(*lua.State) int {
	logger := s.logger
	return func(L *lua.State) int {
		moduleName := L.CheckString(1)

		logger := logger.WithFields(logrus.Fields{
			"component": "lua_module_loader",
			"require":   moduleName,
		})
		var files map[string]string
		L.GetGlobal("__files")
		err := luar.LuaToGo(L, -1, &files)
		L.Pop(1)
		if err != nil {
			logger.WithError(err).Error("failed to put the module files into the lua state")
		}

		var moduleNames []string
		pathToModule := strings.Replace(moduleName, ".", "/", -1)
		moduleNames = append(moduleNames, pathToModule+"/init.lua")
		moduleNames = append(moduleNames, pathToModule+".lua")
		for _, filePath := range moduleNames {
			moduleData, ok := files[filePath]
			if ok && L.LoadBuffer([]byte(moduleData), len(moduleData), moduleName) != 0 {
				err = fmt.Errorf(L.ToString(-1))
				logger.WithError(err).Error("failed to put the module data into the lua state")
				L.Pop(1)
				break
			}
		}

		return 1
	}
}

func (s *State) getRegisterCalls() func(*lua.State) {
	return func(L *lua.State) {
		L.GetGlobal("unsafe_pcall")
		L.SetGlobal("pcall")
		L.GetGlobal("unsafe_xpcall")
		L.SetGlobal("xpcall")
	}
}

func (s *State) getRegisterLoader() func(*lua.State) {
	moduleLoader := s.getModuleLoader()
	return func(L *lua.State) {
		top := L.GetTop()
		L.GetGlobal(lua.LUA_LOADLIBNAME)
		L.GetField(-1, "loaders")
		L.PushGoFunction(moduleLoader)
		L.RawSeti(-2, int(L.ObjLen(-2)+1))
		L.SetTop(top)
	}
}

func normPath(args ...string) string {
	var path string

	if runtime.GOOS == "windows" {
		args = append(args, "?.dll")
		path = filepath.Join(args...)
		path = strings.Replace(path, "\\", "\\\\", -1)
	} else if runtime.GOOS == "darwin" {
		path = filepath.Join(append(args, "lib?.dylib")...)
		path = path + ";" + filepath.Join(append(args, "?.dylib")...)
		path = path + ";" + filepath.Join(append(args, "lib?.so")...)
		path = path + ";" + filepath.Join(append(args, "?.so")...)
	} else {
		path = filepath.Join(append(args, "lib?.so")...)
		path = path + ";" + filepath.Join(append(args, "?.so")...)
	}

	return path
}

func (s *State) getRegisterFFILoader() func(*lua.State, string) error {
	return func(L *lua.State, tmpdir string) error {
		var err error
		regFFILoader := fmt.Sprintf(`
	local ffi = require'ffi'
	local ffi_load = ffi.load
	--overload ffi.load for received system libs
	function ffi.load(name, ...)
		local libpath = package.searchpath(name, "%s")
		if libpath ~= nil then
			return ffi_load(libpath)
		else
			return ffi_load(name)
		end
	end`, normPath(tmpdir, "sys")+";"+normPath(tmpdir))

		if err = L.DoString(regFFILoader); err != nil {
			return err
		}

		return nil
	}
}

func (s *State) getRegisterPanicRecover() func(*lua.State) {
	logger := s.logger
	return func(L *lua.State) {
		var currentPanicf lua.LuaGoFunction
		newPanic := func(L1 *lua.State) int {
			le := (&lua.LuaError{}).New(L1, 0, L1.ToString(-1))
			logger.WithError(le).WithField("component", "lua_panic_recovery").
				Error("lua state panic recovery")
			if currentPanicf != nil {
				return currentPanicf(L1)
			}
			return 1
		}
		currentPanicf = L.AtPanic(newPanic)
	}
}

// Context is getter for internal context from lua state
func (s *State) Context() context.Context {
	return s.ctx
}

// Exec is blocked function for data execution
func (s *State) Exec() (string, error) {
	s.closed = false
	defer func() {
		s.closed = true
	}()

	s.logger.WithContext(s.ctx).Info("the state was started")
	defer func(s *State) {
		s.logger.WithContext(s.ctx).Info("the state was stopped")
	}(s)

	err := s.L.DoString(`return require('main')`)
	if err != nil {
		s.logger.WithContext(s.ctx).WithError(err).
			Error("the executing state caught an error")
		return "", err
	}

	return s.L.CheckString(1), nil
}

// IsClose is nonblocked function which check a state of module
func (s *State) IsClose() bool {
	return s.closed
}

func (s *State) RegisterLogger(level logrus.Level, fields logrus.Fields) error {
	makeLogFn := func(s *State, lvl logrus.Level, flds logrus.Fields) func(...interface{}) {
		return func(values ...interface{}) {
			logger := s.logger.WithContext(s.ctx).WithFields(flds)
			switch lvl {
			case logrus.ErrorLevel:
				logger.Error(values)
			case logrus.WarnLevel:
				logger.Warn(values)
			case logrus.InfoLevel:
				logger.Info(values)
			case logrus.DebugLevel:
				logger.Debug(values)
			default:
				logger.Info(values)
			}
		}
	}

	s.L.CreateTable(0, 9)
	s.L.PushInteger(int64(level))
	s.L.SetField(-2, "level")
	s.L.PushInteger(int64(logrus.ErrorLevel))
	s.L.SetField(-2, "level_error")
	s.L.PushInteger(int64(logrus.WarnLevel))
	s.L.SetField(-2, "level_warn")
	s.L.PushInteger(int64(logrus.InfoLevel))
	s.L.SetField(-2, "level_info")
	s.L.PushInteger(int64(logrus.DebugLevel))
	s.L.SetField(-2, "level_debug")
	luar.GoToLua(s.L, makeLogFn(s, logrus.ErrorLevel, fields))
	s.L.SetField(-2, "_error")
	luar.GoToLua(s.L, makeLogFn(s, logrus.WarnLevel, fields))
	s.L.SetField(-2, "_warn")
	luar.GoToLua(s.L, makeLogFn(s, logrus.InfoLevel, fields))
	s.L.SetField(-2, "_info")
	luar.GoToLua(s.L, makeLogFn(s, logrus.DebugLevel, fields))
	s.L.SetField(-2, "_debug")
	s.L.SetGlobal("__log")

	s.L.DoString(`
	function __log.error(...)
		if __log.level >= __log.level_error then
			__log._error(...)
		end
	end
	function __log.warn(...)
		if __log.level >= __log.level_warn then
			__log._warn(...)
		end
	end
	function __log.info(...)
		if __log.level >= __log.level_info then
			__log._info(...)
		end
	end
	function __log.debug(...)
		if __log.level >= __log.level_debug then
			__log._debug(...)
		end
	end
	`)

	s.L.DoString(`
	function __log.errorf(fmt, ...)
		if __log.level >= __log.level_error then
			__log._error(string.format(fmt, ...))
		end
	end
	function __log.warnf(fmt, ...)
		if __log.level >= __log.level_warn then
			__log._warn(string.format(fmt, ...))
		end
	end
	function __log.infof(fmt, ...)
		if __log.level >= __log.level_info then
			__log._info(string.format(fmt, ...))
		end
	end
	function __log.debugf(fmt, ...)
		if __log.level >= __log.level_debug then
			__log._debug(string.format(fmt, ...))
		end
	end
	`)

	return nil
}

func (s *State) RegisterMeter(fields logrus.Fields) error {
	registry, err := obs.NewMetricRegistry()
	if err != nil {
		return err
	}

	logger := s.logger.WithFields(fields)
	attrs := []attribute.KeyValue{
		attribute.String("component", "lua_state"),
	}
	for name, value := range fields {
		switch val := value.(type) {
		case string:
			attrs = append(attrs, attribute.String(name, val))
		case fmt.Stringer:
			attrs = append(attrs, attribute.String(name, val.String()))
		default:
			attrs = append(attrs, attribute.String(name, fmt.Sprintf("%v", val)))
		}
	}
	makeIntMetricFn := func(s *State, mtype string) func(string, int64) {
		return func(name string, value int64) {
			switch mtype {
			case "counter":
				counter, err := registry.NewInt64Counter(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add int counter")
					return
				}
				counter.Add(s.ctx, value, attrs...)
			case "updown_counter":
				counter, err := registry.NewInt64UpDownCounter(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add int updown counter")
					return
				}
				counter.Add(s.ctx, value, attrs...)
			case "gauge_counter":
				counter, err := registry.NewInt64GaugeCounter(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add int gauge counter")
					return
				}
				counter.Record(s.ctx, value, attrs...)
			case "histogram":
				histogram, err := registry.NewInt64Histogram(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add int histogram value")
					return
				}
				histogram.Record(s.ctx, value, attrs...)
			}
		}
	}
	makeFloatMetricFn := func(s *State, mtype string) func(string, float64) {
		return func(name string, value float64) {
			switch mtype {
			case "counter":
				counter, err := registry.NewFloat64Counter(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add float counter")
					return
				}
				counter.Add(s.ctx, value, attrs...)
			case "updown_counter":
				counter, err := registry.NewFloat64UpDownCounter(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add float updown counter")
					return
				}
				counter.Add(s.ctx, value, attrs...)
			case "gauge_counter":
				counter, err := registry.NewFloat64GaugeCounter(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add float gauge counter")
					return
				}
				counter.Record(s.ctx, value, attrs...)
			case "histogram":
				histogram, err := registry.NewFloat64Histogram(name)
				if err != nil {
					logger.WithContext(s.ctx).Error("failed to add float histogram value")
					return
				}
				histogram.Record(s.ctx, value, attrs...)
			}
		}
	}

	s.L.CreateTable(0, 8)
	luar.GoToLua(s.L, makeIntMetricFn(s, "counter"))
	s.L.SetField(-2, "add_int_counter")
	luar.GoToLua(s.L, makeIntMetricFn(s, "gauge_counter"))
	s.L.SetField(-2, "add_int_gauge_counter")
	luar.GoToLua(s.L, makeIntMetricFn(s, "updown_counter"))
	s.L.SetField(-2, "add_int_updown_counter")
	luar.GoToLua(s.L, makeIntMetricFn(s, "histogram"))
	s.L.SetField(-2, "add_int_histogram")
	luar.GoToLua(s.L, makeFloatMetricFn(s, "counter"))
	s.L.SetField(-2, "add_float_counter")
	luar.GoToLua(s.L, makeFloatMetricFn(s, "gauge_counter"))
	s.L.SetField(-2, "add_float_gauge_counter")
	luar.GoToLua(s.L, makeFloatMetricFn(s, "updown_counter"))
	s.L.SetField(-2, "add_float_updown_counter")
	luar.GoToLua(s.L, makeFloatMetricFn(s, "histogram"))
	s.L.SetField(-2, "add_float_histogram")
	s.L.SetGlobal("__metric")

	return nil
}

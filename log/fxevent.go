package log

import (
	"github.com/rs/zerolog"
	"go.uber.org/dig"
	"go.uber.org/fx/fxevent"
)

// fxLogger is an event logger that logs events to Zerolog.
type fxLogger struct {
	*zerolog.Logger
}

// InitFxLogger returns the logger instance for Zerolog.
func InitFxLogger(logger *zerolog.Logger) fxevent.Logger {
	return fxLogger{Logger: logger}
}

type moduleName string

func (m moduleName) MarshalZerologObject(event *zerolog.Event) {
	if m != "" {
		event.Str("name", string(m))
	}
}

// LogEvent logs the given event to the provided Zerolog.
func (l fxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.Trace().
			Str("callee", e.FunctionName).
			Str("caller", e.CallerName).
			Msg("OnStart hook executing")
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.Error().
				Str("callee", e.FunctionName).
				Str("caller", e.CallerName).
				Err(dig.RootCause(e.Err)).
				Msg("OnStart hook failed")
		} else {
			l.Trace().
				Str("callee", e.FunctionName).
				Str("caller", e.CallerName).
				Dur("runtime", e.Runtime).
				Msg("OnStart hook executed")
		}
	case *fxevent.OnStopExecuting:
		l.Trace().
			Str("callee", e.FunctionName).
			Str("caller", e.CallerName).
			Msg("OnStop hook executing")
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.Error().
				Str("callee", e.FunctionName).
				Str("caller", e.CallerName).
				Err(dig.RootCause(e.Err)).
				Msg("OnStop hook failed")
		} else {
			l.Trace().
				Str("callee", e.FunctionName).
				Str("caller", e.CallerName).
				Dur("runtime", e.Runtime).
				Msg("OnStop hook executed")
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.Error().
				Str("type", e.TypeName).
				Strs("moduleTrace", e.ModuleTrace).
				Strs("stackTrace", e.StackTrace).
				EmbedObject(moduleName(e.ModuleName)).
				Err(dig.RootCause(e.Err)).
				Msg("Error encountered while applying options")
		} else {
			l.Info().
				Str("type", e.TypeName).
				Strs("moduleTrace", e.ModuleTrace).
				Strs("stackTrace", e.StackTrace).
				EmbedObject(moduleName(e.ModuleName)).
				Msg("Supplied")
		}
	case *fxevent.Provided:
		if e.Err != nil {
			l.Error().
				Str("constructor", e.ConstructorName).
				Strs("types", e.OutputTypeNames).
				Strs("moduleTrace", e.ModuleTrace).
				Strs("stackTrace", e.StackTrace).
				EmbedObject(moduleName(e.ModuleName)).
				Err(dig.RootCause(e.Err)).
				Msg("Error encountered while applying options")
		} else {
			l.Info().
				Str("constructor", e.ConstructorName).
				Strs("types", e.OutputTypeNames).
				EmbedObject(moduleName(e.ModuleName)).
				Bool("private", e.Private).
				Msg("Provided")
		}
	case *fxevent.Replaced:
		if e.Err != nil {
			l.Error().
				Strs("types", e.OutputTypeNames).
				Strs("moduleTrace", e.ModuleTrace).
				Strs("stackTrace", e.StackTrace).
				EmbedObject(moduleName(e.ModuleName)).
				Err(dig.RootCause(e.Err)).
				Msg("Error encountered while replacing")
		} else {
			l.Info().
				Strs("types", e.OutputTypeNames).
				EmbedObject(moduleName(e.ModuleName)).
				Msg("Replaced")
		}
	case *fxevent.Decorated:
		if e.Err != nil {
			l.Error().
				Str("decorator", e.DecoratorName).
				Strs("types", e.OutputTypeNames).
				Strs("moduleTrace", e.ModuleTrace).
				Strs("stackTrace", e.StackTrace).
				EmbedObject(moduleName(e.ModuleName)).
				Err(dig.RootCause(e.Err)).
				Msg("Error encountered while applying options")
		} else {
			l.Info().
				Str("decorator", e.DecoratorName).
				Strs("types", e.OutputTypeNames).
				EmbedObject(moduleName(e.ModuleName)).
				Msg("Decorated")
		}
	case *fxevent.BeforeRun:
		l.Trace().
			Str("name", e.Name).
			Str("kind", e.Kind).
			EmbedObject(moduleName(e.ModuleName)).
			Msg("Before run")
	case *fxevent.Run:
		if e.Err != nil {
			l.Error().
				Str("name", e.Name).
				Str("kind", e.Kind).
				EmbedObject(moduleName(e.ModuleName)).
				Err(dig.RootCause(e.Err)).
				Dur("runtime", e.Runtime).
				Msg("Run failed")
		} else {
			l.Trace().
				Str("name", e.Name).
				Str("kind", e.Kind).
				EmbedObject(moduleName(e.ModuleName)).
				Dur("runtime", e.Runtime).
				Msg("After run")
		}
	case *fxevent.Invoking:
		// Do not log stack as it will make logs hard to read.
		l.Info().
			Str("function", e.FunctionName).
			EmbedObject(moduleName(e.ModuleName)).
			Msg("Invoking")
	case *fxevent.Invoked:
		if e.Err != nil {
			l.Error().
				Str("function", e.FunctionName).
				EmbedObject(moduleName(e.ModuleName)).
				Err(dig.RootCause(e.Err)).
				Str("stack", e.Trace).
				Msg("Invoke failed")
		}
	case *fxevent.Stopping:
		l.Info().
			Stringer("signal", e.Signal).
			Msg("Received signal")
	case *fxevent.Stopped:
		if e.Err != nil {
			l.Error().Err(dig.RootCause(e.Err)).Msg("Stop failed")
		}
	case *fxevent.RollingBack:
		l.Error().Err(e.StartErr).Msg("Start failed, rolling back")
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.Error().Err(dig.RootCause(e.Err)).Msg("Rollback failed")
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.Error().Err(dig.RootCause(e.Err)).Msg("Start failed")
		} else {
			l.Info().Msg("Started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.Error().Err(dig.RootCause(e.Err)).Msg("Logger initialization failed")
		} else {
			l.Info().Str("function", e.ConstructorName).Msg("Initialized logger")
		}
	}
}

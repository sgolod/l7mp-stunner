package logger

import (
	"strings"
	"sync"

	"github.com/pion/logging"
)

// logLevels maps the string representation of a log level to the pion LogLevel constant.
var logLevels = map[string]logging.LogLevel{
	"DISABLE": logging.LogLevelDisabled,
	"ERROR":   logging.LogLevelError,
	"WARN":    logging.LogLevelWarn,
	"INFO":    logging.LogLevelInfo,
	"DEBUG":   logging.LogLevelDebug,
	"TRACE":   logging.LogLevelTrace,
}

// levelSetter is the union of a leveled logger and the ability to change its level at runtime.
// Every logger stored in a factory's registry must satisfy this interface; the assertion
// l.(levelSetter) in NewLogger is the single enforcement point.
type levelSetter interface {
	logging.LeveledLogger
	SetLevel(logging.LogLevel)
}

// scopedLevels is the shared level-management base embedded by all logger factories.
// It owns the mutex, the level maps, and the logger registry, so SetLevel and GetLevel
// have identical semantics across backends.
type scopedLevels struct {
	DefaultLogLevel logging.LogLevel
	ScopeLevels     map[string]logging.LogLevel
	loggers         map[string]levelSetter
	lock            sync.RWMutex
}

func newScopedLevels() scopedLevels {
	return scopedLevels{
		DefaultLogLevel: logging.LogLevelError,
		ScopeLevels:     make(map[string]logging.LogLevel),
		loggers:         make(map[string]levelSetter),
	}
}

// parseLevelSpec applies a comma-separated "scope:level" spec to the level tables.
// Caller must hold the write lock.
func (s *scopedLevels) parseLevelSpec(levelSpec string) {
	for spec := range strings.SplitSeq(levelSpec, ",") {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			continue
		}
		l, ok := logLevels[strings.ToUpper(parts[1])]
		if !ok {
			continue
		}
		if strings.ToLower(parts[0]) == "all" {
			for c := range s.ScopeLevels {
				s.ScopeLevels[c] = l
			}
			s.DefaultLogLevel = l
			continue
		}
		s.ScopeLevels[parts[0]] = l
	}
}

// levelFor returns the effective log level for scope. Caller must hold the lock.
func (s *scopedLevels) levelFor(scope string) logging.LogLevel {
	if l, ok := s.ScopeLevels[scope]; ok {
		return l
	}
	return s.DefaultLogLevel
}

// SetLevel applies a scoped level spec and propagates the new levels to all registered loggers.
func (s *scopedLevels) SetLevel(levelSpec string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.parseLevelSpec(levelSpec)
	for scope, l := range s.loggers {
		l.SetLevel(s.levelFor(scope))
	}
}

// GetLevel returns the log level string for the given scope.
func (s *scopedLevels) GetLevel(scope string) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.levelFor(scope).String()
}

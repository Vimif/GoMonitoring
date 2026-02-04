package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Level reprÃ©sente le niveau de log
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String retourne la reprÃ©sentation textuelle du niveau
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger est un logger structurÃ© simple
type Logger struct {
	level  Level
	prefix string
}

// New crÃ©e un nouveau logger
func New(prefix string, level Level) *Logger {
	return &Logger{
		level:  level,
		prefix: prefix,
	}
}

// Default retourne un logger par dÃ©faut
func Default() *Logger {
	return &Logger{
		level:  InfoLevel,
		prefix: "",
	}
}

// WithPrefix retourne un nouveau logger avec un prÃ©fixe
func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		level:  l.level,
		prefix: l.prefix + prefix,
	}
}

// SetLevel change le niveau de log
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// Debug log un message de debug
func (l *Logger) Debug(msg string, fields ...Field) {
	if l.level <= DebugLevel {
		l.log(DebugLevel, msg, fields...)
	}
}

// Info log un message d'information
func (l *Logger) Info(msg string, fields ...Field) {
	if l.level <= InfoLevel {
		l.log(InfoLevel, msg, fields...)
	}
}

// Warn log un avertissement
func (l *Logger) Warn(msg string, fields ...Field) {
	if l.level <= WarnLevel {
		l.log(WarnLevel, msg, fields...)
	}
}

// Error log une erreur
func (l *Logger) Error(msg string, fields ...Field) {
	if l.level <= ErrorLevel {
		l.log(ErrorLevel, msg, fields...)
	}
}

// Fatal log une erreur fatale et termine le programme
func (l *Logger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, msg, fields...)
	os.Exit(1)
}

// log effectue le logging avec formatage
func (l *Logger) log(level Level, msg string, fields ...Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Construire le message
	logMsg := fmt.Sprintf("[%s] %s %s%s", timestamp, level.String(), l.prefix, msg)

	// Ajouter les champs
	if len(fields) > 0 {
		logMsg += " |"
		for _, field := range fields {
			logMsg += fmt.Sprintf(" %s=%v", field.Key, field.Value)
		}
	}

	log.Println(logMsg)
}

// Field reprÃ©sente un champ de log structurÃ©
type Field struct {
	Key   string
	Value interface{}
}

// String crÃ©e un champ string
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int crÃ©e un champ int
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 crÃ©e un champ int64
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 crÃ©e un champ float64
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool crÃ©e un champ bool
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error crÃ©e un champ erreur
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Duration crÃ©e un champ durÃ©e
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Any crÃ©e un champ de type quelconque
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger global par dÃ©faut
var defaultLogger = Default()

// Debug log un message de debug avec le logger global
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

// Info log un message d'information avec le logger global
func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

// Warn log un avertissement avec le logger global
func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

// Error log une erreur avec le logger global
func ErrorLog(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}

// Fatal log une erreur fatale avec le logger global
func Fatal(msg string, fields ...Field) {
	defaultLogger.Fatal(msg, fields...)
}

// SetGlobalLevel change le niveau du logger global
func SetGlobalLevel(level Level) {
	defaultLogger.SetLevel(level)
}

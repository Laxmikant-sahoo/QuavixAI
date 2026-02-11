package logger

import "log"

type Logger struct{}

func New(env string) *Logger {
	return &Logger{}
}

func (l *Logger) Info(v ...any)  { log.Println(v...) }
func (l *Logger) Error(v ...any) { log.Println(v...) }

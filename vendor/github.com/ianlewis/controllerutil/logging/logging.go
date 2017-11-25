// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logging provides standard log.Logger wrappers around the github.com/golang/glog package.
// These wrappers can be set with log message prefixes.
package logging

import (
	"log"

	"github.com/golang/glog"
)

type Level glog.Level

// PrintLogger is a simple interface for loggers that is implemented by both info and error loggers
type PrintLogger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

type infoWriter struct {
	level glog.Level
}

// Write implements the io.Writer interface.
func (w infoWriter) Write(data []byte) (n int, err error) {
	glog.V(w.level).Info(string(data))
	return len(data), nil
}

type errorWriter struct{}

// Write implements the io.Writer interface.
func (w errorWriter) Write(data []byte) (n int, err error) {
	glog.Error(string(data))
	return len(data), nil
}

// InfoLogger is a simple wrapper around glog that provides a way to easily create loggers at a particular log level.
type InfoLogger struct {
	prefix string
}

// newInfoLogger returns a Logger object that provides a V method to log info messages at a particular log level.
func newInfoLogger(prefix string) *InfoLogger {
	return &InfoLogger{
		prefix: prefix,
	}
}

// V returns a standard log.Logger at the particular log level. Logs will only be printed if glog was set up at a level equal to or higher than the level provided to V.
func (l *InfoLogger) V(level Level) *log.Logger {
	return log.New(infoWriter{glog.Level(level)}, l.prefix, 0)
}

// Print prints directly to the info logger regardless of level
func (l *InfoLogger) Print(v ...interface{}) {
	glog.Info(v)
}

// Printf prints directly to the info logger regardless of level
func (l *InfoLogger) Printf(format string, v ...interface{}) {
	glog.Infof(format, v...)
}

type Logger struct {
	Info  *InfoLogger
	Error *log.Logger
}

func New(prefix string) *Logger {
	return &Logger{
		Info:  newInfoLogger(prefix),
		Error: newErrorLogger(prefix),
	}
}

// newErrorLogger returns a standard log.Logger that can be used to write error log messages.
func newErrorLogger(prefix string) *log.Logger {
	return log.New(errorWriter{}, prefix, 0)
}

// PrintMulti logs the highest level message that is at or below the configured glog level
func PrintMulti(logger PrintLogger, msgMap map[Level]string) {
	var level Level
	level = -1
	msg := ""
	for l, m := range msgMap {
		if glog.V(glog.Level(l)) && l > level {
			level = l
			msg = m
		}
	}

	if level > -1 {
		logger.Print(msg)
	}
}

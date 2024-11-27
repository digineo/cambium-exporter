package exporter

import (
	"fmt"
	"log"
)

type logger bool

const logDepth = 2 // log.Output + exporter.{Error,Info,Debug}f

func (logger) Errorf(format string, v ...interface{}) {
	_ = log.Output(logDepth, fmt.Sprintf("[error] "+format, v...))
}

func (logger) Infof(format string, v ...interface{}) {
	_ = log.Output(logDepth, fmt.Sprintf("[info] "+format, v...))
}

func (l logger) Debugf(format string, v ...interface{}) {
	if l {
		_ = log.Output(logDepth, fmt.Sprintf("[debug] "+format, v...))
	}
}

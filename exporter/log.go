package exporter

import (
	"fmt"
	"log"
)

type logger bool

func (logger) Errorf(format string, v ...interface{}) {
	_ = log.Output(2, fmt.Sprintf("[error] "+format, v...))
}

func (logger) Infof(format string, v ...interface{}) {
	_ = log.Output(2, fmt.Sprintf("[info] "+format, v...))
}

func (l logger) Debugf(format string, v ...interface{}) {
	if l {
		_ = log.Output(2, fmt.Sprintf("[debug] "+format, v...))
	}
}

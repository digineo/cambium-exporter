package exporter

import "log"

type logger bool

func (logger) Errorf(format string, v ...interface{}) {
	log.Printf("[error] "+format, v...)
}

func (logger) Infof(format string, v ...interface{}) {
	log.Printf("[info]  "+format, v...)
}

func (l logger) Debugf(format string, v ...interface{}) {
	if l {
		log.Printf("[debug] "+format, v...)
	}
}

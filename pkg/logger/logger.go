package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	Ldate         = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

const (
	DEBUG = iota + 1
	INFO
	WARN
	ERROR
	FATAL
)

type Loginterface interface {
	INIT()
	DEBUG(format string, v ...interface{})
	INFO(format string, v ...interface{})
	WARN(format string, v ...interface{})
	ERROR(format string, v ...interface{})
	FATAL(format string, v ...interface{})
}

type Logx struct {
	LogLevel int
	Logout   io.Writer
	Logflag  int
	Logger   log.Logger
}

func Newlogger(LogLevel int, Logout io.Writer, Logflag int) Loginterface {
	logx := Logx{LogLevel, Logout, Logflag, log.Logger{}}
	var logger Loginterface
	logger = &logx
	logger.INIT()
	return logger
}

func (l *Logx) INIT() {
	l.Logger.SetOutput(l.Logout)
	l.Logger.SetFlags(l.Logflag)
}

func (l *Logx) DEBUG(format string, v ...interface{}) {
	if l.LogLevel <= DEBUG {
		l.Logger.SetPrefix("[DEBUG] ")
		_ = l.Logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logx) INFO(format string, v ...interface{}) {
	if l.LogLevel <= INFO {
		l.Logger.SetPrefix("[INFO] ")
		//Logger.Println(v...)
		_ = l.Logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logx) WARN(format string, v ...interface{}) {
	if l.LogLevel <= WARN {
		l.Logger.SetPrefix("[WARN] ")
		_ = l.Logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logx) ERROR(format string, v ...interface{}) {
	if l.LogLevel <= ERROR {
		l.Logger.SetPrefix("[ERROR] ")
		_ = l.Logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logx) FATAL(format string, v ...interface{}) {
	if l.LogLevel <= FATAL {
		l.Logger.SetPrefix("[FATAL] ")
		_ = l.Logger.Output(2, fmt.Sprintf(format, v...))
		os.Exit(1)
	}
}

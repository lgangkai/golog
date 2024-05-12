package golog

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Logger struct {
	out           *os.File
	debugLogger   *log.Logger
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger

	logToFile   bool
	traceKey    string
	rootPath    string
	logFileName string
	date        int32 // day in year
	maxKeepDays int   // days to keep the log files. -1 means never delete.
	dateLock    sync.Mutex
	fileLock    sync.Mutex
}

func Default() *Logger {
	return NewLogger(true, "logs", "log", "traceKey", 7)
}

func NewLogger(logToFile bool, rootPath string, logFileName string, traceKey string, maxKeepDays int) *Logger {
	out := createFileOfToday(rootPath, logFileName, logToFile)

	debugLogger := log.New(out, "[Debug] ", log.Llongfile|log.LstdFlags)
	infoLogger := log.New(out, "[Info] ", log.Llongfile|log.LstdFlags)
	warningLogger := log.New(out, "[Warning] ", log.Llongfile|log.LstdFlags)
	errorLogger := log.New(out, "[Error] ", log.Llongfile|log.LstdFlags)

	return &Logger{
		out:           out,
		debugLogger:   debugLogger,
		infoLogger:    infoLogger,
		warningLogger: warningLogger,
		errorLogger:   errorLogger,
		logToFile:     logToFile,
		traceKey:      traceKey,
		rootPath:      rootPath,
		logFileName:   logFileName,
		date:          int32(time.Now().YearDay()),
		maxKeepDays:   maxKeepDays,
		dateLock:      sync.Mutex{},
		fileLock:      sync.Mutex{},
	}
}

func createDateFolder(rootPath, date string) error {
	folderPath := filepath.Join(rootPath, date)
	if !folderExists(folderPath) {
		return os.MkdirAll(folderPath, os.ModePerm)
	}
	return nil
}

func createFileOfToday(rootPath string, filename string, logToFile bool) *os.File {
	td := time.Now().Format("2006-01-02")
	_ = createDateFolder(rootPath, td)

	var out *os.File
	if !logToFile {
		out = os.Stderr
	} else {
		out, _ = os.OpenFile(filepath.Join(rootPath, td, filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	}
	return out
}

func folderExists(folderName string) bool {
	_, err := os.Stat(folderName)
	if err != nil && os.IsExist(err) {
		return true
	}
	return false
}

func (l *Logger) updateLogger() {
	l.out.Close()

	out := createFileOfToday(l.rootPath, l.logFileName, l.logToFile)

	l.debugLogger = log.New(out, "[Debug] ", log.Llongfile|log.LstdFlags)
	l.infoLogger = log.New(out, "[Info] ", log.Llongfile|log.LstdFlags)
	l.warningLogger = log.New(out, "[Warning] ", log.Llongfile|log.LstdFlags)
	l.errorLogger = log.New(out, "[Error] ", log.Llongfile|log.LstdFlags)
}

// If date changes, update l.date of today and return true.
func (l *Logger) compareDateAndUpdate() bool {
	nd := int32(time.Now().YearDay())
	// We use double-checked locking to keep thread safety and performance.
	if atomic.LoadInt32(&l.date) != nd {
		l.dateLock.Lock()
		defer l.dateLock.Unlock()
		if l.date != nd {
			l.date = nd
			return true
		}
	}
	return false
}

func (l *Logger) deleteExpiredFiles() {
	es, err := os.ReadDir(l.rootPath)
	if err != nil {
		return
	}
	nt := time.Now().Unix()
	for _, e := range es {
		if e.IsDir() {
			date, _ := time.Parse("2006-01-02", e.Name())
			if nt-date.Unix() >= int64((l.maxKeepDays-1)*24*60*60) {
				os.RemoveAll(filepath.Join(l.rootPath, e.Name()))
				log.Println("folder ", e.Name(), " deleted!")
			}
		}
	}
}

// Debug for debug level
func (l *Logger) Debug(ctx context.Context, args ...interface{}) {
	l.wrappedLog(ctx, l.debugLogger, l.getMsgWithTrace(ctx, args...))
}

// Info for info level
func (l *Logger) Info(ctx context.Context, args ...interface{}) {
	l.wrappedLog(ctx, l.infoLogger, l.getMsgWithTrace(ctx, args...))
}

// Warning for warning level
func (l *Logger) Warning(ctx context.Context, args ...interface{}) {
	l.wrappedLog(ctx, l.warningLogger, l.getMsgWithTrace(ctx, args...))
}

// Error for error level
func (l *Logger) Error(ctx context.Context, args ...interface{}) {
	l.wrappedLog(ctx, l.errorLogger, l.getMsgWithTrace(ctx, args...))
}

func (l *Logger) wrappedLog(ctx context.Context, logger *log.Logger, msg string) {
	if l.compareDateAndUpdate() {
		l.fileLock.Lock()
		l.updateLogger()
		l.deleteExpiredFiles()
		l.fileLock.Unlock()
	}
	_ = logger.Output(3, msg)
}

func (l *Logger) getMsgWithTrace(ctx context.Context, args ...interface{}) string {
	msg := fmt.Sprint(args...)
	msg = strings.Trim(msg, "[]")

	traceDataStr := ""
	traceData, ok := ctx.Value(l.traceKey).(map[string]any)
	if !ok {
		return msg
	}
	for k, v := range traceData {
		traceDataStr = fmt.Sprintf("%v %v %v", traceDataStr, k, v)
	}
	traceDataStr = strings.Trim(traceDataStr, " ")

	msg = fmt.Sprintf("{%v} %v", traceDataStr, msg)
	return msg
}

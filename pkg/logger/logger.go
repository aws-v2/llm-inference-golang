package logger

import (
	"log"
	"os"
)


const (
	InfoLevel = iota
	DebugLevel
	WarningLevel
	ErrorLevel
	TraceLevel
	FatalLevel // its susppose to represent 'log this and then exit with status 1'
)

type Logger struct{
	Level int
	InfoLogger *log.Logger
	DebugLogger *log.Logger
	WarningLogger *log.Logger
	ErrorLogger *log.Logger
	TraceLogger *log.Logger
	FatalLogger *log.Logger
}


var logger *Logger


func init(){
	logger = &Logger{
		Level: InfoLevel,
		InfoLogger: log.New(os.Stdout,"INFO: ", log.LstdFlags|log.Ldate|log.Lshortfile),
		DebugLogger: log.New(os.Stdout,"DEBUG: ", log.LstdFlags|log.Ldate|log.Lshortfile),
		WarningLogger: log.New(os.Stdout,"WARNING: ", log.LstdFlags|log.Ldate|log.Lshortfile),
		TraceLogger: log.New(os.Stdout,"TRACE: ", log.LstdFlags|log.Ldate|log.Lshortfile),
		ErrorLogger: log.New(os.Stdout,"ERROR: ", log.LstdFlags|log.Ldate|log.Lshortfile),
		FatalLogger: log.New(os.Stdout,"FATAL: ", log.LstdFlags|log.Ldate|log.Lshortfile),
	}
}


func SetLevel(level int){
	logger.Level = level
}




func Info(message string, v ...any){
	if logger.Level <= InfoLevel{
		logger.InfoLogger.Println(message)
	}

}
func Debug(message string, v ...any){
		if logger.Level <= DebugLevel{
		logger.InfoLogger.Println(message)
	}
}
func Warn(message string, v ...any){
		if logger.Level <= WarningLevel{
		logger.InfoLogger.Println(message)
	}
}
func Trace(message string, v ...any){
		if logger.Level <= TraceLevel{
		logger.InfoLogger.Println(message)
	}
}
func Error(message string, v ...any){
		if logger.Level <= ErrorLevel{
		logger.InfoLogger.Println(message)
	}
}
func Fatal(message string, v ...any){
		if logger.Level <= FatalLevel{
		logger.InfoLogger.Println(message)
	}
}
package main

import (
    "log"
    "os"
)

var (
    iLog *log.Logger
    wLog *log.Logger
    eLog *log.Logger
)

func init() {
    iLog = log.New(os.Stdout, "[INF] ", log.Ldate|log.Ltime)
    wLog = log.New(os.Stdout, "[WRN] ", log.Ldate|log.Ltime)
    eLog = log.New(os.Stdout, "[ERR] ", log.Ldate|log.Ltime)
}

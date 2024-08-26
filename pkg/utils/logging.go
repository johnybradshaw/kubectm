package utils

import (
    "log"
    "os"
    "time"
    "github.com/fatih/color"
)

// Loggers for different log levels
var (
    InfoLogger    = log.New(os.Stdout, color.GreenString("[INFO] "), 0)
    WarnLogger    = log.New(os.Stdout, color.YellowString("[WARN] "), 0)
    ErrorLogger   = log.New(os.Stderr, color.RedString("[ERROR] "), 0)
    ActionLogger  = log.New(os.Stdout, color.CyanString("[ACTION] "), 0)
)

// iso8601Time returns the current time formatted in ISO 8601
func Iso8601Time() string {
    return time.Now().Format(time.RFC3339)
}
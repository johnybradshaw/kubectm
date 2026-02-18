package utils

import (
    "log"
    "io"
    "os"
    "time"
    "strings"
    "github.com/fatih/color"
)

func init() {
    // Disable the default logger by redirecting it to io.Discard
    log.SetOutput(io.Discard)
}

// Loggers for different log levels
var (
    InfoLogger    = log.New(os.Stdout, color.GreenString("[INFO] "), 0)
    WarnLogger    = log.New(os.Stdout, color.YellowString("[WARN] "), 0)
    ErrorLogger   = log.New(os.Stderr, color.RedString("[ERROR] "), 0)
    ActionLogger  = log.New(os.Stdout, color.CyanString("[ACTION] "), 0)
)

// iso8601Time returns the current time formatted in ISO 8601
//
// It formats the current time in the ISO 8601 format, which is the
// recommended format for timestamps in the Kubernetes API.
func Iso8601Time() string {
    return time.Now().Format(time.RFC3339)
}

// ObfuscateCredential partially hides a credential string, showing only the first and last 4 characters.
func ObfuscateCredential(credential string) string {
    if len(credential) <= 8 {
        return credential // If the credential is too short, return it as is
    }
    return credential[:4] + strings.Repeat("*", len(credential)-8) + credential[len(credential)-4:]
}
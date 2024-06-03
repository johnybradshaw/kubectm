package utils

import (
    "fmt"
    "os"
    "os/exec"

    "github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants"
)

func CheckDependencies() {
    dependencies := []string{"kubectl"}
    for _, dependency := range dependencies {
        _, err := exec.LookPath(dependency)
        if err != nil {
            fmt.Printf("%sError:%s %s is not installed. Please install it and try again.\n", constants.ColorRed, constants.ColorReset, dependency)
            os.Exit(1)
        }
    }
}

func PrintDebug(format string, a ...interface{}) {
    if constants.DebugMode {
        fmt.Printf("%sDEBUG:%s "+format+"\n", append([]interface{}{constants.ColorYellow, constants.ColorReset}, a...)...)
    }
}

func DisplayHelp() {
    fmt.Println("Usage: acc-kubeconfig-cli [--debug] [--help]")
    fmt.Println("Merges the kubeconfig files of all Linode Kubernetes Engine (LKE) clusters into a single file.")
    fmt.Println()
    fmt.Println("Options:")
    fmt.Println("  --debug   Enable debug mode to print additional information during script execution")
    fmt.Println("  --help    Display this help information")
    fmt.Println()
    fmt.Println("Environment Variables:")
    fmt.Printf("  %sLINODE_API_TOKEN%s   Linode API token for authentication\n", constants.ColorCyan, constants.ColorReset)
    fmt.Println()
    fmt.Println("For more information and source code, visit:")
    fmt.Println("https://github.com/johnybradshaw/acc-kubeconfig-cli")
}
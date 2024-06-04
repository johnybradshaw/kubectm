package utils

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants"
)

// CheckDependencies checks if the required dependencies are installed.
//
// It iterates over a list of dependencies and uses the exec.LookPath function
// to check if each dependency is installed on the system. If a dependency is
// not found, it prints an error message and exits the program with a status code
// of 1.
//
// No parameters.
// No return values.
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

// PrintDebug prints a debug message if the debug mode is enabled.
//
// The function takes a format string and optional arguments to format the message.
// If the debug mode is enabled, it prints the debug message with yellow color and
// resets the color to the default after the message.
//
// Parameters:
// - format: the format string for the debug message.
// - a: optional arguments to format the message.
//
// Return:
// None.
func PrintDebug(format string, a ...interface{}) {
	if constants.DebugMode {
		fmt.Printf("%sDEBUG:%s "+format+"\n", append([]interface{}{constants.ColorYellow, constants.ColorReset}, a...)...)
	}
}

// DisplayHelp prints the help information for the acc-kubeconfig-cli command.
//
// It prints the usage information, a description of the command, and options available.
// It also prints information about environment variables and a link to the source code.
//
// No parameters.
// No return value.
func DisplayHelp() {
	fmt.Println("Usage: acc-kubeconfig-cli [--debug] [--help]")
	fmt.Println("Merges the kubeconfig files of all Linode Kubernetes Engine (LKE) clusters into a single file.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --debug   Enable debug mode to print additional information during script execution")
	fmt.Println("  --help    Display this help information")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Printf("  %sLINODE_API_TOKEN%s   Linode API token for authentication (optional)\n", constants.ColorCyan, constants.ColorReset)
	fmt.Println()
	fmt.Println("For more information and source code, visit:")
	fmt.Println("https://github.com/johnybradshaw/acc-kubeconfig-cli")
}

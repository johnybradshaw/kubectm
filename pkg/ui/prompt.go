package ui

import (
    "fmt"
    "kubectm/pkg/credentials"

    "github.com/AlecAivazis/survey/v2"
)

func SelectCredentials(creds []credentials.Credential) []credentials.Credential {
    if len(creds) == 1 {
        fmt.Println("Only one set of credentials found, using it by default.")
        return creds
    }

    selectedIndexes := []int{}
    options := []string{}

    for _, cred := range creds {
        options = append(options, fmt.Sprintf("%s credentials", cred.Provider))
    }

    prompt := &survey.MultiSelect{
        Message: "Multiple credentials found. Please select which ones to use:",
        Options: options,
    }

    err := survey.AskOne(prompt, &selectedIndexes)
    if err != nil {
        fmt.Println("Error during selection:", err)
        return nil
    }

    selectedCreds := []credentials.Credential{}
    for _, index := range selectedIndexes {
        selectedCreds = append(selectedCreds, creds[index])
    }

    return selectedCreds
}
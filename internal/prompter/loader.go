package prompter

import (
	_ "embed"
	"os"
)

//go:embed prompts/profession.txt
var embeddedProfessionPrompt string

var professionPrompt string

func Init(promptFile string) error {

	professionPrompt = embeddedProfessionPrompt

	if promptFile == "" {
		return nil
	}

	data, err := os.ReadFile(promptFile)

	if err != nil {
		return err
	}

	professionPrompt = string(data)

	return nil
}

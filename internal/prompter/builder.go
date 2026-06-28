package prompter

import "strings"

func BuildProfessionPrompt(title, description string) string {
	return strings.NewReplacer(
		"{{title}}", title,
		"{{description}}", description,
	).Replace(professionPrompt)
}

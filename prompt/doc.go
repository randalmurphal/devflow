// Package prompt provides prompt template loading and management.
//
// Core types:
//   - Loader: Loads prompt templates from files or embedded resources
//   - Template: A loaded prompt template with variable substitution
//
// Example usage:
//
//	loader := prompt.NewLoader(prompt.Config{
//	    TemplateDir: ".devflow/prompts",
//	    EmbedFS:     embeddedPrompts,
//	})
//	tmpl, err := loader.Load("generate-spec")
//	result := tmpl.Execute(map[string]string{
//	    "ticket": "TK-421",
//	    "title":  "Add authentication",
//	})
package prompt

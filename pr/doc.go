// Package pr provides pull request operations for GitHub and GitLab.
//
// Core types:
//   - Provider: Interface for creating and managing pull requests
//   - Options: Configuration for creating a pull request
//   - PullRequest: Represents a created pull request with URL and number
//   - Builder: Fluent builder for constructing PR descriptions
//
// Implementations:
//   - GitHubProvider: GitHub PR provider using go-github
//   - GitLabProvider: GitLab MR provider using go-gitlab
//
// Example usage:
//
//	provider, _ := pr.NewGitHubProvider(token, "owner", "repo")
//	pull, err := provider.CreatePR(ctx, pr.Options{
//	    Title: "Add feature",
//	    Body:  "Description",
//	    Base:  "main",
//	    Head:  "feature/my-branch",
//	})
package pr

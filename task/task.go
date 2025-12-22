package task

import (
	"github.com/randalmurphal/llmkit/model"
)

// Type represents the type of task an agent is performing.
// This determines which model tier is appropriate.
type Type string

const (
	// Investigation and architecture - need reasoning
	Investigate  Type = "investigate"
	Architecture Type = "architecture"
	VoteJudge    Type = "vote_judge"

	// Standard dev tasks - default tier
	Implement Type = "implement"
	Review    Type = "review"
	Validate  Type = "validate"
	Fix       Type = "fix"

	// Fast tasks - can use smaller models
	Search    Type = "search"
	Transform Type = "transform"
	Summarize Type = "summarize"
)

// DefaultModelMap maps task types to default models.
var DefaultModelMap = map[Type]model.ModelName{
	Investigate:  model.ModelOpus,
	Architecture: model.ModelOpus,
	VoteJudge:    model.ModelOpus,
	Implement:    model.ModelSonnet,
	Review:       model.ModelSonnet,
	Validate:     model.ModelSonnet,
	Fix:          model.ModelSonnet,
	Search:       model.ModelHaiku,
	Transform:    model.ModelHaiku,
	Summarize:    model.ModelHaiku,
}

// TierForTask returns the appropriate tier for a task type.
func TierForTask(t Type) model.Tier {
	switch t {
	case Investigate, Architecture, VoteJudge:
		return model.TierThinking
	case Search, Transform, Summarize:
		return model.TierFast
	default:
		return model.TierDefault
	}
}

// NewSelector creates a model selector configured for dev workflow tasks.
// It uses the standard task-to-tier mapping.
func NewSelector(opts ...model.SelectorOption) *model.Selector {
	// Prepend the tier function to use Type
	allOpts := append([]model.SelectorOption{
		model.WithTierFunc(func(task any) model.Tier {
			if t, ok := task.(Type); ok {
				return TierForTask(t)
			}
			return model.TierDefault
		}),
	}, opts...)

	return model.NewSelector(allOpts...)
}

// SelectModel selects the appropriate model for a task type.
// Uses the default model map unless overridden.
func SelectModel(t Type) model.ModelName {
	if m, ok := DefaultModelMap[t]; ok {
		return m
	}
	// Fall back to tier-based selection
	switch TierForTask(t) {
	case model.TierThinking:
		return model.ModelOpus
	case model.TierFast:
		return model.ModelHaiku
	default:
		return model.ModelSonnet
	}
}

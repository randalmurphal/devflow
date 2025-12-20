package devflow

import (
	"github.com/rmurphy/flowgraph/pkg/flowgraph/model"
)

// TaskType represents the type of task an agent is performing.
// This determines which model tier is appropriate.
type TaskType string

const (
	// Investigation and architecture - need reasoning
	TaskInvestigate  TaskType = "investigate"
	TaskArchitecture TaskType = "architecture"
	TaskVoteJudge    TaskType = "vote_judge"

	// Standard dev tasks - default tier
	TaskImplement TaskType = "implement"
	TaskReview    TaskType = "review"
	TaskValidate  TaskType = "validate"
	TaskFix       TaskType = "fix"

	// Fast tasks - can use smaller models
	TaskSearch    TaskType = "search"
	TaskTransform TaskType = "transform"
	TaskSummarize TaskType = "summarize"
)

// DefaultModelMap maps task types to default models.
var DefaultModelMap = map[TaskType]model.ModelName{
	TaskInvestigate:  model.ModelOpus,
	TaskArchitecture: model.ModelOpus,
	TaskVoteJudge:    model.ModelOpus,
	TaskImplement:    model.ModelSonnet,
	TaskReview:       model.ModelSonnet,
	TaskValidate:     model.ModelSonnet,
	TaskFix:          model.ModelSonnet,
	TaskSearch:       model.ModelHaiku,
	TaskTransform:    model.ModelHaiku,
	TaskSummarize:    model.ModelHaiku,
}

// TierForTask returns the appropriate tier for a task type.
func TierForTask(task TaskType) model.Tier {
	switch task {
	case TaskInvestigate, TaskArchitecture, TaskVoteJudge:
		return model.TierThinking
	case TaskSearch, TaskTransform, TaskSummarize:
		return model.TierFast
	default:
		return model.TierDefault
	}
}

// NewTaskSelector creates a model selector configured for dev workflow tasks.
// It uses the standard task-to-tier mapping.
func NewTaskSelector(opts ...model.SelectorOption) *model.Selector {
	// Prepend the tier function to use TaskType
	allOpts := append([]model.SelectorOption{
		model.WithTierFunc(func(task any) model.Tier {
			if t, ok := task.(TaskType); ok {
				return TierForTask(t)
			}
			return model.TierDefault
		}),
	}, opts...)

	return model.NewSelector(allOpts...)
}

// SelectModel selects the appropriate model for a task type.
// Uses the default model map unless overridden.
func SelectModel(task TaskType) model.ModelName {
	if m, ok := DefaultModelMap[task]; ok {
		return m
	}
	// Fall back to tier-based selection
	switch TierForTask(task) {
	case model.TierThinking:
		return model.ModelOpus
	case model.TierFast:
		return model.ModelHaiku
	default:
		return model.ModelSonnet
	}
}

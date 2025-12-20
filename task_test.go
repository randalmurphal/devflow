package devflow

import (
	"testing"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph/model"
)

func TestTierForTask(t *testing.T) {
	tests := []struct {
		task         TaskType
		expectedTier model.Tier
	}{
		{TaskInvestigate, model.TierThinking},
		{TaskArchitecture, model.TierThinking},
		{TaskVoteJudge, model.TierThinking},
		{TaskImplement, model.TierDefault},
		{TaskReview, model.TierDefault},
		{TaskValidate, model.TierDefault},
		{TaskFix, model.TierDefault},
		{TaskSearch, model.TierFast},
		{TaskTransform, model.TierFast},
		{TaskSummarize, model.TierFast},
	}

	for _, tt := range tests {
		t.Run(string(tt.task), func(t *testing.T) {
			tier := TierForTask(tt.task)
			if tier != tt.expectedTier {
				t.Errorf("TierForTask(%s) = %s, want %s", tt.task, tier, tt.expectedTier)
			}
		})
	}
}

func TestSelectModel(t *testing.T) {
	tests := []struct {
		task     TaskType
		expected model.ModelName
	}{
		{TaskInvestigate, model.ModelOpus},
		{TaskArchitecture, model.ModelOpus},
		{TaskVoteJudge, model.ModelOpus},
		{TaskImplement, model.ModelSonnet},
		{TaskReview, model.ModelSonnet},
		{TaskValidate, model.ModelSonnet},
		{TaskFix, model.ModelSonnet},
		{TaskSearch, model.ModelHaiku},
		{TaskTransform, model.ModelHaiku},
		{TaskSummarize, model.ModelHaiku},
	}

	for _, tt := range tests {
		t.Run(string(tt.task), func(t *testing.T) {
			m := SelectModel(tt.task)
			if m != tt.expected {
				t.Errorf("SelectModel(%s) = %s, want %s", tt.task, m, tt.expected)
			}
		})
	}
}

func TestSelectModel_Unknown(t *testing.T) {
	// Unknown task should fall back to sonnet (default tier)
	m := SelectModel(TaskType("unknown"))
	if m != model.ModelSonnet {
		t.Errorf("SelectModel(unknown) = %s, want %s", m, model.ModelSonnet)
	}
}

func TestNewTaskSelector(t *testing.T) {
	t.Run("default behavior", func(t *testing.T) {
		selector := NewTaskSelector()

		// Thinking tier tasks get opus
		if got := selector.Select(TaskInvestigate); got != model.ModelOpus {
			t.Errorf("Select(TaskInvestigate) = %s, want %s", got, model.ModelOpus)
		}

		// Default tier tasks get sonnet
		if got := selector.Select(TaskImplement); got != model.ModelSonnet {
			t.Errorf("Select(TaskImplement) = %s, want %s", got, model.ModelSonnet)
		}

		// Fast tier tasks get haiku
		if got := selector.Select(TaskSearch); got != model.ModelHaiku {
			t.Errorf("Select(TaskSearch) = %s, want %s", got, model.ModelHaiku)
		}
	})

	t.Run("with global override", func(t *testing.T) {
		selector := NewTaskSelector(model.WithGlobalOverride(model.ModelHaiku))

		// All tasks get the global override
		if got := selector.Select(TaskInvestigate); got != model.ModelHaiku {
			t.Errorf("Select(TaskInvestigate) = %s, want %s", got, model.ModelHaiku)
		}
		if got := selector.Select(TaskImplement); got != model.ModelHaiku {
			t.Errorf("Select(TaskImplement) = %s, want %s", got, model.ModelHaiku)
		}
	})

	t.Run("with task override", func(t *testing.T) {
		selector := NewTaskSelector(model.WithTaskOverride(TaskReview, model.ModelOpus))

		// Overridden task
		if got := selector.Select(TaskReview); got != model.ModelOpus {
			t.Errorf("Select(TaskReview) = %s, want %s", got, model.ModelOpus)
		}

		// Non-overridden task uses tier func
		if got := selector.Select(TaskImplement); got != model.ModelSonnet {
			t.Errorf("Select(TaskImplement) = %s, want %s", got, model.ModelSonnet)
		}
	})
}

package core

type Policy interface {
	ResetEpisode(*EpisodeContext)
	UpdateEpisode(*EpisodeContext)
	PickAction(*StepContext, State, []Action) Action
	UpdateStep(*StepContext, State, Action, State)
	Reset()
}

type PolicyConstructor interface {
	NewPolicy() Policy
}

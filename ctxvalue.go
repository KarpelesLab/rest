package rest

type ContextRequest int

const (
	BackendURL   ContextRequest = 1
	SkipDebugLog ContextRequest = 2
)

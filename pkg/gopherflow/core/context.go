package core

type ctxKey string

const (
	CtxKeyExecutorId ctxKey = ctxKey("executorId")
	CtxKeyUsername   ctxKey = ctxKey("username")
)

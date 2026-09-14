package core

type ctxKey string

const (
	CtxKeyExecutorId ctxKey = ctxKey("executorId")
	CtxKeyUsername   ctxKey = ctxKey("username")
	CtxKeyWorkerId   ctxKey = ctxKey("worker_id")
)

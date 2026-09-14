// Package core defines the contract a workflow must satisfy.
//
// Implement [Workflow] to define a workflow, embedding [BaseWorkflow] to inherit
// state variable handling. Each state named in StateTransitions is a method on
// the type with the signature:
//
//	func (w *MyWorkflow) MyState(ctx context.Context) (*models.NextState, error)
//
// State methods should be idempotent: the engine retries them according to the
// workflow's RetryConfig, so a method may run more than once for a single
// transition.
package core

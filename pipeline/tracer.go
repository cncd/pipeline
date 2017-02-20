package pipeline

// Tracer handles process tracing.
type Tracer interface {
	Trace(*State) error
}

// TraceFunc type is an adapter to allow the use of ordinary
// functions as a Tracer.
type TraceFunc func(*State) error

// Trace calls f(proc, state).
func (f TraceFunc) Trace(state *State) error {
	return f(state)
}

package queues

import "fsos-server/internal/domain/ports"

var All []QueueDefinition

// QueueDefinition binds a queue topic to a handler. Mirrors the migrations
// registry (external/migrations): the engine owns the plumbing, game code
// registers concrete entries. The handler receives the shared ScriptEngine so a
// consumer can run game logic (dispatch to a Lua script) the same way the
// scheduler and TCP dispatcher do — the engine itself stays domain-agnostic.
type QueueDefinition struct {
	Topic   string
	Handler func(engine ports.ScriptEngine, msg ports.QueueMessage)
}

func Register(q QueueDefinition) {
	All = append(All, q)
}

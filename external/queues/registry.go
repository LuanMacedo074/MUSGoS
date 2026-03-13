package queues

var All []QueueDefinition

type QueueDefinition struct {
	Topic   string
	Handler func(msg []byte)
}

func Register(q QueueDefinition) {
	All = append(All, q)
}

package lingo

type LVoid struct {
    BaseLValue
}

func NewLVoid() *LVoid {
    return &LVoid{BaseLValue{ValueType: VtVoid}}
}

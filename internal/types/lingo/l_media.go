package lingo

type LMedia struct {
    BaseLValue
    Data []byte
}

func NewLMedia(data []byte) *LMedia {
    return &LMedia{
        BaseLValue: BaseLValue{ValueType: VtMedia},
        Data:       data,
    }
}

func (v *LMedia) ToBytes() []byte {
    return v.Data
}

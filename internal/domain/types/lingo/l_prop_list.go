package lingo

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type LPropList struct {
	BaseLValue
	Properties []LValue
	Values     []LValue
}

func NewLPropList() *LPropList {
	return &LPropList{
		BaseLValue: BaseLValue{ValueType: VtPropList},
		Properties: make([]LValue, 0),
		Values:     make([]LValue, 0),
	}
}

func (v *LPropList) addElement(property LValue, value LValue) bool {
	v.Properties = append(v.Properties, property)
	v.Values = append(v.Values, value)
	return true
}

func (v *LPropList) GetElementAt(pos int) LValue {
	if pos < 0 || pos >= v.Count() {
		return NewLVoid()
	}
	return v.Values[pos]
}

func (v *LPropList) GetPropAt(pos int) LValue {
	if pos < 0 || pos >= v.Count() {
		return NewLVoid()
	}

	return v.Properties[pos]
}

func (v *LPropList) GetElement(propName string) (LValue, error) {
	for i, prop := range v.Properties {
		if prop.String() == propName {
			return v.Values[i], nil
		}
	}

	return NewLVoid(), fmt.Errorf("property not found: %s", propName)
}

func (v *LPropList) Count() int {
	return len(v.Properties)
}

func (v *LPropList) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+4 > len(rawBytes) {
		return 0
	}

	numElements := int(binary.BigEndian.Uint32(rawBytes[offset:]))
	chunkSize := 4

	for i := 0; i < numElements; i++ {
		if offset+chunkSize+2 > len(rawBytes) {
			break
		}

		propValue := FromRawBytes(rawBytes, offset+chunkSize)
		propSize := propValue.ExtractFromBytes(rawBytes, offset+chunkSize+2)
		v.Properties = append(v.Properties, propValue)
		chunkSize += 2 + propSize

		if offset+chunkSize+2 > len(rawBytes) {
			break
		}

		elemValue := FromRawBytes(rawBytes, offset+chunkSize)
		elemSize := elemValue.ExtractFromBytes(rawBytes, offset+chunkSize+2)
		v.Values = append(v.Values, elemValue)
		chunkSize += 2 + elemSize
	}

	return chunkSize
}

func (v *LPropList) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("[")

	for i := 0; i < len(v.Properties); i++ {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(v.Properties[i].String())
		buffer.WriteString(": ")
		buffer.WriteString(v.Values[i].String())
	}

	buffer.WriteString("]")
	return buffer.String()
}

func (v *LPropList) GetBytes() []byte {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.BigEndian, VtPropList)
	binary.Write(&buffer, binary.BigEndian, int32(len(v.Properties)))

	for i := 0; i < len(v.Properties); i++ {
		propBytes := v.Properties[i].GetBytes()
		buffer.Write(propBytes)

		valueBytes := v.Values[i].GetBytes()
		buffer.Write(valueBytes)
	}

	return buffer.Bytes()
}

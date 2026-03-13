package lingo

import "encoding/json"

type StoredLValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value,omitempty"`
}

func MarshalLValue(value LValue) ([]byte, error) {
	var sv StoredLValue

	var raw []byte
	var err error

	switch v := value.(type) {
	case *LInteger:
		raw, err = json.Marshal(v.Value)
		sv = StoredLValue{Type: "integer", Value: raw}
	case *LFloat:
		raw, err = json.Marshal(v.Value)
		sv = StoredLValue{Type: "float", Value: raw}
	case *LString:
		raw, err = json.Marshal(v.Value)
		sv = StoredLValue{Type: "string", Value: raw}
	case *LSymbol:
		raw, err = json.Marshal(v.Value)
		sv = StoredLValue{Type: "symbol", Value: raw}
	case *LPropList:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "proplist", Value: raw}
	case *LList:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "list", Value: raw}
	default:
		sv = StoredLValue{Type: "void"}
	}

	if err != nil {
		return nil, err
	}

	return json.Marshal(sv)
}

func UnmarshalLValue(data []byte) (LValue, error) {
	var sv StoredLValue
	if err := json.Unmarshal(data, &sv); err != nil {
		return NewLVoid(), err
	}

	switch sv.Type {
	case "integer":
		var v int32
		if err := json.Unmarshal(sv.Value, &v); err != nil {
			return NewLVoid(), err
		}
		return NewLInteger(v), nil
	case "float":
		var v float64
		if err := json.Unmarshal(sv.Value, &v); err != nil {
			return NewLVoid(), err
		}
		return NewLFloat(v), nil
	case "string":
		var v string
		if err := json.Unmarshal(sv.Value, &v); err != nil {
			return NewLVoid(), err
		}
		return NewLString(v), nil
	case "symbol":
		var v string
		if err := json.Unmarshal(sv.Value, &v); err != nil {
			return NewLVoid(), err
		}
		return NewLSymbol(v), nil
	case "proplist", "list":
		var b []byte
		if err := json.Unmarshal(sv.Value, &b); err != nil {
			return NewLVoid(), err
		}
		return FromRawBytes(b, 0), nil
	default:
		return NewLVoid(), nil
	}
}

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
	case *LPoint:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "point", Value: raw}
	case *LRect:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "rect", Value: raw}
	case *LColor:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "color", Value: raw}
	case *LDate:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "date", Value: raw}
	case *L3dVector:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "3dvector", Value: raw}
	case *L3dTransform:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "3dtransform", Value: raw}
	case *LPicture:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "picture", Value: raw}
	case *LMedia:
		raw, err = json.Marshal(v.GetBytes())
		sv = StoredLValue{Type: "media", Value: raw}
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
	// Binary types are stored as raw GetBytes() output. The type field in the JSON wrapper
	// is informational — FromRawBytes reads the actual type from the first 2 bytes of the binary data.
	case "proplist", "list", "point", "rect", "color", "date", "3dvector", "3dtransform", "picture", "media":
		var b []byte
		if err := json.Unmarshal(sv.Value, &b); err != nil {
			return NewLVoid(), err
		}
		return FromRawBytes(b, 0), nil
	default:
		return NewLVoid(), nil
	}
}

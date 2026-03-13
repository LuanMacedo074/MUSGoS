package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"

	lua "github.com/yuin/gopher-lua"
)

func TestRoundTrip_LInteger(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	original := lingo.NewLInteger(42)
	luaVal := lingo.LValueToLua(L, original)
	result := lingo.LuaToLValue(luaVal)

	intResult, ok := result.(*lingo.LInteger)
	if !ok {
		t.Fatalf("expected *LInteger, got %T", result)
	}
	if intResult.Value != 42 {
		t.Errorf("expected 42, got %d", intResult.Value)
	}
}

func TestRoundTrip_LFloat(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	original := lingo.NewLFloat(3.14)
	luaVal := lingo.LValueToLua(L, original)
	result := lingo.LuaToLValue(luaVal)

	floatResult, ok := result.(*lingo.LFloat)
	if !ok {
		t.Fatalf("expected *LFloat, got %T", result)
	}
	if floatResult.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", floatResult.Value)
	}
}

func TestRoundTrip_LString(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	original := lingo.NewLString("hello")
	luaVal := lingo.LValueToLua(L, original)
	result := lingo.LuaToLValue(luaVal)

	strResult, ok := result.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", result)
	}
	if strResult.Value != "hello" {
		t.Errorf("expected \"hello\", got %q", strResult.Value)
	}
}

func TestRoundTrip_LList(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	original := lingo.NewLList()
	original.Values = append(original.Values, lingo.NewLInteger(1), lingo.NewLString("two"), lingo.NewLInteger(3))

	luaVal := lingo.LValueToLua(L, original)
	result := lingo.LuaToLValue(luaVal)

	listResult, ok := result.(*lingo.LList)
	if !ok {
		t.Fatalf("expected *LList, got %T", result)
	}
	if len(listResult.Values) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(listResult.Values))
	}

	if v, ok := listResult.Values[0].(*lingo.LInteger); !ok || v.Value != 1 {
		t.Errorf("element 0: expected LInteger(1), got %v", listResult.Values[0])
	}
	if v, ok := listResult.Values[1].(*lingo.LString); !ok || v.Value != "two" {
		t.Errorf("element 1: expected LString(\"two\"), got %v", listResult.Values[1])
	}
	if v, ok := listResult.Values[2].(*lingo.LInteger); !ok || v.Value != 3 {
		t.Errorf("element 2: expected LInteger(3), got %v", listResult.Values[2])
	}
}

func TestRoundTrip_LPropList(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	original := lingo.NewLPropList()
	original.Properties = append(original.Properties, lingo.NewLSymbol("name"))
	original.Values = append(original.Values, lingo.NewLString("Alice"))
	original.Properties = append(original.Properties, lingo.NewLSymbol("score"))
	original.Values = append(original.Values, lingo.NewLInteger(100))

	luaVal := lingo.LValueToLua(L, original)
	result := lingo.LuaToLValue(luaVal)

	propResult, ok := result.(*lingo.LPropList)
	if !ok {
		t.Fatalf("expected *LPropList, got %T", result)
	}
	if propResult.Count() != 2 {
		t.Fatalf("expected 2 properties, got %d", propResult.Count())
	}

	nameVal, err := propResult.GetElement("name")
	if err != nil {
		t.Fatalf("property 'name' not found: %v", err)
	}
	if v, ok := nameVal.(*lingo.LString); !ok || v.Value != "Alice" {
		t.Errorf("expected LString(\"Alice\"), got %v", nameVal)
	}

	scoreVal, err := propResult.GetElement("score")
	if err != nil {
		t.Fatalf("property 'score' not found: %v", err)
	}
	if v, ok := scoreVal.(*lingo.LInteger); !ok || v.Value != 100 {
		t.Errorf("expected LInteger(100), got %v", scoreVal)
	}
}

func TestNested_LPropListWithLList(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	innerList := lingo.NewLList()
	innerList.Values = append(innerList.Values, lingo.NewLInteger(10), lingo.NewLInteger(20))

	original := lingo.NewLPropList()
	original.Properties = append(original.Properties, lingo.NewLSymbol("items"))
	original.Values = append(original.Values, innerList)

	luaVal := lingo.LValueToLua(L, original)
	result := lingo.LuaToLValue(luaVal)

	propResult, ok := result.(*lingo.LPropList)
	if !ok {
		t.Fatalf("expected *LPropList, got %T", result)
	}

	itemsVal, err := propResult.GetElement("items")
	if err != nil {
		t.Fatalf("property 'items' not found: %v", err)
	}

	listVal, ok := itemsVal.(*lingo.LList)
	if !ok {
		t.Fatalf("expected *LList, got %T", itemsVal)
	}
	if len(listVal.Values) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(listVal.Values))
	}
	if v, ok := listVal.Values[0].(*lingo.LInteger); !ok || v.Value != 10 {
		t.Errorf("element 0: expected LInteger(10), got %v", listVal.Values[0])
	}
	if v, ok := listVal.Values[1].(*lingo.LInteger); !ok || v.Value != 20 {
		t.Errorf("element 1: expected LInteger(20), got %v", listVal.Values[1])
	}
}

func TestLValueToLua_Nil(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	result := lingo.LValueToLua(L, nil)
	if result != lua.LNil {
		t.Errorf("expected LNil, got %v", result)
	}
}

func TestLValueToLua_LVoid(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	result := lingo.LValueToLua(L, lingo.NewLVoid())
	if result != lua.LNil {
		t.Errorf("expected LNil, got %v", result)
	}
}

func TestLuaToLValue_LNil(t *testing.T) {
	result := lingo.LuaToLValue(lua.LNil)
	if _, ok := result.(*lingo.LVoid); !ok {
		t.Errorf("expected *LVoid, got %T", result)
	}
}

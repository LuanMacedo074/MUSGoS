package outbound

import (
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/crypto/bcrypt"
)

const luaQueryTypeName = "query"

func registerDBModule(L *lua.LState, musMod *lua.LTable, db ports.DBAdapter, qb ports.QueryBuilder, logger ports.Logger) {
	dbMod := L.NewTable()

	// Register query userdata type
	mt := L.NewTypeMetatable(luaQueryTypeName)
	L.SetField(mt, "__index", L.NewFunction(queryIndex))

	// mus.db.table(name) -> query userdata
	if qb != nil {
		// active is the builder mus.db.table(...) routes through. It's the root
		// builder except while inside a mus.db.transaction callback, when it
		// points at the tx-bound builder. Mutating it without a lock is safe:
		// each script runs on its own LState in a single goroutine.
		active := qb
		dbMod.RawSetString("table", L.NewFunction(func(L *lua.LState) int {
			name := L.CheckString(1)
			q := active.Table(name)
			L.Push(wrapQuery(L, q))
			return 1
		}))

		// mus.db.transaction(fn) runs fn with all mus.db.table(...) ops enrolled
		// in a single transaction. Commits when fn returns (anything but false);
		// rolls back if fn returns false or raises an error (error re-raised).
		// Nested transactions are rejected. Note: mus.db player/application/user
		// ops are NOT enrolled — only mus.db.table(...) operations.
		dbMod.RawSetString("transaction", L.NewFunction(func(L *lua.LState) int {
			fn := L.CheckFunction(1)

			txqb, ok := qb.(ports.TransactionalQueryBuilder)
			if !ok {
				L.RaiseError("mus.db.transaction: transactions are not supported by this database")
				return 0
			}
			if _, nested := active.(ports.Tx); nested {
				L.RaiseError("mus.db.transaction: nested transactions are not supported")
				return 0
			}

			tx, err := txqb.Begin()
			if err != nil {
				L.RaiseError("mus.db.transaction: begin failed: %v", err)
				return 0
			}

			active = tx
			L.Push(fn)
			callErr := L.PCall(0, 1, nil)
			active = qb // restore before finalizing so later table() calls hit root

			if callErr != nil {
				_ = tx.Rollback()
				L.RaiseError("mus.db.transaction: rolled back: %v", callErr)
				return 0
			}

			ret := L.Get(-1)
			L.Pop(1)
			if ret == lua.LFalse {
				if rbErr := tx.Rollback(); rbErr != nil {
					L.RaiseError("mus.db.transaction: rollback failed: %v", rbErr)
					return 0
				}
				L.Push(lua.LFalse)
				return 1
			}

			if cErr := tx.Commit(); cErr != nil {
				_ = tx.Rollback()
				L.RaiseError("mus.db.transaction: commit failed: %v", cErr)
				return 0
			}
			L.Push(lua.LTrue)
			return 1
		}))
	}

	if db != nil {
		registerDBPlayerOps(L, dbMod, db)
		registerDBApplicationOps(L, dbMod, db)
		registerDBAdminOps(L, dbMod, db, logger)
	}

	musMod.RawSetString("db", dbMod)
}

func wrapQuery(L *lua.LState, q ports.Query) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = q
	L.SetMetatable(ud, L.GetTypeMetatable(luaQueryTypeName))
	return ud
}

func checkQuery(L *lua.LState) ports.Query {
	ud := L.CheckUserData(1)
	if q, ok := ud.Value.(ports.Query); ok {
		return q
	}
	L.ArgError(1, "query expected")
	return nil
}

func queryIndex(L *lua.LState) int {
	_ = L.CheckUserData(1)
	method := L.CheckString(2)

	switch method {
	case "where":
		L.Push(L.NewFunction(queryWhere))
	case "first":
		L.Push(L.NewFunction(queryFirst))
	case "get":
		L.Push(L.NewFunction(queryGet))
	case "insert":
		L.Push(L.NewFunction(queryInsert))
	case "update":
		L.Push(L.NewFunction(queryUpdate))
	case "delete":
		L.Push(L.NewFunction(queryDelete))
	case "count":
		L.Push(L.NewFunction(queryCount))
	default:
		L.ArgError(2, "unknown query method: "+method)
		return 0
	}
	return 1
}

func queryWhere(L *lua.LState) int {
	q := checkQuery(L)
	col := L.CheckString(2)
	val := luaToGoValue(L.Get(3))
	newQ := q.Where(col, val)
	L.Push(wrapQuery(L, newQ))
	return 1
}

func queryFirst(L *lua.LState) int {
	q := checkQuery(L)
	row, err := q.First()
	if err != nil {
		L.RaiseError("query first failed: %s", err.Error())
		return 0
	}
	if row == nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(rowToLuaTable(L, row))
	return 1
}

func queryGet(L *lua.LState) int {
	q := checkQuery(L)
	rows, err := q.Get()
	if err != nil {
		L.RaiseError("query get failed: %s", err.Error())
		return 0
	}
	tbl := L.NewTable()
	for _, row := range rows {
		tbl.Append(rowToLuaTable(L, row))
	}
	L.Push(tbl)
	return 1
}

func queryInsert(L *lua.LState) int {
	q := checkQuery(L)
	tbl := L.CheckTable(2)
	data := luaTableToMap(L, tbl)
	if err := q.Insert(data); err != nil {
		L.RaiseError("query insert failed: %s", err.Error())
	}
	return 0
}

func queryUpdate(L *lua.LState) int {
	q := checkQuery(L)
	tbl := L.CheckTable(2)
	data := luaTableToMap(L, tbl)
	affected, err := q.Update(data)
	if err != nil {
		L.RaiseError("query update failed: %s", err.Error())
		return 0
	}
	L.Push(lua.LNumber(affected))
	return 1
}

func queryDelete(L *lua.LState) int {
	q := checkQuery(L)
	affected, err := q.Delete()
	if err != nil {
		L.RaiseError("query delete failed: %s", err.Error())
		return 0
	}
	L.Push(lua.LNumber(affected))
	return 1
}

func queryCount(L *lua.LState) int {
	q := checkQuery(L)
	count, err := q.Count()
	if err != nil {
		L.RaiseError("query count failed: %s", err.Error())
		return 0
	}
	L.Push(lua.LNumber(count))
	return 1
}

func luaToGoValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case lua.LBool:
		if v {
			return 1
		}
		return 0
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LNilType:
		return nil
	default:
		return lv.String()
	}
}

func luaTableToMap(L *lua.LState, tbl *lua.LTable) map[string]interface{} {
	data := make(map[string]interface{})
	tbl.ForEach(func(k, v lua.LValue) {
		if str, ok := k.(lua.LString); ok {
			data[string(str)] = luaToGoValue(v)
		}
	})
	return data
}

func rowToLuaTable(L *lua.LState, row ports.QueryResult) *lua.LTable {
	tbl := L.NewTable()
	for k, v := range row {
		switch val := v.(type) {
		case int64:
			tbl.RawSetString(k, lua.LNumber(val))
		case float64:
			tbl.RawSetString(k, lua.LNumber(val))
		case string:
			tbl.RawSetString(k, lua.LString(val))
		case []byte:
			tbl.RawSetString(k, lua.LString(string(val)))
		case nil:
			tbl.RawSetString(k, lua.LNil)
		default:
			tbl.RawSetString(k, lua.LNil)
		}
	}
	return tbl
}

// --- DBPlayer standard ops ---

func registerDBPlayerOps(L *lua.LState, dbMod *lua.LTable, db ports.DBAdapter) {
	dbMod.RawSetString("getPlayerAttribute", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		userID := L.CheckString(2)
		attr := L.CheckString(3)
		val, err := db.GetPlayerAttribute(app, userID, attr)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lingo.LValueToLua(L, val))
		return 1
	}))
	dbMod.RawSetString("setPlayerAttribute", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		userID := L.CheckString(2)
		attr := L.CheckString(3)
		value := lingo.LuaToLValue(L.Get(4))
		if err := db.SetPlayerAttribute(app, userID, attr, value); err != nil {
			L.RaiseError("setPlayerAttribute failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("deletePlayerAttribute", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		userID := L.CheckString(2)
		attr := L.CheckString(3)
		if err := db.DeletePlayerAttribute(app, userID, attr); err != nil {
			L.RaiseError("deletePlayerAttribute failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("getPlayerAttributeNames", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		userID := L.CheckString(2)
		names, err := db.GetPlayerAttributeNames(app, userID)
		if err != nil {
			L.Push(L.NewTable())
			return 1
		}
		tbl := L.NewTable()
		for _, n := range names {
			tbl.Append(lua.LString(n))
		}
		L.Push(tbl)
		return 1
	}))
}

// --- DBApplication standard ops ---

func registerDBApplicationOps(L *lua.LState, dbMod *lua.LTable, db ports.DBAdapter) {
	dbMod.RawSetString("getApplicationAttribute", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		attr := L.CheckString(2)
		val, err := db.GetApplicationAttribute(app, attr)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lingo.LValueToLua(L, val))
		return 1
	}))
	dbMod.RawSetString("setApplicationAttribute", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		attr := L.CheckString(2)
		value := lingo.LuaToLValue(L.Get(3))
		if err := db.SetApplicationAttribute(app, attr, value); err != nil {
			L.RaiseError("setApplicationAttribute failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("deleteApplicationAttribute", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		attr := L.CheckString(2)
		if err := db.DeleteApplicationAttribute(app, attr); err != nil {
			L.RaiseError("deleteApplicationAttribute failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("getApplicationAttributeNames", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		names, err := db.GetApplicationAttributeNames(app)
		if err != nil {
			L.Push(L.NewTable())
			return 1
		}
		tbl := L.NewTable()
		for _, n := range names {
			tbl.Append(lua.LString(n))
		}
		L.Push(tbl)
		return 1
	}))
}

// --- DBAdmin standard ops ---

func registerDBAdminOps(L *lua.LState, dbMod *lua.LTable, db ports.DBAdapter, logger ports.Logger) {
	dbMod.RawSetString("createApplication", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		if err := db.CreateApplication(app); err != nil {
			L.RaiseError("createApplication failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("deleteApplication", L.NewFunction(func(L *lua.LState) int {
		app := L.CheckString(1)
		if err := db.DeleteApplication(app); err != nil {
			L.RaiseError("deleteApplication failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("createUser", L.NewFunction(func(L *lua.LState) int {
		userID := L.CheckString(1)
		password := L.CheckString(2)
		userLevel := L.CheckInt(3)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			L.RaiseError("createUser failed: %s", err.Error())
			return 0
		}
		if err := db.CreateUser(userID, string(hash), userLevel); err != nil {
			L.RaiseError("createUser failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("setPassword", L.NewFunction(func(L *lua.LState) int {
		userID := L.CheckString(1)
		password := L.CheckString(2)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			L.RaiseError("setPassword failed: %s", err.Error())
			return 0
		}
		if err := db.UpdateUserPassword(userID, string(hash)); err != nil {
			L.RaiseError("setPassword failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("deleteUser", L.NewFunction(func(L *lua.LState) int {
		userID := L.CheckString(1)
		if err := db.DeleteUser(userID); err != nil {
			L.RaiseError("deleteUser failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("getUser", L.NewFunction(func(L *lua.LState) int {
		userID := L.CheckString(1)
		user, err := db.GetUser(userID)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		tbl := L.NewTable()
		tbl.RawSetString("id", lua.LNumber(user.ID))
		tbl.RawSetString("username", lua.LString(user.Username))
		tbl.RawSetString("userLevel", lua.LNumber(user.UserLevel))
		L.Push(tbl)
		return 1
	}))
	dbMod.RawSetString("ban", L.NewFunction(func(L *lua.LState) int {
		userID := L.CheckString(1)
		reason := L.CheckString(2)
		user, err := db.GetUser(userID)
		if err != nil {
			L.RaiseError("ban failed: user not found: %s", err.Error())
			return 0
		}
		if err := db.CreateBan(&user.ID, nil, reason, nil); err != nil {
			L.RaiseError("ban failed: %s", err.Error())
		}
		return 0
	}))
	dbMod.RawSetString("revokeBan", L.NewFunction(func(L *lua.LState) int {
		userID := L.CheckString(1)
		user, err := db.GetUser(userID)
		if err != nil {
			L.RaiseError("revokeBan failed: user not found: %s", err.Error())
			return 0
		}
		ban, err := db.GetActiveBanByUserID(user.ID)
		if err != nil {
			L.RaiseError("revokeBan failed: no active ban: %s", err.Error())
			return 0
		}
		if err := db.RevokeBan(ban.ID); err != nil {
			L.RaiseError("revokeBan failed: %s", err.Error())
		}
		return 0
	}))
}

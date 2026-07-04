package outbound

import (
	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

// registerEmailModule builds mus.email. Only registered when an EmailSender
// is configured (SMTP_HOST set), so `mus.email == nil` is the script-side
// "email disabled" signal.
//
//	mus.email.send(to, subject, body) -> ok, err
func registerEmailModule(L *lua.LState, musMod *lua.LTable, emailSender ports.EmailSender, logger ports.Logger) {
	emailMod := L.NewTable()

	emailMod.RawSetString("send", L.NewFunction(func(L *lua.LState) int {
		to := L.CheckString(1)
		subject := L.CheckString(2)
		body := L.CheckString(3)
		err := emailSender.SendEmail(&ports.EmailMessage{
			To:      to,
			Subject: subject,
			Body:    body,
		})
		if err != nil {
			if logger != nil {
				logger.Error("mus.email.send failed", map[string]interface{}{
					"to":      to,
					"subject": subject,
					"error":   err.Error(),
				})
			}
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))

	musMod.RawSetString("email", emailMod)
}

-- ping.lua — HTTP->script bridge smoke test (RFC-021 ticket 01).
--
-- The generic HTTP adapter hands the request to the script as a JSON envelope in
-- mus.getContent() ({ method, path, body }), with NO MUS session established.
-- The script replies with mus.response(json), a JSON envelope carrying the HTTP
-- status and body the adapter writes back.
--
-- Reachable when HTTP_PORT is set, e.g.:
--   curl http://127.0.0.1:$HTTP_PORT/ping
local req = mus.json.decode(mus.getContent())
mus.response(mus.json.encode({
	status = 200,
	body = {
		pong = true,
		method = req.method,
		path = req.path,
	},
}))

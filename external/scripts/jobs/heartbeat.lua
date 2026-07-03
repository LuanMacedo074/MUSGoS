-- jobs/heartbeat — periodic liveness signal proving the scheduler is running.
-- Registered by external/jobs/heartbeat.go (interval 300s). Serves as the
-- reference example for migrating the legacy Fase-10 timers (heals, respawns,
-- crafting delivery) into external/scripts/jobs/<name>.lua.
mus.log.info("scheduler heartbeat", "users_online", mus.server.getUserCount())

-- @job interval=300
-- jobs/heartbeat — periodic liveness signal proving the scheduler is running.
-- The "@job" header above makes it discoverable (RFC: data-driven jobs, no Go
-- registration). Reference example for FSOS jobs under external/scripts/jobs/.
mus.log.info("scheduler heartbeat", "users_online", mus.server.getUserCount())

-- name: GetDevice :one
SELECT id, user_id, created_at, last_seen_at
FROM devices
WHERE id = ?;

-- name: UpsertDevice :exec
INSERT INTO devices (id, user_id, created_at, last_seen_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  user_id = excluded.user_id,
  last_seen_at = excluded.last_seen_at;

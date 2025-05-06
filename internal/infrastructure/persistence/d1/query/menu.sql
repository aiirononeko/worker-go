-- name: ListMenusByDeviceId :many
SELECT
    id,
    device_id,
    name,
    description,
    sort_order,
    created_at,
    updated_at
FROM
    menus
WHERE
    device_id = ?
ORDER BY
    sort_order ASC,
    created_at DESC;

-- name: CreateMenu :one
INSERT INTO menus (
    id,
    device_id,
    name,
    description,
    sort_order,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

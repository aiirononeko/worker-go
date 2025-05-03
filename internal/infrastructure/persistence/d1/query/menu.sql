-- name: ListMenusByUserId :many
SELECT
    id,
    name,
    description,
    sort_order,
    created_at,
    updated_at
FROM
    menus
WHERE
    user_id = ?
ORDER BY
    sort_order ASC,
    created_at DESC;

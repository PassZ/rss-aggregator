-- name: CreateFeedFollow :one
WITH inserted_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT 
    if.id,
    if.created_at,
    if.updated_at,
    if.user_id,
    if.feed_id,
    u.name as user_name,
    f.name as feed_name
FROM inserted_follow if
JOIN users u ON if.user_id = u.id
JOIN feeds f ON if.feed_id = f.id;

-- name: GetFeedFollowsForUser :many
SELECT 
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    ff.feed_id,
    u.name as user_name,
    f.name as feed_name
FROM feed_follows ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f ON ff.feed_id = f.id
WHERE ff.user_id = $1
ORDER BY ff.created_at DESC;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;
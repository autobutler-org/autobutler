-- Channel keys, key grants and channel events (#2417). The Quark stores sealed
-- keys and signatures it can't open or make; clients do the crypto.

-- name: ListChatChannelIDs :many
SELECT id FROM chat_channels ORDER BY id;

-- name: ListChatChannelKeys :many
SELECT * FROM chat_channel_keys WHERE channel_id = sqlc.arg(channel_id) ORDER BY version;

-- name: CreateChatChannelKey :one
INSERT INTO chat_channel_keys (channel_id, version, created_by) VALUES (sqlc.arg(channel_id), sqlc.arg(version), sqlc.arg(created_by)) RETURNING *;

-- ListChatKeyGrantsForUser is one account's grants on a channel, every version.
-- name: ListChatKeyGrantsForUser :many
SELECT * FROM chat_key_grants WHERE channel_id = sqlc.arg(channel_id) AND user_id = sqlc.arg(user_id) ORDER BY version;

-- ListChatKeyGrantRecipients is who holds which version of a channel's key. A
-- NULL user_id is a grant to an account since deleted.
-- name: ListChatKeyGrantRecipients :many
SELECT version, user_id FROM chat_key_grants WHERE channel_id = sqlc.arg(channel_id) ORDER BY version, user_id NULLS FIRST;

-- InsertChatKeyGrant stores a grant unless the account already has one for
-- that version: the first grant wins.
-- name: InsertChatKeyGrant :exec
INSERT INTO chat_key_grants (
    channel_id, version, user_id, sealed_key, granted_by, granter_sign_key, signature
) VALUES (
    sqlc.arg(channel_id), sqlc.arg(version), sqlc.arg(user_id), sqlc.arg(sealed_key), sqlc.arg(granted_by), sqlc.arg(granter_sign_key), sqlc.arg(signature)
) ON CONFLICT (channel_id, version, user_id) DO NOTHING;

-- name: GetChatKeyGrant :one
SELECT * FROM chat_key_grants WHERE channel_id = sqlc.arg(channel_id) AND version = sqlc.arg(version) AND user_id = sqlc.arg(user_id);

-- DeleteChatKeyGrant drops one account's grant of one version, for when the
-- recipient can't open or verify it (#2486).
-- name: DeleteChatKeyGrant :execrows
DELETE FROM chat_key_grants WHERE channel_id = sqlc.arg(channel_id) AND version = sqlc.arg(version) AND user_id = sqlc.arg(user_id);

-- DeleteUserChatKeyGrants drops every grant sealed to an account, for when its
-- X25519 key changes and they can no longer be opened.
-- name: DeleteUserChatKeyGrants :exec
DELETE FROM chat_key_grants WHERE user_id = sqlc.arg(user_id);

-- ListActiveUserChatPublicKeys is every active account's published keys.
-- name: ListActiveUserChatPublicKeys :many
SELECT
    user_chat_keys.user_id,
    user_chat_keys.box_public_key,
    user_chat_keys.sign_public_key
FROM
    user_chat_keys
    JOIN users ON users.id = user_chat_keys.user_id
WHERE
    users.status = 'active';

-- name: CreateChatChannelEvent :one
INSERT INTO chat_channel_events (channel_id, kind, actor_id, payload) VALUES (sqlc.arg(channel_id), sqlc.arg(kind), sqlc.arg(actor_id), sqlc.arg(payload)) RETURNING *;

-- ListChatChannelEvents pages a channel's events oldest first, after an id.
-- name: ListChatChannelEvents :many
SELECT * FROM chat_channel_events WHERE channel_id = sqlc.arg(channel_id) AND id > sqlc.arg(id) ORDER BY id LIMIT CAST(sqlc.arg('limit') AS BIGINT);

-- name: GetChatChannelEvent :one
SELECT * FROM chat_channel_events WHERE id = sqlc.arg(id) AND channel_id = sqlc.arg(channel_id);

-- SignChatChannelEvent attaches the actor's signature to an event, once.
-- name: SignChatChannelEvent :execrows
UPDATE chat_channel_events
SET
    signature = sqlc.arg(signature),
    signer_sign_key = sqlc.arg(signer_sign_key)
WHERE
    id = sqlc.arg(id)
    AND channel_id = sqlc.arg(channel_id)
    AND actor_id = sqlc.arg(actor_id)
    AND signature IS NULL;

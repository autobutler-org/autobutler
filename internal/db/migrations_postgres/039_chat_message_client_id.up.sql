-- A client names each message it sends (#3531), so a retry after a lost
-- response is answered with the stored message rather than stored again. The
-- nonce check of 026 can't see such a retry: the client encrypts the text
-- again, under a fresh nonce. Messages from before this, and from clients
-- that send no id, have none, and a unique index lets any number of NULLs
-- through.
ALTER TABLE chat_messages ADD COLUMN client_message_id TEXT;

CREATE UNIQUE INDEX idx_chat_messages_client_message_id ON chat_messages (channel_id, author_id, client_message_id);

DROP INDEX IF EXISTS idx_chat_messages_client_message_id;
ALTER TABLE chat_messages DROP COLUMN client_message_id;

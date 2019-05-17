BEGIN;

DROP TRIGGER IF EXISTS notifications_broadcast ON notifications;
DROP FUNCTION IF EXISTS notify_sm();
DROP TABLE IF EXISTS notification_labels;
DROP TABLE IF EXISTS notifications;
DROP TYPE IF EXISTS notification_type;

COMMIT;

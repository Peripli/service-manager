BEGIN;

CREATE TYPE notification_type AS ENUM ('CREATED', 'DELETED', 'MODIFIED');

CREATE TABLE notifications
(
  id            varchar(100) PRIMARY KEY,
  resource      varchar(100) NOT NULL,
  type          notification_type NOT NULL,
  platform_id   varchar(100) REFERENCES platforms (id) ON DELETE CASCADE, -- value is platform_id from platforms table or null
  revision      bigserial    NOT NULL,
  created_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  payload       json         NOT NULL -- json with the notification payload
);

CREATE TABLE notification_labels
(
  id              varchar(100) PRIMARY KEY,
  key             varchar(255) NOT NULL CHECK (key <> ''),
  val             varchar(255),
  notification_id varchar(100) NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
  created_at      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, notification_id)
);

CREATE OR REPLACE FUNCTION notify_sm() RETURNS TRIGGER AS $$
  DECLARE
    data json;

  BEGIN
    data = json_build_object(
      'notification_id', NEW.id,
      'platform_id', NEW.platform_id,
      'revision', NEW.revision
    );
    PERFORM pg_notify('notifications', data::text);

    -- Result is ignored since this is an AFTER trigger
    RETURN NULL;
  END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notifications_broadcast
  AFTER INSERT ON notifications
  FOR EACH ROW EXECUTE PROCEDURE notify_sm();

COMMIT;

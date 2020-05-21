BEGIN;

ALTER TABLE operations DROP COLUMN cascade_root_id;
ALTER TABLE operations DROP COLUMN parent_id;

DROP TYPE operation_state;
CREATE TYPE operation_state AS ENUM ('succeeded', 'failed', 'pending');

COMMIT;
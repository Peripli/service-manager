BEGIN;

ALTER TABLE operations DROP COLUMN root;
ALTER TABLE operations DROP COLUMN cascade;
ALTER TABLE operations DROP COLUMN parent;

DROP TYPE operation_state;
CREATE TYPE operation_state AS ENUM ('succeeded', 'failed', 'pending');

COMMIT;
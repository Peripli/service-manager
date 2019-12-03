BEGIN;

DROP TABLE IF EXISTS operations;
DROP TABLE IF EXISTS operation_labels;
DROP TYPE IF EXISTS operation_type;
DROP TYPE IF EXISTS operation_state;
DROP index operations_paging_sequence_uindex;

COMMIT;

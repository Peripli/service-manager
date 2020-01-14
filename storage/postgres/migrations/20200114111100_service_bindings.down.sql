BEGIN;

DROP TABLE IF EXISTS service_bindings;
DROP TABLE IF EXISTS service_bindings_labels;
DROP index service_bindings_paging_sequence_uindex;

COMMIT;
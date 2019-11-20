BEGIN;

DROP TABLE IF EXISTS service_instances;
DROP TABLE IF EXISTS service_instances_labels;
DROP index service_instances_paging_sequence_uindex;

COMMIT;
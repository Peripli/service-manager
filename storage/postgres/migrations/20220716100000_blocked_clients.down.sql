BEGIN;

DROP TABLE IF EXISTS blocked_clients;
DROP TABLE IF EXISTS blocked_clients_labels;
DROP index blocked_clients_paging_sequence_uindex;

COMMIT;
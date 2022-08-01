BEGIN;

DROP TABLE IF EXISTS blocked_clients;
DROP TRIGGER IF EXISTS value_insert;
DROP FUNCTION IF EXISTS new_blocked_client();
DROP TABLE IF EXISTS blocked_clients_labels;
DROP index blocked_clients_paging_sequence_uindex;

COMMIT;
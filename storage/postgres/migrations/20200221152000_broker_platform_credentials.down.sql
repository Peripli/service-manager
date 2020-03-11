BEGIN;

DROP TABLE IF EXISTS broker_platform_credentials;
DROP TABLE IF EXISTS broker_platform_credential_labels;
DROP index broker_platform_credentials_paging_sequence_uindex;

COMMIT;
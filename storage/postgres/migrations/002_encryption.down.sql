BEGIN;

REVOKE ALL ON SCHEMA vault FROM access_user;

DROP TABLE IF EXISTS vault.safe;
DROP SCHEMA IF EXISTS vault;

REVOKE CONNECT ON DATABASE postgres FROM access_user;
REVOKE access_user FROM guard;
DROP ROLE IF EXISTS access_user;

COMMIT;
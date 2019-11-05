BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS brokers_paging_sequence_uindex
    on brokers (paging_sequence);
CREATE UNIQUE INDEX IF NOT EXISTS platforms_paging_sequence_uindex
    on platforms (paging_sequence);
CREATE UNIQUE INDEX IF NOT EXISTS service_offerings_paging_sequence_uindex
    on service_offerings (paging_sequence);
CREATE UNIQUE INDEX IF NOT EXISTS service_plans_paging_sequence_uindex
    on service_plans (paging_sequence);
CREATE UNIQUE INDEX IF NOT EXISTS visibilities_paging_sequence_uindex
    on visibilities (paging_sequence);
CREATE UNIQUE INDEX IF NOT EXISTS notifications_paging_sequence_uindex
    on notifications (paging_sequence);

COMMIT;
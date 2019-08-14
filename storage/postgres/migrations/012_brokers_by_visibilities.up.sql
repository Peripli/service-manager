BEGIN;

CREATE OR REPLACE VIEW broker_visibilities AS
    SELECT brokers.id, brokers.created_at, brokers.updated_at, brokers.description,
                    brokers.broker_url, brokers.catalog, brokers.name, brokers.username,
                    brokers.password, visibilities.platform_id
            FROM visibilities
            JOIN service_plans ON visibilities.service_plan_id=service_plans.id
            JOIN service_offerings ON service_plans.service_offering_id=service_offerings.id
            JOIN brokers ON service_offerings.broker_id=brokers.id;

END;
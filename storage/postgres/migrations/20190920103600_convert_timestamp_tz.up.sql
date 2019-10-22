BEGIN ;

ALTER TABLE brokers ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE brokers ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE broker_labels ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE broker_labels ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

ALTER TABLE notifications ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE notifications ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE notification_labels ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE notification_labels ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

ALTER TABLE platforms ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE platforms ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE platforms ALTER last_active TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE platform_labels ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE platform_labels ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

ALTER TABLE service_offerings ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_offerings ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_offering_labels ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_offering_labels ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

ALTER TABLE service_plans ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_plans ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_plan_labels ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_plan_labels ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

ALTER TABLE visibilities ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE visibilities ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE visibility_labels ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE visibility_labels ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

ALTER TABLE safe ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE safe ALTER updated_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';

END;
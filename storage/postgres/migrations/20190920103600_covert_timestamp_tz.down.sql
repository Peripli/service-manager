BEGIN ;

ALTER TABLE brokers ALTER created_at TYPE timestamp;
ALTER TABLE brokers ALTER updated_at TYPE timestamp;
ALTER TABLE broker_labels ALTER created_at TYPE timestamp;
ALTER TABLE broker_labels ALTER updated_at TYPE timestamp;

ALTER TABLE notifications ALTER created_at TYPE timestamp;
ALTER TABLE notifications ALTER updated_at TYPE timestamp;
ALTER TABLE notification_labels ALTER created_at TYPE timestamp;
ALTER TABLE notification_labels ALTER updated_at TYPE timestamp;

ALTER TABLE platforms ALTER created_at TYPE timestamp;
ALTER TABLE platforms ALTER updated_at TYPE timestamp;
ALTER TABLE platforms ALTER last_active TYPE timestamp;
ALTER TABLE platform_labels ALTER created_at TYPE timestamp;
ALTER TABLE platform_labels ALTER updated_at TYPE timestamp;

ALTER TABLE service_offerings ALTER created_at TYPE timestamp;
ALTER TABLE service_offerings ALTER updated_at TYPE timestamp;
ALTER TABLE service_offering_labels ALTER created_at TYPE timestamp;
ALTER TABLE service_offering_labels ALTER updated_at TYPE timestamp;

ALTER TABLE service_plans ALTER created_at TYPE timestamp;
ALTER TABLE service_plans ALTER updated_at TYPE timestamp;
ALTER TABLE service_plan_labels ALTER created_at TYPE timestamp;
ALTER TABLE service_plan_labels ALTER updated_at TYPE timestamp;

ALTER TABLE visibilities ALTER created_at TYPE timestamp;
ALTER TABLE visibilities ALTER updated_at TYPE timestamp;
ALTER TABLE visibility_labels ALTER created_at TYPE timestamp;
ALTER TABLE visibility_labels ALTER updated_at TYPE timestamp;

ALTER TABLE safe ALTER created_at TYPE timestamp;
ALTER TABLE safe ALTER updated_at TYPE timestamp;

END;
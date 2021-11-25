BEGIN;

ALTER TABLE visibility_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE visibility_labels DROP CONSTRAINT visibility_labels_val_check;
ALTER TABLE broker_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE broker_labels DROP CONSTRAINT broker_labels_val_check;
ALTER TABLE platform_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE platform_labels DROP CONSTRAINT platform_labels_val_check;
ALTER TABLE service_offering_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE service_offering_labels DROP CONSTRAINT service_offering_labels_val_check;
ALTER TABLE service_plan_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE service_plan_labels DROP CONSTRAINT service_plan_labels_val_check;
ALTER TABLE notification_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE notification_labels DROP CONSTRAINT notification_labels_val_check;
ALTER TABLE service_instance_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE service_instance_labels DROP CONSTRAINT service_instance_labels_val_check;
ALTER TABLE service_binding_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE service_binding_labels DROP CONSTRAINT service_binding_labels_val_check;
ALTER TABLE broker_platform_credential_labels ALTER COLUMN val DROP NOT NULL;
ALTER TABLE broker_platform_credential_labels DROP CONSTRAINT broker_platform_credential_labels_val_check;

COMMIT;
BEGIN;

ALTER TABLE visibility_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE visibility_labels ADD CONSTRAINT visibility_labels_val_check CHECK (val <> '');
ALTER TABLE broker_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE broker_labels ADD CONSTRAINT broker_labels_val_check CHECK (val <> '');
ALTER TABLE platform_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE platform_labels ADD CONSTRAINT platform_labels_val_check CHECK (val <> '');
ALTER TABLE service_offering_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE service_offering_labels ADD CONSTRAINT service_offering_labels_val_check CHECK (val <> '');
ALTER TABLE service_plan_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE service_plan_labels ADD CONSTRAINT service_plan_labels_val_check CHECK (val <> '');
ALTER TABLE notification_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE notification_labels ADD CONSTRAINT notification_labels_val_check CHECK (val <> '');
ALTER TABLE service_instance_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE service_instance_labels ADD CONSTRAINT service_instance_labels_val_check CHECK (val <> '');
ALTER TABLE service_binding_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE service_binding_labels ADD CONSTRAINT service_binding_labels_val_check CHECK (val <> '');
ALTER TABLE broker_platform_credential_labels ALTER COLUMN val SET NOT NULL;
ALTER TABLE broker_platform_credential_labels ADD CONSTRAINT broker_platform_credential_labels_val_check CHECK (val <> '');

COMMIT;
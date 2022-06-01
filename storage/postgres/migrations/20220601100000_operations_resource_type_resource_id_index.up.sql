CREATE INDEX IF NOT EXISTS operations_resource_type_resource_id_index
    on operations (resource_type, resource_id);


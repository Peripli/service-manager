CREATE TRIGGER blocked_clients_notify_event
    AFTER INSERT OR DELETE ON blocked_clients
    FOR EACH ROW EXECUTE PROCEDURE notify_event()
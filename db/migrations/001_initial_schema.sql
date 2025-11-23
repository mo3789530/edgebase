-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    device_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_name TEXT NOT NULL,
    device_type TEXT NOT NULL,
    location TEXT,
    registered_at TIMESTAMPTZ DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    status TEXT DEFAULT 'active',
    metadata JSONB
);

-- Telemetry data table
CREATE TABLE IF NOT EXISTS telemetry_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(device_id),
    sensor_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    data_type TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit TEXT,
    metadata JSONB,
    version INT DEFAULT 1,
    synced_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_device_time ON telemetry_data(device_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_time ON telemetry_data(sensor_id, timestamp DESC);

-- Commands table
CREATE TABLE IF NOT EXISTS commands (
    command_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(device_id),
    command_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT now(),
    delivered_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    result JSONB
);

CREATE INDEX IF NOT EXISTS idx_device_status ON commands(device_id, status, created_at);

-- Sync status table
CREATE TABLE IF NOT EXISTS sync_status (
    device_id UUID PRIMARY KEY REFERENCES devices(device_id),
    last_sync_at TIMESTAMPTZ,
    last_sync_status TEXT,
    pending_records_count INT DEFAULT 0,
    total_synced_records BIGINT DEFAULT 0,
    last_error TEXT,
    updated_at TIMESTAMPTZ DEFAULT now()
);

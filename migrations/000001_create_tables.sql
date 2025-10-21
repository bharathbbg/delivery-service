CREATE TABLE IF NOT EXISTS deliveries (
    id VARCHAR(36) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    tracking_number VARCHAR(50) UNIQUE NOT NULL,
    courier_id VARCHAR(36),
    estimated_delivery_time TIMESTAMP NOT NULL,
    actual_delivery_time TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS delivery_addresses (
    delivery_id VARCHAR(36) PRIMARY KEY,
    street VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL,
    zip_code VARCHAR(20) NOT NULL,
    FOREIGN KEY (delivery_id) REFERENCES deliveries(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS delivery_events (
    id VARCHAR(36) PRIMARY KEY,
    delivery_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    location VARCHAR(100),
    description TEXT,
    timestamp TIMESTAMP NOT NULL,
    FOREIGN KEY (delivery_id) REFERENCES deliveries(id) ON DELETE CASCADE
);

CREATE INDEX delivery_tracking_idx ON deliveries(tracking_number);
CREATE INDEX delivery_order_idx ON deliveries(order_id);
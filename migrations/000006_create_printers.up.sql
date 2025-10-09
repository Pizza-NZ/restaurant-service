CREATE TABLE IF NOT EXISTS printers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('thermal', 'kitchen', 'receipt', 'other')),
    ip_address VARCHAR(45) NULL,    -- Can be IP address or hostname
    port INT NULL,                  -- Port number if applicable
    model VARCHAR(100) NULL,        -- Printer model
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS displays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('kitchen', 'customer', 'other')),
    ip_address VARCHAR(45) NULL,    -- Can be IP address or hostname
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Update stations table to reference printer and display IDs
ALTER TABLE stations 
ADD CONSTRAINT fk_stations_printer 
FOREIGN KEY (printer_id) REFERENCES printers(id);

ALTER TABLE stations 
ADD CONSTRAINT fk_stations_display 
FOREIGN KEY (display_id) REFERENCES displays(id);
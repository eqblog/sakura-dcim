-- Add MAC address field to servers for PXE-based inventory scanning
ALTER TABLE servers ADD COLUMN IF NOT EXISTS mac_address VARCHAR(20) NOT NULL DEFAULT '';

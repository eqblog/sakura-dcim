-- Phase 6: Switch Automation
-- Add SSH/Netconf credentials and vendor type to switches

ALTER TABLE switches
    ADD COLUMN vendor       VARCHAR(50)  NOT NULL DEFAULT '',
    ADD COLUMN ssh_user     VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN ssh_pass     VARCHAR(512) NOT NULL DEFAULT '',
    ADD COLUMN ssh_port     INT          NOT NULL DEFAULT 22,
    ADD COLUMN model        VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW();

-- Add VLAN and status tracking to switch ports
ALTER TABLE switch_ports
    ADD COLUMN vlan_id      INT          NOT NULL DEFAULT 0,
    ADD COLUMN admin_status VARCHAR(20)  NOT NULL DEFAULT 'up',
    ADD COLUMN oper_status  VARCHAR(20)  NOT NULL DEFAULT 'unknown',
    ADD COLUMN last_polled  TIMESTAMPTZ;

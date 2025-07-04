-- DNS Server Database Schema
-- This file creates the necessary tables and indexes for the DNS server

-- Create the dns_records table
CREATE TABLE IF NOT EXISTS dns_records (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,           -- Domain name (e.g., "example.com", "www.example.com")
    record_type VARCHAR(10) NOT NULL,     -- DNS record type (A, AAAA, CNAME, TXT, MX, NS)
    target TEXT NOT NULL,                 -- Target value (IP address, domain name, text, etc.)
    ttl INTEGER NOT NULL DEFAULT 300,     -- Time to live in seconds
    priority INTEGER NOT NULL DEFAULT 0,  -- Priority for MX records, general priority for others
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    serial INTEGER DEFAULT NULL,
    mbox TEXT DEFAULT NULL,
    refresh INTEGER DEFAULT NULL,
    retry INTEGER DEFAULT NULL,
    expire INTEGER DEFAULT NULL,
    minttl INTEGER DEFAULT NULL,
    weight INTEGER DEFAULT NULL,
    port SMALLINT DEFAULT NULL,
    tag TEXT DEFAULT NULL,
    
    -- Constraints
    CONSTRAINT dns_records_ttl_check CHECK (ttl >= 0 AND ttl <= 2147483647),
    CONSTRAINT dns_records_priority_check CHECK (priority >= 0),
    CONSTRAINT dns_records_name_check CHECK (LENGTH(name) > 0),
    CONSTRAINT dns_records_target_check CHECK (LENGTH(target) > 0),
    CONSTRAINT dns_records_type_check CHECK (record_type IN ('A', 'AAAA', 'CNAME', 'TXT', 'MX', 'NS', 'SOA', 'PTR', 'SRV', 'CAA'))
);

-- Create indexes for performance
-- Primary lookup index: name + record_type (case-insensitive name)
CREATE INDEX IF NOT EXISTS idx_dns_records_name_type 
    ON dns_records(LOWER(name), record_type);

-- Index for name-only lookups (case-insensitive)
CREATE INDEX IF NOT EXISTS idx_dns_records_name 
    ON dns_records(LOWER(name));

-- Index for record type lookups
CREATE INDEX IF NOT EXISTS idx_dns_records_type 
    ON dns_records(record_type);

-- Index for priority ordering within name/type combinations
CREATE INDEX IF NOT EXISTS idx_dns_records_name_type_priority 
    ON dns_records(LOWER(name), record_type, priority DESC);

-- Index for efficient cleanup and management queries
CREATE INDEX IF NOT EXISTS idx_dns_records_created_at 
    ON dns_records(created_at);

CREATE INDEX IF NOT EXISTS idx_dns_records_updated_at 
    ON dns_records(updated_at);

-- Index for CAA records
CREATE INDEX IF NOT EXISTS idx_dns_records_caa_tag 
    ON dns_records(LOWER(name), record_type, tag) 
    WHERE record_type = 'CAA';

-- Function to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_dns_records_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at on record modifications
DROP TRIGGER IF EXISTS trigger_dns_records_updated_at ON dns_records;
CREATE TRIGGER trigger_dns_records_updated_at
    BEFORE UPDATE ON dns_records
    FOR EACH ROW
    EXECUTE FUNCTION update_dns_records_updated_at();

-- Insert sample DNS records for testing and development
INSERT INTO dns_records (name, record_type, target, ttl, priority) VALUES
    -- Basic A records for test.internal domain
    ('test.internal', 'A', '10.0.0.10', 300, 10),
    ('www.test.internal', 'A', '10.0.0.10', 300, 10),
    ('mail.test.internal', 'A', '10.0.0.20', 300, 10),
    ('api.test.internal', 'A', '10.0.0.30', 300, 10),
    
    -- AAAA records (IPv6)
    ('test.internal', 'AAAA', 'fd00::1', 300, 10),
    ('www.test.internal', 'AAAA', 'fd00::1', 300, 10),
    
    -- CNAME records
    ('ftp.test.internal', 'CNAME', 'www.test.internal', 300, 10),
    ('blog.test.internal', 'CNAME', 'www.test.internal', 300, 10),
    
    -- MX records (priority matters here - lower number = higher priority)
    ('test.internal', 'MX', 'mail.test.internal', 300, 10),
    ('test.internal', 'MX', 'mail2.test.internal', 300, 20),
    
    -- TXT records
    ('test.internal', 'TXT', 'v=spf1 include:_spf.test.internal ~all', 300, 10),
    ('_dmarc.test.internal', 'TXT', 'v=DMARC1; p=none; ruf=mailto:dmarc@test.internal', 300, 10),
    
    -- NS records
    ('test.internal', 'NS', 'ns1.test.internal', 86400, 10),
    ('test.internal', 'NS', 'ns2.test.internal', 86400, 10),
    
    -- Additional test domains with creative names
    ('errant-dns-test.internal', 'A', '10.0.1.10', 60, 10),
    ('short-ttl.internal', 'A', '10.0.1.20', 30, 10),
    ('long-ttl.internal', 'A', '10.0.1.30', 3600, 10),
    
    -- Fun test domains
    ('dns-cache-test.internal', 'A', '10.0.2.10', 120, 10),
    
    -- Priority test records - tied priorities for testing round-robin
    ('priority-test.internal', 'A', '10.0.2.20', 300, 10),  -- Highest priority
    ('priority-test.internal', 'A', '10.0.2.21', 300, 10),  -- Tied for highest priority
    ('priority-test.internal', 'A', '10.0.2.22', 300, 10),  -- Tied for highest priority
    ('priority-test.internal', 'A', '10.0.2.30', 300, 20),  -- Lower priority (should not be returned)
    
    -- Round-robin test records - multiple records with same priority
    ('round-robin.internal', 'A', '10.0.3.10', 300, 10),
    ('round-robin.internal', 'A', '10.0.3.11', 300, 10),
    ('round-robin.internal', 'A', '10.0.3.12', 300, 10),
    ('round-robin.internal', 'A', '10.0.3.13', 300, 10),
    
    -- Wildcard preparation domains (for future wildcard testing)
    ('wildcard-parent.internal', 'A', '10.0.4.10', 300, 10),
    ('sub1.wildcard-parent.internal', 'A', '10.0.4.11', 300, 10),
    ('sub2.wildcard-parent.internal', 'A', '10.0.4.12', 300, 10)
ON CONFLICT DO NOTHING;

-- Add SOA record example:
INSERT INTO dns_records (name, record_type, target, ttl, priority, mbox, serial, refresh, retry, expire, minttl) VALUES
    ('test.internal', 'SOA', 'ns1.test.internal', 86400, 1, 'admin.test.internal', 2024062301, 7200, 3600, 604800, 300);

-- Add SRV record examples:
INSERT INTO dns_records (name, record_type, target, ttl, priority, weight, port) VALUES
    ('_http._tcp.test.internal', 'SRV', 'web1.test.internal', 300, 10, 5, 80),
    ('_http._tcp.test.internal', 'SRV', 'web2.test.internal', 300, 10, 5, 80);

-- Add PTR record example:
INSERT INTO dns_records (name, record_type, target, ttl, priority) VALUES
    ('10.0.0.10.in-addr.arpa', 'PTR', 'test.internal', 300, 10);

-- Additional PTR records for reverse DNS
INSERT INTO dns_records (name, record_type, target, ttl, priority) VALUES
('20.0.0.10.in-addr.arpa', 'PTR', 'mail.test.internal', 300, 10),
('30.0.0.10.in-addr.arpa', 'PTR', 'api.test.internal', 300, 10),
-- IPv6 PTR record (fd00::1 reverse)
('1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.d.f.ip6.arpa', 'PTR', 'test.internal', 300, 10);

-- Additional SRV records for service discovery
INSERT INTO dns_records (name, record_type, target, ttl, priority, weight, port) VALUES
-- HTTPS service
('_https._tcp.test.internal', 'SRV', 'www.test.internal', 300, 10, 5, 443),
-- SMTP service  
('_smtp._tcp.test.internal', 'SRV', 'mail.test.internal', 300, 10, 5, 25),
-- IMAP service
('_imap._tcp.test.internal', 'SRV', 'mail.test.internal', 300, 10, 5, 143),
-- SIP service
('_sip._tcp.test.internal', 'SRV', 'sip.test.internal', 300, 10, 5, 5060),
-- LDAP service  
('_ldap._tcp.test.internal', 'SRV', 'ldap.test.internal', 300, 10, 5, 389),
-- Multiple SRV with different priorities (for priority testing)
('_web._tcp.test.internal', 'SRV', 'web1.test.internal', 300, 10, 5, 80),
('_web._tcp.test.internal', 'SRV', 'web2.test.internal', 300, 20, 5, 80),
('_web._tcp.test.internal', 'SRV', 'web3.test.internal', 300, 30, 5, 80),
-- Multiple SRV with same priority but different weights (for weight testing)
('_cluster._tcp.test.internal', 'SRV', 'node1.test.internal', 300, 10, 10, 8080),
('_cluster._tcp.test.internal', 'SRV', 'node2.test.internal', 300, 10, 20, 8080),
('_cluster._tcp.test.internal', 'SRV', 'node3.test.internal', 300, 10, 30, 8080);

-- Additional A records needed for SRV targets
INSERT INTO dns_records (name, record_type, target, ttl, priority) VALUES
('sip.test.internal', 'A', '10.0.0.40', 300, 10),
('ldap.test.internal', 'A', '10.0.0.50', 300, 10),
('web1.test.internal', 'A', '10.0.5.10', 300, 10),
('web2.test.internal', 'A', '10.0.5.20', 300, 10),
('web3.test.internal', 'A', '10.0.5.30', 300, 10),
('node1.test.internal', 'A', '10.0.6.10', 300, 10),
('node2.test.internal', 'A', '10.0.6.20', 300, 10),
('node3.test.internal', 'A', '10.0.6.30', 300, 10);

-- Additional NS records for the hosts that need A records
INSERT INTO dns_records (name, record_type, target, ttl, priority) VALUES
('ns1.test.internal', 'A', '10.0.0.100', 86400, 10),
('ns2.test.internal', 'A', '10.0.0.101', 86400, 10),
('mail2.test.internal', 'A', '10.0.0.21', 300, 10);


-- Add sample CAA records for testing
INSERT INTO dns_records (name, record_type, target, ttl, priority, tag) VALUES
    -- Basic CAA records for test.internal
    ('test.internal', 'CAA', 'letsencrypt.org', 300, 0, 'issue'),
    ('test.internal', 'CAA', 'digicert.com', 300, 0, 'issue'),
    ('test.internal', 'CAA', 'mailto:security@test.internal', 300, 0, 'iodef'),
    
    -- Wildcard restriction example
    ('secure.test.internal', 'CAA', ';', 300, 0, 'issuewild'),
    
    -- Critical flag example
    ('critical.test.internal', 'CAA', 'sectigo.com', 300, 128, 'issue'),
    
    -- HTTPS iodef example
    ('api.test.internal', 'CAA', 'https://test.internal/caa-incident', 300, 0, 'iodef'),
    
    -- Multiple CAs allowed with different priorities
    ('multi-ca.test.internal', 'CAA', 'letsencrypt.org', 300, 10, 'issue'),
    ('multi-ca.test.internal', 'CAA', 'digicert.com', 300, 20, 'issue'),
    
    -- Deny all example
    ('no-certs.test.internal', 'CAA', ';', 300, 0, 'issue')
ON CONFLICT DO NOTHING;

-- Create a view for easier record management and reporting
CREATE OR REPLACE VIEW dns_records_view AS
SELECT 
    id,
    name,
    record_type,
    target,
    ttl,
    priority,
    created_at,
    updated_at,
    mbox,
    serial,
    refresh,
    retry,
    expire,
    minttl,
    weight,
    port,
    tag,
    -- Additional computed columns for convenience
    CASE 
        WHEN record_type = 'MX' THEN priority
        WHEN record_type = 'CAA' THEN priority
        ELSE NULL 
    END as record_priority,
    CASE 
        WHEN record_type = 'CAA' THEN 
            CASE priority 
                WHEN 0 THEN 'non-critical'
                WHEN 128 THEN 'critical'
                ELSE 'invalid'
            END
        ELSE NULL
    END as caa_flag_description,
    EXTRACT(EPOCH FROM (updated_at - created_at)) as age_seconds
FROM dns_records
ORDER BY name, record_type, priority DESC;

-- Function to safely add DNS records with validation
CREATE OR REPLACE FUNCTION add_dns_record(
    p_name VARCHAR(255),
    p_record_type VARCHAR(10),
    p_target TEXT,
    p_ttl INTEGER DEFAULT 300,
    p_priority INTEGER DEFAULT 0,
    p_mbox TEXT DEFAULT NULL,
    p_serial INTEGER DEFAULT NULL,
    p_refresh INTEGER DEFAULT NULL,
    p_retry INTEGER DEFAULT NULL,
    p_expire INTEGER DEFAULT NULL,
    p_minttl INTEGER DEFAULT NULL,
    p_weight INTEGER DEFAULT NULL,
    p_port SMALLINT DEFAULT NULL,
    p_tag TEXT DEFAULT NULL
) RETURNS INTEGER AS $$
DECLARE
    record_id INTEGER;
BEGIN
    -- Basic validation
    IF p_name IS NULL OR LENGTH(p_name) = 0 THEN
        RAISE EXCEPTION 'Name cannot be empty';
    END IF; 

    -- Validate record type
    IF p_record_type IS NULL OR p_record_type NOT IN ('A', 'AAAA', 'CNAME', 'TXT', 'MX', 'NS', 'SOA', 'PTR', 'SRV', 'CAA') THEN
        RAISE EXCEPTION 'Invalid record type: %', p_record_type;
    END IF;
    
    -- Validate target
    IF p_target IS NULL OR LENGTH(p_target) = 0 THEN
        RAISE EXCEPTION 'Target cannot be empty';
    END IF;
    
    -- Validate TTL
    IF p_ttl < 0 OR p_ttl > 2147483647 THEN
        RAISE EXCEPTION 'TTL must be between 0 and 2147483647';
    END IF;

    -- Validate SOA record
    IF p_record_type = 'SOA' THEN
        IF p_mbox IS NULL OR LENGTH(p_mbox) = 0 THEN
            RAISE EXCEPTION 'Mbox cannot be empty for SOA records';
        END IF;
    END IF;
    
    -- Validate SRV record
    IF p_record_type = 'SRV' THEN
        IF p_weight IS NULL OR p_weight < 0 OR p_weight > 65535 THEN
            RAISE EXCEPTION 'Weight must be between 0 and 65535';
        END IF;
    END IF;
    
    -- Validate CAA record
    IF p_record_type = 'CAA' THEN
        IF p_tag IS NULL OR LENGTH(p_tag) = 0 THEN
            RAISE EXCEPTION 'CAA records require a tag';
        END IF;
        
        IF LOWER(p_tag) NOT IN ('issue', 'issuewild', 'iodef') THEN
            RAISE EXCEPTION 'CAA tag must be issue, issuewild, or iodef';
        END IF;
        
        IF p_priority NOT IN (0, 128) THEN
            RAISE EXCEPTION 'CAA flag (priority) must be 0 or 128';
        END IF;
    END IF;

    -- Insert the record
    INSERT INTO dns_records (name, record_type, target, ttl, priority, mbox, serial, refresh, retry, expire, minttl, weight, port, tag)
    VALUES (LOWER(p_name), UPPER(p_record_type), p_target, p_ttl, p_priority, p_mbox, p_serial, p_refresh, p_retry, p_expire, p_minttl, p_weight, p_port, p_tag)
    RETURNING id INTO record_id;
    
    RETURN record_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get DNS statistics
CREATE OR REPLACE FUNCTION get_dns_stats()
RETURNS TABLE(
    record_type VARCHAR(10),
    count BIGINT,
    avg_ttl NUMERIC(10,2),
    min_ttl INTEGER,
    max_ttl INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        dr.record_type,
        COUNT(*) as count,
        ROUND(AVG(dr.ttl), 2) as avg_ttl,
        MIN(dr.ttl) as min_ttl,
        MAX(dr.ttl) as max_ttl
    FROM dns_records dr
    GROUP BY dr.record_type
    ORDER BY count DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old records (for maintenance)
CREATE OR REPLACE FUNCTION cleanup_old_records(days_old INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM dns_records 
    WHERE created_at < NOW() - INTERVAL '1 day' * days_old;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

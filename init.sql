-- ========================================
-- AlertBot Database Initialization Script
-- ========================================
-- This script provides basic database setup.
-- Full schema migration is handled by the Go migration tool.

-- Create required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";    -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- Trigram matching for fuzzy search
CREATE EXTENSION IF NOT EXISTS "btree_gin";    -- GIN index support for btree types

-- Set database configuration for optimal performance
ALTER DATABASE alertbot SET shared_preload_libraries = 'pg_stat_statements';
ALTER DATABASE alertbot SET log_statement = 'mod';  -- Log DDL and DML statements
ALTER DATABASE alertbot SET log_min_duration_statement = 1000;  -- Log slow queries
ALTER DATABASE alertbot SET default_statistics_target = 1000;   -- Better query planning

-- Grant privileges to alertbot user
GRANT ALL PRIVILEGES ON DATABASE alertbot TO alertbot;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO alertbot;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO alertbot;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO alertbot;

-- Grant privileges on future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO alertbot;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO alertbot;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON FUNCTIONS TO alertbot;

-- Create schema version tracking (if not exists)
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    dirty BOOLEAN NOT NULL DEFAULT FALSE,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert initial schema version
INSERT INTO schema_migrations (version, dirty) 
VALUES ('000_initial', FALSE) 
ON CONFLICT (version) DO NOTHING;

-- Create basic monitoring views for database health
CREATE OR REPLACE VIEW db_health_check AS
SELECT 
    'AlertBot Database' as service,
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'alerts') 
        THEN 'healthy' 
        ELSE 'unhealthy' 
    END as status,
    NOW() as checked_at,
    current_setting('server_version') as version;

-- Log initialization completion
INSERT INTO schema_migrations (version, dirty) 
VALUES ('001_extensions_and_permissions', FALSE) 
ON CONFLICT (version) DO NOTHING;
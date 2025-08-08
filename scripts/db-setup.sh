#!/bin/bash

# =============================================
# AlertBot Database Setup Script
# =============================================
# This script provides database setup options for different deployment scenarios

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-alertbot}"
POSTGRES_USER="${POSTGRES_USER:-alertbot}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-password}"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Function to check if PostgreSQL is running
check_postgres() {
    print_status "Checking PostgreSQL connection..."
    if ! pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" > /dev/null 2>&1; then
        print_error "Cannot connect to PostgreSQL at $POSTGRES_HOST:$POSTGRES_PORT"
        print_error "Please ensure PostgreSQL is running and credentials are correct"
        exit 1
    fi
    print_status "PostgreSQL connection successful"
}

# Function to run SQL file
run_sql_file() {
    local sql_file="$1"
    local description="$2"
    
    if [[ ! -f "$sql_file" ]]; then
        print_error "SQL file not found: $sql_file"
        return 1
    fi
    
    print_status "Running $description..."
    PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        -v ON_ERROR_STOP=1 \
        -f "$sql_file"
    
    print_status "$description completed successfully"
}

# Function to run Go migration tool
run_go_migration() {
    print_status "Running Go migration tool..."
    cd "$PROJECT_ROOT"
    
    if [[ ! -f "go.mod" ]]; then
        print_error "go.mod not found. Please run this script from the project root."
        exit 1
    fi
    
    # Set environment variables for Go migration
    export DATABASE_HOST="$POSTGRES_HOST"
    export DATABASE_PORT="$POSTGRES_PORT"
    export DATABASE_USER="$POSTGRES_USER"
    export DATABASE_PASSWORD="$POSTGRES_PASSWORD"
    export DATABASE_NAME="$POSTGRES_DB"
    
    if go run cmd/migrate/main.go; then
        print_status "Go migration completed successfully"
    else
        print_error "Go migration failed"
        return 1
    fi
}

# Function to create database backup
create_backup() {
    local backup_file="${1:-alertbot_backup_$(date +%Y%m%d_%H%M%S).sql}"
    print_status "Creating database backup: $backup_file"
    
    PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        --no-password \
        --verbose \
        --clean \
        --if-exists \
        --create \
        > "$backup_file"
    
    print_status "Backup created successfully: $backup_file"
}

# Function to restore database from backup
restore_backup() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        print_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    print_warning "This will completely replace the current database!"
    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Restore cancelled"
        exit 0
    fi
    
    print_status "Restoring database from backup: $backup_file"
    PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "postgres" \
        -v ON_ERROR_STOP=1 \
        -f "$backup_file"
    
    print_status "Database restored successfully"
}

# Function to initialize fresh database
init_fresh() {
    print_header "Fresh Database Initialization"
    check_postgres
    
    # Run basic initialization
    run_sql_file "$PROJECT_ROOT/init.sql" "basic database initialization"
    
    # Run Go migration for full schema
    run_go_migration
    
    print_status "Fresh database initialization completed!"
}

# Function to initialize from SQL schema
init_from_sql() {
    print_header "SQL Schema Initialization"
    check_postgres
    
    # Run basic initialization
    run_sql_file "$PROJECT_ROOT/init.sql" "basic database initialization"
    
    # Run complete schema migration
    run_sql_file "$PROJECT_ROOT/migrations/001_complete_schema.sql" "complete schema initialization"
    
    print_status "SQL schema initialization completed!"
}

# Function to update existing database
update_existing() {
    print_header "Database Update"
    check_postgres
    
    # Create backup before updating
    create_backup "backup_before_update_$(date +%Y%m%d_%H%M%S).sql"
    
    # Run Go migration to apply updates
    run_go_migration
    
    print_status "Database update completed!"
}

# Function to show database status
show_status() {
    print_header "Database Status"
    check_postgres
    
    print_status "Database connection: OK"
    print_status "Host: $POSTGRES_HOST:$POSTGRES_PORT"
    print_status "Database: $POSTGRES_DB"
    print_status "User: $POSTGRES_USER"
    
    # Check if tables exist
    table_count=$(PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';" 2>/dev/null || echo "0")
    
    print_status "Tables found: $table_count"
    
    if [[ "$table_count" -gt 0 ]]; then
        print_status "Schema appears to be initialized"
        
        # Check for alerts table specifically
        if PGPASSWORD="$POSTGRES_PASSWORD" psql \
            -h "$POSTGRES_HOST" \
            -p "$POSTGRES_PORT" \
            -U "$POSTGRES_USER" \
            -d "$POSTGRES_DB" \
            -t -c "SELECT 1 FROM information_schema.tables WHERE table_name = 'alerts';" 2>/dev/null | grep -q 1; then
            
            alert_count=$(PGPASSWORD="$POSTGRES_PASSWORD" psql \
                -h "$POSTGRES_HOST" \
                -p "$POSTGRES_PORT" \
                -U "$POSTGRES_USER" \
                -d "$POSTGRES_DB" \
                -t -c "SELECT COUNT(*) FROM alerts;" 2>/dev/null || echo "0")
            print_status "Alerts in database: $alert_count"
        fi
    else
        print_warning "Schema appears to be uninitialized"
    fi
}

# Function to show help
show_help() {
    echo "AlertBot Database Setup Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  init-fresh     Initialize a fresh database using Go migration"
    echo "  init-sql       Initialize database using SQL schema files"
    echo "  update         Update existing database with new migrations"
    echo "  backup [file]  Create database backup (optional filename)"
    echo "  restore <file> Restore database from backup file"
    echo "  status         Show database status"
    echo "  help           Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  POSTGRES_HOST     PostgreSQL host (default: localhost)"
    echo "  POSTGRES_PORT     PostgreSQL port (default: 5432)"
    echo "  POSTGRES_DB       Database name (default: alertbot)"
    echo "  POSTGRES_USER     Database user (default: alertbot)"
    echo "  POSTGRES_PASSWORD Database password (default: password)"
    echo ""
    echo "Examples:"
    echo "  $0 init-fresh                    # Initialize fresh database"
    echo "  $0 backup alertbot_backup.sql    # Create backup"
    echo "  $0 restore alertbot_backup.sql   # Restore from backup"
    echo "  $0 status                        # Check database status"
}

# Main execution
case "${1:-help}" in
    "init-fresh")
        init_fresh
        ;;
    "init-sql")
        init_from_sql
        ;;
    "update")
        update_existing
        ;;
    "backup")
        create_backup "$2"
        ;;
    "restore")
        if [[ -z "$2" ]]; then
            print_error "Backup file required for restore command"
            echo "Usage: $0 restore <backup_file>"
            exit 1
        fi
        restore_backup "$2"
        ;;
    "status")
        show_status
        ;;
    "help"|*)
        show_help
        ;;
esac
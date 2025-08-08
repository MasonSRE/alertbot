#!/bin/bash

# =============================================
# AlertBot Startup Script with Database Migration
# =============================================
# This script ensures database is ready before starting the application

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MAX_RETRIES=30
RETRY_INTERVAL=2

# Database configuration
DATABASE_HOST="${DATABASE_HOST:-localhost}"
DATABASE_PORT="${DATABASE_PORT:-5432}"
DATABASE_USER="${DATABASE_USER:-alertbot}"
DATABASE_PASSWORD="${DATABASE_PASSWORD:-password}"
DATABASE_NAME="${DATABASE_NAME:-alertbot}"

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

# Function to wait for PostgreSQL to be ready
wait_for_postgres() {
    print_status "Waiting for PostgreSQL to be ready..."
    
    local retries=0
    while [[ $retries -lt $MAX_RETRIES ]]; do
        if pg_isready -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" > /dev/null 2>&1; then
            print_status "PostgreSQL is ready!"
            return 0
        fi
        
        retries=$((retries + 1))
        print_status "PostgreSQL not ready yet... ($retries/$MAX_RETRIES)"
        sleep $RETRY_INTERVAL
    done
    
    print_error "PostgreSQL failed to become ready after $MAX_RETRIES attempts"
    return 1
}

# Function to check if database exists
check_database_exists() {
    PGPASSWORD="$DATABASE_PASSWORD" psql \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -lqt | cut -d \| -f 1 | grep -qw "$DATABASE_NAME"
}

# Function to create database if it doesn't exist
create_database_if_needed() {
    if ! check_database_exists; then
        print_status "Database '$DATABASE_NAME' does not exist, creating..."
        PGPASSWORD="$DATABASE_PASSWORD" createdb \
            -h "$DATABASE_HOST" \
            -p "$DATABASE_PORT" \
            -U "$DATABASE_USER" \
            "$DATABASE_NAME"
        print_status "Database '$DATABASE_NAME' created successfully"
    else
        print_status "Database '$DATABASE_NAME' already exists"
    fi
}

# Function to check if tables exist
check_tables_exist() {
    local table_count
    table_count=$(PGPASSWORD="$DATABASE_PASSWORD" psql \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';" 2>/dev/null | tr -d ' ')
    
    [[ "$table_count" -gt 0 ]]
}

# Function to run database initialization
run_database_init() {
    print_header "Database Initialization"
    
    # Run basic initialization first
    if [[ -f "$PROJECT_ROOT/init.sql" ]]; then
        print_status "Running basic database initialization..."
        PGPASSWORD="$DATABASE_PASSWORD" psql \
            -h "$DATABASE_HOST" \
            -p "$DATABASE_PORT" \
            -U "$DATABASE_USER" \
            -d "$DATABASE_NAME" \
            -v ON_ERROR_STOP=1 \
            -f "$PROJECT_ROOT/init.sql"
        print_status "Basic initialization completed"
    fi
    
    # Run Go migration tool
    print_status "Running database migration..."
    cd "$PROJECT_ROOT"
    
    # Export environment variables for Go migration
    export DATABASE_HOST="$DATABASE_HOST"
    export DATABASE_PORT="$DATABASE_PORT"
    export DATABASE_USER="$DATABASE_USER"
    export DATABASE_PASSWORD="$DATABASE_PASSWORD"
    export DATABASE_NAME="$DATABASE_NAME"
    
    if go run cmd/migrate/main.go; then
        print_status "Database migration completed successfully"
    else
        print_error "Database migration failed"
        return 1
    fi
}

# Function to perform health check
health_check() {
    print_status "Performing database health check..."
    
    # Check basic connectivity
    if ! PGPASSWORD="$DATABASE_PASSWORD" psql \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        -c "SELECT 1;" > /dev/null 2>&1; then
        print_error "Database health check failed"
        return 1
    fi
    
    # Check if critical tables exist
    local critical_tables=("alerts" "routing_rules" "notification_channels")
    for table in "${critical_tables[@]}"; do
        if ! PGPASSWORD="$DATABASE_PASSWORD" psql \
            -h "$DATABASE_HOST" \
            -p "$DATABASE_PORT" \
            -U "$DATABASE_USER" \
            -d "$DATABASE_NAME" \
            -c "SELECT 1 FROM $table LIMIT 1;" > /dev/null 2>&1; then
            print_error "Critical table '$table' not found or not accessible"
            return 1
        fi
    done
    
    print_status "Database health check passed"
    return 0
}

# Function to start the application
start_application() {
    print_header "Starting AlertBot Application"
    
    cd "$PROJECT_ROOT"
    
    # Check if binary exists
    if [[ -f "./alertbot" ]]; then
        print_status "Starting AlertBot from binary..."
        exec ./alertbot
    elif [[ -f "cmd/server/main.go" ]]; then
        print_status "Starting AlertBot from source..."
        exec go run cmd/server/main.go
    else
        print_error "Neither binary nor source files found"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "AlertBot Startup Script with Database Migration"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --skip-migration    Skip database migration (use existing schema)"
    echo "  --force-migration   Force migration even if tables exist"
    echo "  --health-check-only Only perform health check and exit"
    echo "  --help              Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  DATABASE_HOST       Database host (default: localhost)"
    echo "  DATABASE_PORT       Database port (default: 5432)"
    echo "  DATABASE_USER       Database user (default: alertbot)"
    echo "  DATABASE_PASSWORD   Database password (default: password)"
    echo "  DATABASE_NAME       Database name (default: alertbot)"
}

# Parse command line arguments
SKIP_MIGRATION=false
FORCE_MIGRATION=false
HEALTH_CHECK_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-migration)
            SKIP_MIGRATION=true
            shift
            ;;
        --force-migration)
            FORCE_MIGRATION=true
            shift
            ;;
        --health-check-only)
            HEALTH_CHECK_ONLY=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution flow
main() {
    print_header "AlertBot Startup with Database Migration"
    
    # Wait for PostgreSQL
    if ! wait_for_postgres; then
        exit 1
    fi
    
    # Create database if needed
    create_database_if_needed
    
    # Handle database migration
    if [[ "$HEALTH_CHECK_ONLY" == "true" ]]; then
        health_check
        exit $?
    elif [[ "$SKIP_MIGRATION" == "true" ]]; then
        print_status "Skipping database migration as requested"
    elif [[ "$FORCE_MIGRATION" == "true" ]] || ! check_tables_exist; then
        print_status "Running database migration..."
        if ! run_database_init; then
            print_error "Database initialization failed"
            exit 1
        fi
    else
        print_status "Database tables already exist, skipping migration"
        print_status "Use --force-migration to force migration"
    fi
    
    # Perform health check
    if ! health_check; then
        print_error "Database health check failed, aborting startup"
        exit 1
    fi
    
    # Start the application
    start_application
}

# Execute main function
main "$@"
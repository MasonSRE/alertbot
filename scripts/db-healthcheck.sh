#!/bin/bash

# =============================================
# AlertBot Database Health Check Script
# =============================================
# This script performs comprehensive database health checks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
DATABASE_HOST="${DATABASE_HOST:-localhost}"
DATABASE_PORT="${DATABASE_PORT:-5432}"
DATABASE_USER="${DATABASE_USER:-alertbot}"
DATABASE_PASSWORD="${DATABASE_PASSWORD:-password}"
DATABASE_NAME="${DATABASE_NAME:-alertbot}"

# Health check parameters
MAX_CONNECTION_TIME=5
SLOW_QUERY_THRESHOLD=1000  # milliseconds
MAX_DEAD_TUPLE_RATIO=20    # percentage

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[i]${NC} $1"
}

# Function to run SQL query and return result
run_query() {
    local query="$1"
    PGPASSWORD="$DATABASE_PASSWORD" psql \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        -t -c "$query" 2>/dev/null | tr -d ' '
}

# Function to check basic connectivity
check_connectivity() {
    print_info "Checking database connectivity..."
    
    local start_time=$(date +%s%N)
    if pg_isready -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" > /dev/null 2>&1; then
        local end_time=$(date +%s%N)
        local duration_ms=$(( (end_time - start_time) / 1000000 ))
        
        if [[ $duration_ms -lt $(( MAX_CONNECTION_TIME * 1000 )) ]]; then
            print_status "Database connectivity OK (${duration_ms}ms)"
            return 0
        else
            print_warning "Database connectivity slow (${duration_ms}ms)"
            return 1
        fi
    else
        print_error "Database connectivity failed"
        return 1
    fi
}

# Function to check database version and settings
check_database_info() {
    print_info "Checking database information..."
    
    local version=$(run_query "SELECT version();")
    local db_size=$(run_query "SELECT pg_size_pretty(pg_database_size('$DATABASE_NAME'));")
    local max_connections=$(run_query "SHOW max_connections;")
    local current_connections=$(run_query "SELECT count(*) FROM pg_stat_activity WHERE datname = '$DATABASE_NAME';")
    
    print_status "PostgreSQL Version: ${version:0:50}..."
    print_status "Database Size: $db_size"
    print_status "Connections: $current_connections/$max_connections"
    
    # Check connection usage
    local connection_ratio=$(( current_connections * 100 / max_connections ))
    if [[ $connection_ratio -gt 80 ]]; then
        print_warning "High connection usage: ${connection_ratio}%"
    fi
}

# Function to check table existence and basic stats
check_tables() {
    print_info "Checking table status..."
    
    local critical_tables=("alerts" "routing_rules" "notification_channels" "silences" "alert_history")
    local all_tables_ok=true
    
    for table in "${critical_tables[@]}"; do
        if run_query "SELECT 1 FROM information_schema.tables WHERE table_name = '$table';" | grep -q 1; then
            local row_count=$(run_query "SELECT count(*) FROM $table;")
            print_status "Table '$table': $row_count rows"
        else
            print_error "Critical table '$table' not found"
            all_tables_ok=false
        fi
    done
    
    return $all_tables_ok
}

# Function to check index usage
check_indexes() {
    print_info "Checking index performance..."
    
    local unused_indexes=$(run_query "
        SELECT count(*) 
        FROM pg_stat_user_indexes 
        WHERE idx_scan = 0 
        AND schemaname = 'public'
        AND indexrelname NOT LIKE '%_pkey';")
    
    local low_hit_ratio_indexes=$(run_query "
        SELECT count(*)
        FROM pg_stat_user_indexes pgsui
        JOIN pg_statio_user_indexes pgsiui ON pgsui.indexrelid = pgsiui.indexrelid
        WHERE schemaname = 'public'
        AND (idx_blks_hit::float / NULLIF(idx_blks_hit + idx_blks_read, 0)) < 0.95;")
    
    if [[ "$unused_indexes" -gt 0 ]]; then
        print_warning "$unused_indexes unused indexes found"
    else
        print_status "All indexes are being used"
    fi
    
    if [[ "$low_hit_ratio_indexes" -gt 0 ]]; then
        print_warning "$low_hit_ratio_indexes indexes with low hit ratio"
    else
        print_status "Index hit ratios are good"
    fi
}

# Function to check table bloat
check_table_bloat() {
    print_info "Checking table bloat..."
    
    local tables_with_bloat=$(run_query "
        SELECT count(*)
        FROM pg_stat_user_tables
        WHERE schemaname = 'public'
        AND (n_dead_tup::float / NULLIF(n_live_tup + n_dead_tup, 0) * 100) > $MAX_DEAD_TUPLE_RATIO;")
    
    if [[ "$tables_with_bloat" -gt 0 ]]; then
        print_warning "$tables_with_bloat tables with high dead tuple ratio (>$MAX_DEAD_TUPLE_RATIO%)"
        print_info "Consider running VACUUM ANALYZE"
    else
        print_status "Table bloat levels are acceptable"
    fi
    
    # Check last vacuum/analyze times
    local tables_need_vacuum=$(run_query "
        SELECT count(*)
        FROM pg_stat_user_tables
        WHERE schemaname = 'public'
        AND (last_vacuum IS NULL OR last_vacuum < NOW() - INTERVAL '7 days')
        AND (last_autovacuum IS NULL OR last_autovacuum < NOW() - INTERVAL '7 days');")
    
    if [[ "$tables_need_vacuum" -gt 0 ]]; then
        print_warning "$tables_need_vacuum tables haven't been vacuumed recently"
    fi
}

# Function to check slow queries
check_slow_queries() {
    print_info "Checking for slow queries..."
    
    # This requires pg_stat_statements extension
    if run_query "SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements';" | grep -q 1; then
        local slow_queries=$(run_query "
            SELECT count(*)
            FROM pg_stat_statements
            WHERE mean_time > $SLOW_QUERY_THRESHOLD;")
        
        if [[ "$slow_queries" -gt 0 ]]; then
            print_warning "$slow_queries slow queries detected (>${SLOW_QUERY_THRESHOLD}ms average)"
        else
            print_status "No slow queries detected"
        fi
    else
        print_info "pg_stat_statements extension not available for slow query analysis"
    fi
}

# Function to check AlertBot-specific metrics
check_alertbot_metrics() {
    print_info "Checking AlertBot-specific metrics..."
    
    # Check recent alert activity
    local alerts_last_hour=$(run_query "
        SELECT count(*) 
        FROM alerts 
        WHERE created_at > NOW() - INTERVAL '1 hour';")
    
    local firing_alerts=$(run_query "
        SELECT count(*) 
        FROM alerts 
        WHERE status = 'firing';")
    
    local total_alerts=$(run_query "SELECT count(*) FROM alerts;")
    
    print_status "Total alerts: $total_alerts"
    print_status "Firing alerts: $firing_alerts"
    print_status "Alerts in last hour: $alerts_last_hour"
    
    # Check notification channels
    local active_channels=$(run_query "
        SELECT count(*) 
        FROM notification_channels 
        WHERE enabled = true;")
    
    print_status "Active notification channels: $active_channels"
    
    # Check routing rules
    local active_rules=$(run_query "
        SELECT count(*) 
        FROM routing_rules 
        WHERE enabled = true;")
    
    print_status "Active routing rules: $active_rules"
    
    # Warn if no firing alerts and system seems inactive
    if [[ "$firing_alerts" -eq 0 ]] && [[ "$alerts_last_hour" -eq 0 ]]; then
        print_warning "No recent alert activity - system may be inactive"
    fi
    
    # Warn if no active channels
    if [[ "$active_channels" -eq 0 ]]; then
        print_warning "No active notification channels configured"
    fi
}

# Function to perform full health check
full_health_check() {
    local overall_status=0
    
    echo "========================================="
    echo "AlertBot Database Health Check"
    echo "Host: $DATABASE_HOST:$DATABASE_PORT"
    echo "Database: $DATABASE_NAME"
    echo "Time: $(date)"
    echo "========================================="
    
    # Run all checks
    check_connectivity || overall_status=1
    echo
    
    check_database_info
    echo
    
    check_tables || overall_status=1
    echo
    
    check_indexes
    echo
    
    check_table_bloat
    echo
    
    check_slow_queries
    echo
    
    check_alertbot_metrics
    echo
    
    # Overall result
    echo "========================================="
    if [[ $overall_status -eq 0 ]]; then
        print_status "Overall health check: PASSED"
        echo "Database is healthy and ready for AlertBot operations"
    else
        print_error "Overall health check: FAILED"
        echo "Database has issues that need attention"
    fi
    echo "========================================="
    
    return $overall_status
}

# Function to perform simple readiness check (for containers)
readiness_check() {
    if check_connectivity && check_tables; then
        echo "ready"
        exit 0
    else
        echo "not ready"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "AlertBot Database Health Check Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  full      Perform full health check (default)"
    echo "  ready     Simple readiness check (for containers)"
    echo "  help      Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  DATABASE_HOST       Database host (default: localhost)"
    echo "  DATABASE_PORT       Database port (default: 5432)"
    echo "  DATABASE_USER       Database user (default: alertbot)"
    echo "  DATABASE_PASSWORD   Database password (default: password)"
    echo "  DATABASE_NAME       Database name (default: alertbot)"
}

# Main execution
case "${1:-full}" in
    "full")
        full_health_check
        ;;
    "ready")
        readiness_check
        ;;
    "help"|*)
        show_usage
        ;;
esac
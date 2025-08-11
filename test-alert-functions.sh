#!/bin/bash

# AlertBot Alert Management Functions Test Script
# This script tests all alert management functions to ensure they work correctly

API_BASE="http://localhost:8080/api/v1"

echo "=========================================="
echo "AlertBot Alert Management Functions Test"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
    fi
}

# Get some alert fingerprints for testing
echo "1. Getting existing alerts..."
ALERTS=$(curl -s "$API_BASE/alerts?page=1&size=5" | jq -r '.data.items[] | .fingerprint' 2>/dev/null)
ALERT1=$(echo "$ALERTS" | head -n 1)
ALERT2=$(echo "$ALERTS" | head -n 2 | tail -n 1)

if [ -z "$ALERT1" ]; then
    echo "No alerts found. Creating test alerts..."
    # Create test alerts
    curl -s -X POST "$API_BASE/alerts" \
        -H "Content-Type: application/json" \
        -d '[{
            "labels": {"alertname": "TestAlert1", "severity": "warning"},
            "annotations": {"summary": "Test alert for function testing"},
            "startsAt": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
        }]' > /dev/null
    
    sleep 1
    ALERTS=$(curl -s "$API_BASE/alerts?page=1&size=5" | jq -r '.data.items[] | .fingerprint' 2>/dev/null)
    ALERT1=$(echo "$ALERTS" | head -n 1)
fi

echo "Using alert fingerprints: $ALERT1"
echo ""

# Test individual alert operations
echo "2. Testing Individual Alert Operations"
echo "---------------------------------------"

# Test Silence
echo -n "Testing silence function... "
RESPONSE=$(curl -s -X PUT "$API_BASE/alerts/$ALERT1/silence" \
    -H "Content-Type: application/json" \
    -d '{"duration": "1h", "comment": "Test silence"}')
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
[ "$SUCCESS" = "true" ] && print_result 0 "Silence" || print_result 1 "Silence"

# Test Acknowledge
echo -n "Testing acknowledge function... "
RESPONSE=$(curl -s -X PUT "$API_BASE/alerts/$ALERT1/ack" \
    -H "Content-Type: application/json" \
    -d '{"comment": "Test acknowledge"}')
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
[ "$SUCCESS" = "true" ] && print_result 0 "Acknowledge" || print_result 1 "Acknowledge"

# Test History
echo -n "Testing history function... "
RESPONSE=$(curl -s "$API_BASE/alerts/$ALERT1/history")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
HISTORY_COUNT=$(echo "$RESPONSE" | jq '.data | length')
[ "$SUCCESS" = "true" ] && [ "$HISTORY_COUNT" -gt 0 ] && print_result 0 "History (found $HISTORY_COUNT entries)" || print_result 1 "History"

# Test Relations
echo -n "Testing relations function... "
RESPONSE=$(curl -s "$API_BASE/alerts/$ALERT1/relations")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
[ "$SUCCESS" = "true" ] && print_result 0 "Relations" || print_result 1 "Relations"

echo ""

# Test batch operations
echo "3. Testing Batch Alert Operations"
echo "----------------------------------"

# Get multiple fingerprints for batch testing
FINGERPRINTS=$(curl -s "$API_BASE/alerts?page=1&size=5" | jq -r '[.data.items[] | .fingerprint] | @json' 2>/dev/null)
if [ "$FINGERPRINTS" = "null" ] || [ "$FINGERPRINTS" = "[]" ]; then
    echo "Not enough alerts for batch testing. Creating more test alerts..."
    for i in {1..3}; do
        curl -s -X POST "$API_BASE/alerts" \
            -H "Content-Type: application/json" \
            -d '[{
                "labels": {"alertname": "TestBatchAlert'$i'", "severity": "info"},
                "annotations": {"summary": "Test batch alert '$i'"},
                "startsAt": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
            }]' > /dev/null
    done
    sleep 1
    FINGERPRINTS=$(curl -s "$API_BASE/alerts?page=1&size=5" | jq -r '[.data.items[] | .fingerprint] | @json' 2>/dev/null)
fi

# Test Batch Silence
echo -n "Testing batch silence function... "
RESPONSE=$(curl -s -X PUT "$API_BASE/alerts/batch/silence" \
    -H "Content-Type: application/json" \
    -d "{\"fingerprints\": $FINGERPRINTS, \"duration\": \"2h\", \"comment\": \"Batch silence test\"}")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
PROCESSED=$(echo "$RESPONSE" | jq -r '.data.processed')
[ "$SUCCESS" = "true" ] && print_result 0 "Batch Silence (processed $PROCESSED alerts)" || print_result 1 "Batch Silence"

# Test Batch Acknowledge
echo -n "Testing batch acknowledge function... "
RESPONSE=$(curl -s -X PUT "$API_BASE/alerts/batch/ack" \
    -H "Content-Type: application/json" \
    -d "{\"fingerprints\": $FINGERPRINTS, \"comment\": \"Batch acknowledge test\"}")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
PROCESSED=$(echo "$RESPONSE" | jq -r '.data.processed')
[ "$SUCCESS" = "true" ] && print_result 0 "Batch Acknowledge (processed $PROCESSED alerts)" || print_result 1 "Batch Acknowledge"

# Test Batch Resolve
echo -n "Testing batch resolve function... "
# Get different fingerprints for resolve test (to not affect other tests)
RESOLVE_FINGERPRINTS=$(curl -s "$API_BASE/alerts?page=2&size=3" | jq -r '[.data.items[] | .fingerprint] | @json' 2>/dev/null)
if [ "$RESOLVE_FINGERPRINTS" != "null" ] && [ "$RESOLVE_FINGERPRINTS" != "[]" ]; then
    RESPONSE=$(curl -s -X DELETE "$API_BASE/alerts/batch/resolve" \
        -H "Content-Type: application/json" \
        -d "{\"fingerprints\": $RESOLVE_FINGERPRINTS, \"comment\": \"Batch resolve test\"}")
    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    PROCESSED=$(echo "$RESPONSE" | jq -r '.data.processed')
    [ "$SUCCESS" = "true" ] && print_result 0 "Batch Resolve (processed $PROCESSED alerts)" || print_result 1 "Batch Resolve"
else
    print_result 0 "Batch Resolve (skipped - no alerts to resolve)"
fi

echo ""

# Test alert list with filters
echo "4. Testing Alert List with Filters"
echo "-----------------------------------"

# Test status filter
echo -n "Testing status filter... "
RESPONSE=$(curl -s "$API_BASE/alerts?status=firing&page=1&size=10")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
[ "$SUCCESS" = "true" ] && print_result 0 "Status filter" || print_result 1 "Status filter"

# Test severity filter
echo -n "Testing severity filter... "
RESPONSE=$(curl -s "$API_BASE/alerts?severity=warning&page=1&size=10")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
[ "$SUCCESS" = "true" ] && print_result 0 "Severity filter" || print_result 1 "Severity filter"

# Test pagination
echo -n "Testing pagination... "
RESPONSE=$(curl -s "$API_BASE/alerts?page=1&size=5")
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
PAGE=$(echo "$RESPONSE" | jq -r '.data.page')
SIZE=$(echo "$RESPONSE" | jq -r '.data.size')
[ "$SUCCESS" = "true" ] && [ "$PAGE" = "1" ] && [ "$SIZE" = "5" ] && print_result 0 "Pagination" || print_result 1 "Pagination"

echo ""

# Test resolve function (do this last as it changes alert state)
echo "5. Testing Resolve Function"
echo "----------------------------"

echo -n "Testing resolve function... "
if [ -n "$ALERT1" ]; then
    RESPONSE=$(curl -s -X DELETE "$API_BASE/alerts/$ALERT1" \
        -H "Content-Type: application/json")
    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    [ "$SUCCESS" = "true" ] && print_result 0 "Resolve" || print_result 1 "Resolve"
else
    print_result 1 "Resolve (no alert to test)"
fi

echo ""
echo "=========================================="
echo "Alert Management Functions Test Complete"
echo "=========================================="
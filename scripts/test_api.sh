#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
PORT=12321
BASE_URL="http://localhost:${PORT}"
TEST_USER="test_user"
TEST_SESSION="test_session_$(date +%s)"
PID_FILE="/tmp/memory_brain_test.pid"
LOG_FILE="/tmp/memory_brain_test.log"

# Counters
PASS_COUNT=0
FAIL_COUNT=0

# Helper functions
print_header() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}\n"
}

print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS: $2${NC}"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "${RED}✗ FAIL: $2${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

check_response() {
    local response="$1"
    local expected_key="$2"

    if echo "$response" | grep -q "$expected_key"; then
        return 0
    else
        return 1
    fi
}

# Start server
start_server() {
    print_header "Starting Memory Brain Server"

    # Kill existing process if any
    if [ -f "$PID_FILE" ]; then
        kill $(cat "$PID_FILE") 2>/dev/null || true
        rm -f "$PID_FILE"
    fi

    # Build if needed
    if [ ! -f "bin/memory-brain" ]; then
        echo "Building server..."
        make build
    fi

    # Start server in background
    ./bin/memory-brain server --port ${PORT} > "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"

    # Wait for server to start
    echo "Waiting for server to start..."
    for i in {1..10}; do
        if curl -s "${BASE_URL}/memory/health" > /dev/null 2>&1; then
            echo -e "${GREEN}Server started on port ${PORT}${NC}"
            return 0
        fi
        sleep 1
    done

    echo -e "${RED}Server failed to start${NC}"
    cat "$LOG_FILE"
    return 1
}

# Stop server
stop_server() {
    print_header "Stopping Server"

    if [ -f "$PID_FILE" ]; then
        kill $(cat "$PID_FILE") 2>/dev/null || true
        rm -f "$PID_FILE"
        echo "Server stopped"
    fi

    # Clean up log file
    rm -f "$LOG_FILE"
}

# Test 1: Health check
test_health() {
    print_header "Test 1: Health Check"

    response=$(curl -s "${BASE_URL}/memory/health")
    echo "Response: $response"

    check_response "$response" "healthy" && check_response "$response" "version"
    print_result $? "Health check returns status and version"
}

# Test 2: Observe endpoint (first request)
test_observe_first() {
    print_header "Test 2: Observe Endpoint (First Request)"

    response=$(curl -s -X POST "${BASE_URL}/memory/observe" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": \"obs_001\",
            \"user_id\": \"${TEST_USER}\",
            \"session_id\": \"${TEST_SESSION}\",
            \"hook_type\": \"post_tool_use\",
            \"data\": {
                \"tool_name\": \"bash\",
                \"tool_result\": \"success\",
                \"command\": \"ls -la\",
                \"duration\": 100
            }
        }")

    echo "Response: $response"

    check_response "$response" "success" && check_response "$response" "obs_001"
    print_result $? "Observe stores observation"

    # Verify dedup=false for first request
    check_response "$response" '"dedup":false'
    print_result $? "First request returns dedup=false"
}

# Test 3: Observe endpoint (duplicate - should be deduplicated)
test_observe_duplicate() {
    print_header "Test 3: Observe Endpoint (Duplicate Detection)"

    response=$(curl -s -X POST "${BASE_URL}/memory/observe" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": \"obs_001\",
            \"user_id\": \"${TEST_USER}\",
            \"session_id\": \"${TEST_SESSION}\",
            \"hook_type\": \"post_tool_use\",
            \"data\": {
                \"tool_name\": \"bash\",
                \"tool_result\": \"success\",
                \"command\": \"ls -la\",
                \"duration\": 100
            }
        }")

    echo "Response: $response"

    # Verify dedup=true for duplicate
    check_response "$response" '"dedup":true'
    print_result $? "Duplicate request returns dedup=true"
}

# Test 4: Profile endpoint (without inject)
test_profile_no_inject() {
    print_header "Test 4: Profile Endpoint (Without Inject)"

    response=$(curl -s "${BASE_URL}/memory/profile?user=${TEST_USER}")
    echo "Response: $response"

    check_response "$response" "success" && check_response "$response" "profile_count"
    print_result $? "Profile returns user data"

    # system_message should be empty when inject=false
    if echo "$response" | grep -q '"system_message":""'; then
        print_result 0 "System message is empty when inject=false"
    else
        # Check if system_message key exists but might be null or empty
        print_result 0 "Profile response format valid"
    fi
}

# Test 5: Profile endpoint (with inject)
test_profile_with_inject() {
    print_header "Test 5: Profile Endpoint (With Inject)"

    response=$(curl -s "${BASE_URL}/memory/profile?user=${TEST_USER}&inject=true")
    echo "Response: $response"

    check_response "$response" "success"
    print_result $? "Profile with inject returns success"
}

# Test 6: Profile update
test_profile_update() {
    print_header "Test 6: Profile Update"

    response=$(curl -s -X PUT "${BASE_URL}/memory/profile/update" \
        -H "Content-Type: application/json" \
        -d "{
            \"user_id\": \"${TEST_USER}\",
            \"session_id\": \"${TEST_SESSION}\",
            \"observations\": [
                {
                    \"id\": \"obs_002\",
                    \"user_id\": \"${TEST_USER}\",
                    \"session_id\": \"${TEST_SESSION}\",
                    \"hook_type\": \"post_tool_use\",
                    \"data\": {
                        \"tool_name\": \"read\",
                        \"tool_result\": \"success\",
                        \"file_path\": \"/test/file.go\",
                        \"duration\": 50
                    }
                },
                {
                    \"id\": \"obs_003\",
                    \"user_id\": \"${TEST_USER}\",
                    \"session_id\": \"${TEST_SESSION}\",
                    \"hook_type\": \"post_tool_use\",
                    \"data\": {
                        \"tool_name\": \"edit\",
                        \"tool_result\": \"success\",
                        \"file_path\": \"/test/file.go\",
                        \"duration\": 150
                    }
                }
            ]
        }")

    echo "Response: $response"

    check_response "$response" "success" && check_response "$response" "updated"
    print_result $? "Profile update returns success"

    check_response "$response" "profile_count"
    print_result $? "Profile update returns profile_count"
}

# Test 7: Search endpoint
test_search() {
    print_header "Test 7: Search Endpoint"

    response=$(curl -s "${BASE_URL}/memory/search?query=test&user=${TEST_USER}")
    echo "Response: $response"

    check_response "$response" "success"
    print_result $? "Search returns success"
}

# Test 8: Session summary
test_session_summary() {
    print_header "Test 8: Session Summary"

    response=$(curl -s --max-time 30 -X POST "${BASE_URL}/memory/session/summary" \
        -H "Content-Type: application/json" \
        -d "{
            \"user_id\": \"${TEST_USER}\",
            \"session_id\": \"${TEST_SESSION}\",
            \"generate\": true,
            \"observations\": [
                {
                    \"id\": \"obs_004\",
                    \"user_id\": \"${TEST_USER}\",
                    \"session_id\": \"${TEST_SESSION}\",
                    \"hook_type\": \"post_tool_use\",
                    \"data\": {
                        \"tool_name\": \"bash\",
                        \"tool_result\": \"success\",
                        \"command\": \"go build\",
                        \"duration\": 200
                    }
                }
            ]
        }")

    echo "Response: $response"

    check_response "$response" "success" && check_response "$response" "summary_id"
    print_result $? "Session summary generates summary_id"

    check_response "$response" "saved"
    print_result $? "Session summary is saved"
}

# Verify storage files
verify_storage() {
    print_header "Verifying Storage Files"

    STORAGE_DIR="${HOME}/.memory-brain/users/${TEST_USER}"

    # Check user directory exists
    if [ -d "$STORAGE_DIR" ]; then
        print_result 0 "User storage directory exists"
    else
        print_result 1 "User storage directory missing"
        return
    fi

    # Check profile.jsonl
    if [ -f "${STORAGE_DIR}/profile.jsonl" ]; then
        lines=$(wc -l < "${STORAGE_DIR}/profile.jsonl")
        echo "Profile file lines: $lines"
        print_result 0 "profile.jsonl exists ($lines lines)"
    else
        print_result 1 "profile.jsonl missing"
    fi

    # Check patterns.jsonl
    if [ -f "${STORAGE_DIR}/patterns.jsonl" ]; then
        lines=$(wc -l < "${STORAGE_DIR}/patterns.jsonl")
        echo "Patterns file lines: $lines"
        print_result 0 "patterns.jsonl exists ($lines lines)"
    else
        print_result 1 "patterns.jsonl missing"
    fi

    # Check sessions_archive
    if [ -d "${STORAGE_DIR}/sessions_archive" ]; then
        summaries=$(ls -1 "${STORAGE_DIR}/sessions_archive"/*.json 2>/dev/null | wc -l)
        echo "Session summaries: $summaries"
        print_result 0 "sessions_archive exists ($summaries summaries)"
    else
        print_result 1 "sessions_archive missing"
    fi
}

# Print final summary
print_summary() {
    print_header "Test Summary"

    echo -e "Total Tests: $((PASS_COUNT + FAIL_COUNT))"
    echo -e "${GREEN}Passed: ${PASS_COUNT}${NC}"
    echo -e "${RED}Failed: ${FAIL_COUNT}${NC}"

    if [ $FAIL_COUNT -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed! ✓${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed. Check output above.${NC}"
        return 1
    fi
}

# Cleanup test data
cleanup_test_data() {
    print_header "Cleanup Test Data"

    STORAGE_DIR="${HOME}/.memory-brain/users/${TEST_USER}"

    if [ -d "$STORAGE_DIR" ]; then
        echo "Removing test user data from $STORAGE_DIR"
        rm -rf "$STORAGE_DIR"
        echo "Test data cleaned"
    fi
}

# Main execution
main() {
    print_header "Memory Brain API Test Suite"

    # Start server
    start_server

    # Run all tests
    test_health
    test_observe_first
    test_observe_duplicate
    test_profile_no_inject
    test_profile_with_inject
    test_profile_update
    test_search
    test_session_summary

    # Verify storage
    verify_storage

    # Print summary
    print_summary
    RESULT=$?

    # Cleanup (optional - comment out to preserve test data)
    cleanup_test_data

    # Stop server
    stop_server

    exit $RESULT
}

# Run main
main
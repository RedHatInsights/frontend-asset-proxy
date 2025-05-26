#!/bin/bash

# Caddy Test Script

# --- Configuration ---
CADDY_BASE_URL="http://localhost:8080" # Adjust if your Caddy is on a different port

HEALTH_PATH="/healthz"
ROOT_PATH="/"
SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET="Caddy & Minio Test Page" # A snippet from your index.html <title>
SPA_DEEP_LINK_PATH="/some/deep/spa/link" # This should also serve the SPA entrypoint

# --- Helper Functions ---
# $1: Test Name
# $2: URL to test
# $3: Expected HTTP Status Code
# $4: (Optional) Expected Content-Type (substring match)
# $5: (Optional) Expected Content Snippet (grep match)
run_test() {
    local test_name="$1"
    local url="$2"
    local expected_status="$3"
    local expected_content_type="$4"
    local expected_content_snippet="$5"
    local success=true

    echo "----------------------------------------"
    echo "Running Test: $test_name"
    echo "   URL: $url"

    response=$(curl -s -L -w "\nHTTP_STATUS:%{http_code}\nCONTENT_TYPE:%{content_type}" -o response_body.tmp "$url")
    
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d':' -f2)
    content_type=$(echo "$response" | grep "CONTENT_TYPE:" | cut -d':' -f2 | awk '{$1=$1};1') # awk to trim whitespace
    body_content=$(cat response_body.tmp)
    rm -f response_body.tmp

    echo "   Received Status: $http_status"
    if [ "$http_status" -ne "$expected_status" ]; then
        echo "   FAILED: Expected status $expected_status, got $http_status"
        success=false
    else
        echo "   PASSED: HTTP Status $http_status"
    fi

    if [ -n "$expected_content_type" ]; then
        echo "   Received Content-Type: $content_type"
        if [[ "$content_type" != *"$expected_content_type"* ]]; then
            echo "   FAILED: Expected Content-Type to contain '$expected_content_type', got '$content_type'"
            success=false
        else
            echo "   PASSED: Content-Type matches"
        fi
    fi
    
    if [ -n "$expected_content_snippet" ]; then
        if ! echo "$body_content" | grep -qF "$expected_content_snippet"; then
            echo "   FAILED: Expected content to contain '$expected_content_snippet'"
            echo "      --- Received Body (first 100 chars) ---"
            echo "${body_content:0:100}..."
            echo "      -------------------------------------"
            success=false
        else
            echo "   PASSED: Content snippet found"
        fi
    fi
    
    if [ "$success" = true ]; then
        echo "   Test Result: SUCCESS"
    else
        echo "   Test Result: FAILURE"
    fi
    echo "----------------------------------------"
    echo ""
    return $([ "$success" = true ] && echo 0 || echo 1)
}

# --- Main Test Execution ---
echo "Starting Caddy Server Tests..."
echo "   Targeting: $CADDY_BASE_URL"
echo ""

all_tests_passed=true

# Test 1: Health Check
run_test "Health Check" "${CADDY_BASE_URL}${HEALTH_PATH}" 200 "text/plain" "OK"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 2: Root Path (SPA Entrypoint)
run_test "Root Path (SPA Entrypoint)" "${CADDY_BASE_URL}${ROOT_PATH}" 200 "text/html" "$SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 3: SPA Deep Link
run_test "SPA Deep Link" "${CADDY_BASE_URL}${SPA_DEEP_LINK_PATH}" 200 "text/html" "$SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET"
if [ $? -ne 0 ]; then all_tests_passed=false; fi


# --- Summary ---
echo "All Tests Completed."
if [ "$all_tests_passed" = true ]; then
    echo "All tests passed successfully!"
    exit 0
else
    echo "Some tests failed. Please review the output above."
    exit 1
fi

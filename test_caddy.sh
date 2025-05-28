#!/bin/bash

# Caddy Test Script (for Specific File & Simplified Proxy)

# --- Configuration ---
<<<<<<< HEAD
CADDY_BASE_URL="http://localhost:8080" # Adjust if your Caddy is on a different port

HEALTH_PATH="/healthz"

# Specific file to test, as requested by the user
SPECIFIC_FILE_PATH="/api/chrome-service/v1/static/stable/prod/navigation/edge-navigation.json"
SPECIFIC_FILE_EXPECTED_CONTENT_TYPE="application/json"
# Optional: Add a snippet from your JSON if you want to verify content, e.g.:
# SPECIFIC_FILE_EXPECTED_CONTENT_SNIPPET='"someKey": "someValue"' 

# Configuration for testing index.html directly
INDEX_HTML_PATH="/index.html" # Changed from "/" to "/index.html"
INDEX_HTML_EXPECTED_CONTENT_TYPE="text/html"
INDEX_HTML_EXPECTED_CONTENT_SNIPPET="Caddy & Minio Test Page" # A snippet from your index.html <title>

# --- Helper Functions ---
=======
# Set the base URL for your Caddy server
CADDY_BASE_URL="http://localhost:8080" # Adjust if your Caddy is on a different port

HEALTH_PATH="/healthz"
ROOT_PATH="/"
SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET="Caddy & Minio Test Page" # A snippet from your index.html <title>
SPA_DEEP_LINK_PATH="/some/deep/spa/link" # This should also serve the SPA entrypoint

# --- Helper Functions ---
# Function to make a curl request and check the status code
>>>>>>> 7b41624 (Add test script and README)
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

    response_headers_file=$(mktemp)
    response_body_file=$(mktemp)

    # Perform the curl request
    http_status=$(curl -s -L -w "%{http_code}" -o "$response_body_file" -D "$response_headers_file" "$url")
    
    # Extract Content-Type, removing charset and extra spaces
    content_type=$(grep -i "^Content-Type:" "$response_headers_file" | awk '{$1=$1};1' | cut -d' ' -f2- | sed 's/;.*//')
    body_content=$(cat "$response_body_file")
    
    rm -f "$response_headers_file" "$response_body_file"

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

# Test 2: Specific File Request
# This test assumes the file exists in Minio at the correct path:
# BUCKET_PATH_PREFIX + SPECIFIC_FILE_PATH
# e.g., /frontend-assets/api/chrome-service/v1/static/stable/prod/navigation/edge-navigation.json
run_test "Specific File Request" "${CADDY_BASE_URL}${SPECIFIC_FILE_PATH}" 200 "$SPECIFIC_FILE_EXPECTED_CONTENT_TYPE" # Add SPECIFIC_FILE_EXPECTED_CONTENT_SNIPPET if desired
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 3: Index.html (Direct Request)
# This test assumes index.html exists in Minio at the root of the bucket path prefix
run_test "Index.html (Direct Request)" "${CADDY_BASE_URL}${INDEX_HTML_PATH}" 200 "$INDEX_HTML_EXPECTED_CONTENT_TYPE" "$INDEX_HTML_EXPECTED_CONTENT_SNIPPET"
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

#!/bin/bash

# Proxy Test Script (for /apps and /manifest routes)

# --- Configuration ---
PROXY_BASE_URL="http://localhost:8080" # Adjust if your proxy is on a different port
MINIO_UPSTREAM_URL="http://localhost:9000"
HEALTH_PATH="/healthz"

# Configuration for testing /apps route (serves from /data/ prefix in S3)
APPS_TEST_PATH="/apps/my-app/index.html"
APPS_EXPECTED_CONTENT_TYPE="text/html"
APPS_EXPECTED_CONTENT_SNIPPET="<html>" # Generic HTML snippet to verify it's an HTML file

# Configuration for testing /manifests route (serves directly from bucket path prefix)
MANIFEST_TEST_PATH="/manifests/app-manifest.json"
MANIFEST_EXPECTED_CONTENT_TYPE="application/json"
MANIFEST_EXPECTED_CONTENT_SNIPPET="{" # Generic JSON snippet to verify it's a JSON file

# Alternative /apps test with a different file type
APPS_JS_TEST_PATH="/apps/my-app/main.js"
APPS_JS_EXPECTED_CONTENT_TYPE="application/javascript"

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
echo "Starting Proxy Server Tests for /apps and /manifest routes..."
echo "   Targeting: $PROXY_BASE_URL"
echo ""

all_tests_passed=true

# Test 1: Health Check
run_test "Health Check" "${PROXY_BASE_URL}${HEALTH_PATH}" 200 "text/plain" "OK"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 2: /apps route - HTML file
# This test checks the /apps route which serves files from BUCKET_PATH_PREFIX/data/
# e.g., /apps/my-app/index.html -> BUCKET_PATH_PREFIX/data/apps/my-app/index.html
run_test "Apps Route - HTML File" "${PROXY_BASE_URL}${APPS_TEST_PATH}" 200 "$APPS_EXPECTED_CONTENT_TYPE" "$APPS_EXPECTED_CONTENT_SNIPPET"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 3: /apps route - JavaScript file
# This test checks another file type through the /apps route
run_test "Apps Route - JavaScript File" "${PROXY_BASE_URL}${APPS_JS_TEST_PATH}" 200 "$APPS_JS_EXPECTED_CONTENT_TYPE"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 4: /manifest route
# This test checks the /manifest route which serves files directly from BUCKET_PATH_PREFIX
# e.g., /manifest/app-manifest.json -> BUCKET_PATH_PREFIX/manifest/app-manifest.json
run_test "Manifest Route - JSON File" "${PROXY_BASE_URL}${MANIFEST_TEST_PATH}" 200 "$MANIFEST_EXPECTED_CONTENT_TYPE" "$MANIFEST_EXPECTED_CONTENT_SNIPPET"
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

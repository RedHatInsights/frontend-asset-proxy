#!/bin/bash

# Caddy Test Script

# --- Configuration ---
# Set the base URL for your Caddy server
CADDY_BASE_URL="http://localhost:8080" # Adjust if your Caddy is on a different port

# Define paths to test
HEALTH_PATH="/healthz"
ROOT_PATH="/"
# Assuming you have an index.html at the root of your S3/Minio bucket
SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET="Caddy & Minio Test Page" # A snippet from your index.html <title>

# Assuming you have uploaded css/style.css to your Minio bucket
# If not, this test will likely fail or hit the SPA fallback.
# You can comment it out or change it to an asset you know exists.
ASSET_PATH="/css/style.css"
ASSET_EXPECTED_CONTENT_TYPE="text/css" # Or "application/octet-stream" if Minio doesn't set it

SPA_DEEP_LINK_PATH="/some/deep/spa/link" # This should also serve the SPA entrypoint

# --- Helper Functions ---
# Function to make a curl request and check the status code
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
    echo "🧪 Running Test: $test_name"
    echo "   URL: $url"

    # Perform the curl request, capturing headers and body
    # -s: silent
    # -L: follow redirects (though not expected for these tests)
    # -w "%{http_code}": output HTTP status code
    # -o /dev/null: discard body (we'll capture it separately if needed for content check)
    # -D -: dump headers to stdout
    response=$(curl -s -L -w "\nHTTP_STATUS:%{http_code}\nCONTENT_TYPE:%{content_type}" -o response_body.tmp "$url")
    
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d':' -f2)
    content_type=$(echo "$response" | grep "CONTENT_TYPE:" | cut -d':' -f2 | awk '{$1=$1};1') # awk to trim whitespace
    body_content=$(cat response_body.tmp)
    rm -f response_body.tmp

    echo "   Received Status: $http_status"
    if [ "$http_status" -ne "$expected_status" ]; then
        echo "   ❌ FAILED: Expected status $expected_status, got $http_status"
        success=false
    else
        echo "   ✅ PASSED: HTTP Status $http_status"
    fi

    if [ -n "$expected_content_type" ]; then
        echo "   Received Content-Type: $content_type"
        if [[ "$content_type" != *"$expected_content_type"* ]]; then
            echo "   ❌ FAILED: Expected Content-Type to contain '$expected_content_type', got '$content_type'"
            success=false
        else
            echo "   ✅ PASSED: Content-Type matches"
        fi
    fi
    
    if [ -n "$expected_content_snippet" ]; then
        if ! echo "$body_content" | grep -qF "$expected_content_snippet"; then
            echo "   ❌ FAILED: Expected content to contain '$expected_content_snippet'"
            echo "      --- Received Body (first 100 chars) ---"
            echo "${body_content:0:100}..."
            echo "      -------------------------------------"
            success=false
        else
            echo "   ✅ PASSED: Content snippet found"
        fi
    fi
    
    if [ "$success" = true ]; then
        echo "   🌟 Test Result: SUCCESS"
    else
        echo "   💔 Test Result: FAILURE"
    fi
    echo "----------------------------------------"
    echo ""
    return $([ "$success" = true ] && echo 0 || echo 1)
}

# --- Main Test Execution ---
echo "🚀 Starting Caddy Server Tests..."
echo "   Targeting: $CADDY_BASE_URL"
echo ""

all_tests_passed=true

# Test 1: Health Check
run_test "Health Check" "${CADDY_BASE_URL}${HEALTH_PATH}" 200 "text/plain" "OK"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 2: Root Path (SPA Entrypoint)
run_test "Root Path (SPA Entrypoint)" "${CADDY_BASE_URL}${ROOT_PATH}" 200 "text/html" "$SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET"
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 3: Specific Asset Request
# Note: This test assumes 'css/style.css' exists and Minio serves it with 'text/css'.
# If Minio serves it as 'application/octet-stream' or the file doesn't exist, this might fail
# or trigger the SPA fallback (resulting in text/html and status 200 if SPA fallback works).
# Adjust ASSET_PATH and ASSET_EXPECTED_CONTENT_TYPE as needed for an asset you KNOW exists.
echo "ℹ️  Note: The 'Specific Asset' test might require an actual asset like 'css/style.css' to be present in Minio."
echo "    If it's not, it might fall back to serving index.html (status 200, type text/html)."
run_test "Specific Asset Request" "${CADDY_BASE_URL}${ASSET_PATH}" 200 "$ASSET_EXPECTED_CONTENT_TYPE" # No specific content snippet for CSS
if [ $? -ne 0 ]; then all_tests_passed=false; fi

# Test 4: SPA Deep Link
run_test "SPA Deep Link" "${CADDY_BASE_URL}${SPA_DEEP_LINK_PATH}" 200 "text/html" "$SPA_ENTRYPOINT_EXPECTED_CONTENT_SNIPPET"
if [ $? -ne 0 ]; then all_tests_passed=false; fi


# --- Summary ---
echo "🏁 All Tests Completed."
if [ "$all_tests_passed" = true ]; then
    echo "🎉🎉🎉 All tests passed successfully! 🎉🎉🎉"
    exit 0
else
    echo "☠️☠️☠️ Some tests failed. Please review the output above. ☠️☠️☠️"
    exit 1
fi

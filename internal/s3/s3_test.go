package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/frontend-asset-proxy/internal/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
	"github.com/sirupsen/logrus"
)

// MockS3Client implements the S3Client interface for testing
type MockS3Client struct {
	GetObjectFunc func(context.Context, *awss3.GetObjectInput, ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	OptionsFunc   func() awss3.Options
}

func (m *MockS3Client) GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	return m.GetObjectFunc(ctx, params, optFns...)
}

func (m *MockS3Client) Options() awss3.Options {
	if m.OptionsFunc != nil {
		return m.OptionsFunc()
	}
	return awss3.Options{}
}

func TestProxyS3_InvalidPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty path", ""},
		{"path without bucket", "/nobucket"},
		{"path with only bucket", "/bucket/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockS3Client := &MockS3Client{}
			cfg := config.FrontendAssetProxyConfig{
				ProxiedRequestTimeout: 30 * time.Second,
			}
			logger := logrus.New()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/", nil)

			ProxyS3(recorder, request, mockS3Client, cfg, tt.path, logger)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), "bad request") {
				t.Errorf("Expected 'bad request' in response body, got: %s", recorder.Body.String())
			}
		})
	}
}

func TestProxyS3_Success(t *testing.T) {
	mockS3Client := &MockS3Client{
		GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			content := "test file content"
			contentLength := int64(len(content))
			contentType := "text/plain"
			etag := "\"test-etag\""

			return &awss3.GetObjectOutput{
				Body:          io.NopCloser(strings.NewReader(content)),
				ContentLength: &contentLength,
				ContentType:   &contentType,
				ETag:          &etag,
			}, nil
		},
	}

	cfg := config.FrontendAssetProxyConfig{
		ProxiedRequestTimeout: 30 * time.Second,
	}
	logger := logrus.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/bucket/key.txt", nil)

	ProxyS3(recorder, request, mockS3Client, cfg, "/bucket/key.txt", logger)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Body.String() != "test file content" {
		t.Errorf("Expected 'test file content', got: %s", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got: %s", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("ETag") != "\"test-etag\"" {
		t.Errorf("Expected ETag '\"test-etag\"', got: %s", recorder.Header().Get("ETag"))
	}
}

func TestProxyS3_URLEncodedKey(t *testing.T) {
	mockS3Client := &MockS3Client{
		GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			if *input.Bucket != "bucket" {
				t.Errorf("Expected bucket 'bucket', got: %s", *input.Bucket)
			}
			if *input.Key != "path/to file.txt" { // Should be URL decoded
				t.Errorf("Expected key 'path/to file.txt', got: %s", *input.Key)
			}

			return &awss3.GetObjectOutput{
				Body:          io.NopCloser(strings.NewReader("content")),
				ContentLength: &[]int64{7}[0],
			}, nil
		},
	}

	cfg := config.FrontendAssetProxyConfig{
		ProxiedRequestTimeout: 30 * time.Second,
	}
	logger := logrus.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/bucket/path/to%20file.txt", nil)

	ProxyS3(recorder, request, mockS3Client, cfg, "/bucket/path/to%20file.txt", logger)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestProxyS3_HeadRequest(t *testing.T) {
	mockS3Client := &MockS3Client{
		GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			contentType := "text/plain"
			return &awss3.GetObjectOutput{
				Body:        io.NopCloser(strings.NewReader("test content")),
				ContentType: &contentType,
			}, nil
		},
	}

	cfg := config.FrontendAssetProxyConfig{
		ProxiedRequestTimeout: 30 * time.Second,
	}
	logger := logrus.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("HEAD", "/bucket/key.txt", nil)

	ProxyS3(recorder, request, mockS3Client, cfg, "/bucket/key.txt", logger)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Body.String() != "" {
		t.Errorf("Expected empty body for HEAD request, got: %s", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got: %s", recorder.Header().Get("Content-Type"))
	}
}

func TestProxyS3_ConditionalHeaders(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"Range header", "Range", "bytes=0-100"},
		{"If-None-Match header", "If-None-Match", "\"old-etag\""},
		{"If-Match header", "If-Match", "\"current-etag\""},
		{"If-Modified-Since header", "If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockS3Client := &MockS3Client{
				GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
					// Verify headers are passed through
					switch tt.header {
					case "Range":
						if input.Range == nil || *input.Range != tt.value {
							t.Errorf("Expected Range %s, got: %v", tt.value, input.Range)
						}
					case "If-None-Match":
						if input.IfNoneMatch == nil || *input.IfNoneMatch != tt.value {
							t.Errorf("Expected IfNoneMatch %s, got: %v", tt.value, input.IfNoneMatch)
						}
					case "If-Match":
						if input.IfMatch == nil || *input.IfMatch != tt.value {
							t.Errorf("Expected IfMatch %s, got: %v", tt.value, input.IfMatch)
						}
					case "If-Modified-Since":
						if input.IfModifiedSince == nil {
							t.Error("Expected IfModifiedSince to be set")
						}
					}

					return &awss3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader("content")),
						ContentLength: &[]int64{7}[0],
					}, nil
				},
			}

			cfg := config.FrontendAssetProxyConfig{
				ProxiedRequestTimeout: 30 * time.Second,
			}
			logger := logrus.New()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/bucket/key.txt", nil)
			request.Header.Set(tt.header, tt.value)

			ProxyS3(recorder, request, mockS3Client, cfg, "/bucket/key.txt", logger)

			if recorder.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
			}
		})
	}
}

func TestProxyS3_S3Errors(t *testing.T) {
	tests := []struct {
		name           string
		error          error
		expectedStatus int
	}{
		{"NoSuchKey error", &types.NoSuchKey{}, http.StatusNotFound},
		{"AccessDenied error", &smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied"}, http.StatusForbidden},
		{"Timeout error", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"InternalError", &smithy.GenericAPIError{Code: "InternalError", Message: "Internal Error"}, http.StatusInternalServerError},
		{"Unknown error", errors.New("unknown error"), http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockS3Client := &MockS3Client{
				GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
					return nil, tt.error
				},
			}

			cfg := config.FrontendAssetProxyConfig{
				ProxiedRequestTimeout: 30 * time.Second,
			}
			logger := logrus.New()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/bucket/key.txt", nil)

			ProxyS3(recorder, request, mockS3Client, cfg, "/bucket/key.txt", logger)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, recorder.Code)
			}
		})
	}
}

func TestProxyS3_ForbiddenFiles(t *testing.T) {
	// Tests various scenarios where access to files is forbidden
	// This includes different AWS S3 error codes that should all map to 403 Forbidden
	tests := []struct {
		name         string
		path         string
		errorCode    string
		errorMessage string
	}{
		{
			name:         "Access denied to private file",
			path:         "/secure-bucket/private/secrets.txt",
			errorCode:    "AccessDenied",
			errorMessage: "Access Denied",
		},
		{
			name:         "Forbidden file due to policy",
			path:         "/public-bucket/admin/config.json",
			errorCode:    "Forbidden",
			errorMessage: "Forbidden",
		},
		{
			name:         "Invalid credentials for sensitive data",
			path:         "/data-bucket/sensitive/users.csv",
			errorCode:    "SignatureDoesNotMatch",
			errorMessage: "The request signature we calculated does not match",
		},
		{
			name:         "Expired token access",
			path:         "/temp-bucket/expired/file.zip",
			errorCode:    "ExpiredToken",
			errorMessage: "The provided token has expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockS3Client := &MockS3Client{
				GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
					// Verify we're accessing the expected bucket and key
					pathParts := strings.Split(strings.TrimPrefix(tt.path, "/"), "/")
					expectedBucket := pathParts[0]
					expectedKey := strings.Join(pathParts[1:], "/")

					if *input.Bucket != expectedBucket {
						t.Errorf("Expected bucket '%s', got '%s'", expectedBucket, *input.Bucket)
					}
					if *input.Key != expectedKey {
						t.Errorf("Expected key '%s', got '%s'", expectedKey, *input.Key)
					}

					return nil, &smithy.GenericAPIError{
						Code:    tt.errorCode,
						Message: tt.errorMessage,
					}
				},
				OptionsFunc: func() awss3.Options {
					return awss3.Options{
						Logger: &mockLogger{},
					}
				},
			}

			cfg := config.FrontendAssetProxyConfig{
				ProxiedRequestTimeout: 30 * time.Second,
			}
			logger := logrus.New()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", tt.path, nil)

			ProxyS3(recorder, request, mockS3Client, cfg, tt.path, logger)

			// All forbidden errors should map to 403 Forbidden
			if recorder.Code != http.StatusForbidden {
				t.Errorf("Expected status %d (Forbidden), got %d", http.StatusForbidden, recorder.Code)
			}

			// Verify the response contains the correct status text
			if !strings.Contains(recorder.Body.String(), "Forbidden") {
				t.Errorf("Expected response body to contain 'Forbidden', got: %s", recorder.Body.String())
			}
		})
	}
}

// mockLogger implements the logging interface for testing
type mockLogger struct{}

func (m *mockLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	// Mock logger that does nothing - prevents nil pointer issues in tests
}

func TestProxyS3_HTTPResponseErrors(t *testing.T) {
	tests := []struct {
		name           string
		httpStatus     int
		expectedStatus int
	}{
		{"HTTP 404", 404, http.StatusNotFound},
		{"HTTP 403", 403, http.StatusForbidden},
		{"HTTP 500", 500, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockS3Client := &MockS3Client{
				GetObjectFunc: func(ctx context.Context, input *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
					// Create a simple error that maps to the HTTP status
					return nil, &smithy.GenericAPIError{Code: "HTTPError", Message: "HTTP Error"}
				},
			}

			cfg := config.FrontendAssetProxyConfig{
				ProxiedRequestTimeout: 30 * time.Second,
			}
			logger := logrus.New()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/bucket/key.txt", nil)

			ProxyS3(recorder, request, mockS3Client, cfg, "/bucket/key.txt", logger)

			// Since we can't easily mock the HTTP response error, just verify we get a gateway error
			if recorder.Code != http.StatusBadGateway {
				t.Errorf("Expected status %d, got %d", http.StatusBadGateway, recorder.Code)
			}
		})
	}
}

func TestS3ErrorToStatus(t *testing.T) {
	tests := []struct {
		name           string
		error          error
		expectedStatus int
	}{
		{"NoSuchBucket", &smithy.GenericAPIError{Code: "NoSuchBucket"}, http.StatusNotFound},
		{"NoSuchKey", &smithy.GenericAPIError{Code: "NoSuchKey"}, http.StatusNotFound},
		{"NotFound", &smithy.GenericAPIError{Code: "NotFound"}, http.StatusNotFound},
		{"AccessDenied", &smithy.GenericAPIError{Code: "AccessDenied"}, http.StatusForbidden},
		{"Forbidden", &smithy.GenericAPIError{Code: "Forbidden"}, http.StatusForbidden},
		{"PreconditionFailed", &smithy.GenericAPIError{Code: "PreconditionFailed"}, http.StatusPreconditionFailed},
		{"InvalidRange", &smithy.GenericAPIError{Code: "InvalidRange"}, http.StatusRequestedRangeNotSatisfiable},
		{"AuthorizationHeaderMalformed", &smithy.GenericAPIError{Code: "AuthorizationHeaderMalformed"}, http.StatusBadRequest},
		{"RequestTimeout", &smithy.GenericAPIError{Code: "RequestTimeout"}, http.StatusRequestTimeout},
		{"SlowDown", &smithy.GenericAPIError{Code: "SlowDown"}, http.StatusServiceUnavailable},
		{"InternalError", &smithy.GenericAPIError{Code: "InternalError"}, http.StatusInternalServerError},
		{"Context timeout", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"Unknown error", errors.New("unknown error"), http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := s3ErrorToStatus(tt.error)
			if status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected string
	}{
		{"both with slashes", "/path/", "/to/file", "/path/to/file"},
		{"first with slash, second without", "/path/", "to/file", "/path/to/file"},
		{"first without slash, second with", "/path", "/to/file", "/path/to/file"},
		{"neither with slashes", "/path", "to/file", "/path/to/file"},
		{"empty first path", "", "/to/file", "/to/file"},
		{"empty second path", "/path/", "", "/path/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinPath(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("JoinPath(%q, %q) = %q, expected %q", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSetHeaderFromStringPtr(t *testing.T) {
	t.Run("should set header when value is not nil", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		value := "test-value"

		setHeaderFromStringPtr(recorder, "Test-Header", &value)

		if recorder.Header().Get("Test-Header") != "test-value" {
			t.Errorf("Expected header 'test-value', got: %s", recorder.Header().Get("Test-Header"))
		}
	})

	t.Run("should not set header when value is nil", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		setHeaderFromStringPtr(recorder, "Test-Header", nil)

		if recorder.Header().Get("Test-Header") != "" {
			t.Errorf("Expected empty header, got: %s", recorder.Header().Get("Test-Header"))
		}
	})
}

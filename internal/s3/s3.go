package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/RedHatInsights/frontend-asset-proxy/internal/config"
	"github.com/RedHatInsights/frontend-asset-proxy/internal/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/sirupsen/logrus"
)

func NewS3ClientFromConfig(cfg config.FrontendAssetProxyConfig, log *logrus.Logger) *s3.Client {
	var loadOpts []func(*awsconfig.LoadOptions) error
	loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.Region))
	loadOpts = append(loadOpts, awsconfig.WithLogger(logger.ContextAwareLogger{Base: log}))
	loadOpts = append(loadOpts, awsconfig.WithClientLogMode(cfg.ClientLogMode))
	loadOpts = append(loadOpts, awsconfig.WithRetryMaxAttempts(cfg.MaxRetryAttempts))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")))
	} else if cfg.UpstreamURL != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		panic(err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.UpstreamURL != "" {
			if u, err := url.Parse(cfg.UpstreamURL); err == nil && u.Scheme != "" && u.Host != "" {
				o.BaseEndpoint = aws.String(u.Scheme + "://" + u.Host)
				o.UsePathStyle = true
			}
		}
	})
}

// ProxyS3 resolves bucket/key from full path "/bucket/..." and streams from S3/MinIO
func ProxyS3(w http.ResponseWriter, r *http.Request, s3c *s3.Client, cfg config.FrontendAssetProxyConfig, full string, log *logrus.Logger) {
	path := strings.TrimPrefix(full, "/")
	idx := strings.IndexByte(path, '/')
	if idx <= 0 || idx >= len(path)-1 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	bucket := path[:idx]
	key := path[idx+1:]
	if ukey, err := url.PathUnescape(key); err == nil {
		key = ukey
	}

	ctx, cancel := context.WithTimeout(r.Context(), cfg.ProxiedRequestTimeout)
	defer cancel()

	// Honor basic conditional and range headers
	in := &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	if v := r.Header.Get("Range"); v != "" {
		in.Range = aws.String(v)
	}
	if v := r.Header.Get("If-None-Match"); v != "" {
		in.IfNoneMatch = aws.String(v)
	}
	if v := r.Header.Get("If-Match"); v != "" {
		in.IfMatch = aws.String(v)
	}
	if v := r.Header.Get("If-Modified-Since"); v != "" {
		if t, err := http.ParseTime(v); err == nil {
			in.IfModifiedSince = aws.Time(t)
		}
	}
	if v := r.Header.Get("If-Unmodified-Since"); v != "" {
		if t, err := http.ParseTime(v); err == nil {
			in.IfUnmodifiedSince = aws.Time(t)
		}
	}

	obj, err := s3c.GetObject(ctx, in, func(o *s3.Options) {
		o.Logger = logging.WithContext(r.Context(), logger.ContextAwareLogger{Base: log})
		o.ClientLogMode = cfg.ClientLogMode
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
			if base := s3c.Options().Logger; base != nil {
				logging.WithContext(ctx, base).Logf(logging.Debug, "s3 proxy request timeout bucket=%s key=%s after %v", bucket, key, cfg.ProxiedRequestTimeout)
			}
		}

		// Map common S3 errors to HTTP status
		status := s3ErrorToStatus(err)
		// Optional SPA fallback: on 403/404, serve SPA entry if configured
		// Ensure we only attempt the fallback once by checking current path against SPA path
		if status == http.StatusNotFound || status == http.StatusForbidden {
			if spa := cfg.SPAEntrypointPath; spa != "" {
				spaPath := JoinPath(cfg.BucketPathPrefix, spa)
				if full != spaPath { // guard against recursive fallback
					if base := s3c.Options().Logger; base != nil {
						logging.WithContext(ctx, base).Logf(logging.Debug, "s3 proxy request fallback to SPA entrypoint")
					}
					ProxyS3(w, r, s3c, cfg, spaPath, log)
					return
				}
			}
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	defer obj.Body.Close()

	w.Header().Set("Vary", "Accept-Encoding")
	setHeaderFromStringPtr(w, "Content-Type", obj.ContentType)
	setHeaderFromStringPtr(w, "ETag", obj.ETag)
	setHeaderFromStringPtr(w, "Cache-Control", obj.CacheControl)
	setHeaderFromStringPtr(w, "Content-Encoding", obj.ContentEncoding)
	setHeaderFromStringPtr(w, "Content-Disposition", obj.ContentDisposition)
	setHeaderFromStringPtr(w, "Content-Language", obj.ContentLanguage)
	setHeaderFromStringPtr(w, "Expires", obj.ExpiresString)
	setHeaderFromStringPtr(w, "Accept-Ranges", obj.AcceptRanges)

	if obj.ContentLength != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(*obj.ContentLength, 10))
	}
	if obj.LastModified != nil {
		w.Header().Set("Last-Modified", obj.LastModified.UTC().Format(http.TimeFormat))
	}

	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, obj.Body)
	}
}

// s3ErrorToStatus maps S3 errors to sensible HTTP codes
func s3ErrorToStatus(err error) int {
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout
	}

	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) && respErr != nil && respErr.Response != nil {
		return respErr.Response.StatusCode
	}

	// Unwrap smithy OperationError to its underlying error and re-map
	var opErr *smithy.OperationError
	if errors.As(err, &opErr) && opErr != nil && opErr.Err != nil {
		return s3ErrorToStatus(opErr.Err)
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchBucket", "NoSuchKey", "NotFound", "NoSuchVersion":
			return http.StatusNotFound
		case "AccessDenied", "Forbidden", "SignatureDoesNotMatch", "InvalidAccessKeyId", "ExpiredToken", "RequestTimeTooSkewed", "InvalidObjectState":
			return http.StatusForbidden
		case "PreconditionFailed":
			return http.StatusPreconditionFailed
		case "InvalidRange":
			return http.StatusRequestedRangeNotSatisfiable
		case "AuthorizationHeaderMalformed", "InvalidRequest", "InvalidArgument", "MalformedXML":
			return http.StatusBadRequest
		case "RequestTimeout":
			return http.StatusRequestTimeout
		case "SlowDown", "ServiceUnavailable":
			return http.StatusServiceUnavailable
		case "InternalError":
			return http.StatusInternalServerError
		}
	}

	return http.StatusBadGateway
}

func JoinPath(a, b string) string {
	if strings.HasSuffix(a, "/") && strings.HasPrefix(b, "/") {
		return a + b[1:]
	}
	if !strings.HasSuffix(a, "/") && !strings.HasPrefix(b, "/") {
		return a + "/" + b
	}
	return a + b
}

// setHeaderFromStringPtr sets a response header if the provided value is non-nil.
func setHeaderFromStringPtr(w http.ResponseWriter, key string, val *string) {
	if val != nil {
		w.Header().Set(key, *val)
	}
}

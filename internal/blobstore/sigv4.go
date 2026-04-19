package blobstore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// signRequestV4 adds Authorization and required AWS headers to req for SigV4.
// service is typically "s3".
func signRequestV4(req *http.Request, cfg Config, bodyHash string, service string, now time.Time) error {
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return fmt.Errorf("blobstore: missing access key or secret key")
	}
	region := cfg.Region
	dateStamp := now.UTC().Format("20060102")
	amzDate := now.UTC().Format("20060102T150405Z")

	req.Header.Set("Host", req.URL.Host)
	if bodyHash != "" {
		req.Header.Set("X-Amz-Content-Sha256", bodyHash)
	}
	req.Header.Set("X-Amz-Date", amzDate)

	credScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	signedHeaders, canonicalHeaders := canonicalHeadersFor(req)
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQS := canonicalQueryString(req.URL.Query())

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQS,
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	}, "\n")

	hashCanonical := sha256Hex([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credScope,
		hashCanonical,
	}, "\n")

	signingKey := signingKeyAWS4(cfg.SecretKey, dateStamp, region, service)
	signature := hmacSHA256Hex(signingKey, stringToSign)

	auth := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		cfg.AccessKey, credScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", auth)
	return nil
}

func signingKeyAWS4(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func canonicalHeadersFor(req *http.Request) (signed string, canonical string) {
	keys := make([]string, 0, len(req.Header))
	lower := map[string][]string{}
	for k, vs := range req.Header {
		lk := strings.ToLower(strings.TrimSpace(k))
		if lk == "authorization" {
			continue
		}
		keys = append(keys, lk)
		lower[lk] = vs
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		vals := lower[k]
		sort.Strings(vals)
		joined := strings.Join(vals, ",")
		joined = strings.TrimSpace(joined)
		fmt.Fprintf(&b, "%s:%s\n", k, joined)
	}
	signed = strings.Join(keys, ";")
	return signed, b.String()
}

func canonicalQueryString(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func hmacSHA256Hex(key []byte, data string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

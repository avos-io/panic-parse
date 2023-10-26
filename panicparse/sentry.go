package panicparse

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type HttpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type SentryClient struct {
	Endpoint       string
	PublicKey      string
	SecretKey      string
	UseCompression bool
	DropUntil      time.Time

	httpClient HttpDoer
}

func Init(dsn string) *SentryClient {
	url, err := url.Parse(dsn)
	if err != nil {
		log.Err(err).Str("dsn", dsn).Msg("failed to parse Sentry DSN")
		return nil
	}

	pathElements := strings.Split(url.Path, "/")

	base := url.Scheme + "://" + url.Host + strings.Join(pathElements[:len(pathElements)-1], "/")

	project := pathElements[len(pathElements)-1]

	var publicKey, secretKey string

	if url.User != nil {
		publicKey = url.User.Username()
		if secret, set := url.User.Password(); set {
			secretKey = secret
		} else {
			secretKey = ""
		}
	}

	return &SentryClient{
		Endpoint:       fmt.Sprintf("%s/api/%s/store/", base, project),
		PublicKey:      publicKey,
		SecretKey:      secretKey,
		UseCompression: true,

		httpClient: &http.Client{},
	}
}

func (s *SentryClient) WithHttpClient(client HttpDoer) *SentryClient {
	s.httpClient = client
	return s
}

func (s *SentryClient) Capture(e *Event) (string, error) {
	if s.DropUntil.After(time.Now()) {
		return "", errors.New("dropping event - rate limited")
	}

	id, err := s.trySendCrashReport(e)

	if err == nil {
		return id, nil
	}

	log.Err(err).Msg("failed to send crash report")

	log.Warn().Msg("retrying crash report without panic log")

	delete(e.Extra, "panic")

	return s.trySendCrashReport(e)
}

func (s *SentryClient) trySendCrashReport(e *Event) (string, error) {
	data, err := e.MarshalJSON()
	if err != nil {
		return "", err
	}

	var sentrySecret string
	if s.SecretKey != "" {
		sentrySecret = fmt.Sprintf(", sentry_secret=%s", s.SecretKey)
	}

	var payload io.Reader
	var size int
	var headers http.Header
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=panic-parse/0.0.1, sentry_key="+s.PublicKey+sentrySecret)

	if s.UseCompression {
		var compressedData bytes.Buffer
		gzipWriter := gzip.NewWriter(&compressedData)
		size, err = gzipWriter.Write(data)
		if err != nil {
			log.Err(err).Msg("failed to compress payload")
			return "", err
		}
		gzipWriter.Close()
		payload = &compressedData
		headers.Set("Content-Encoding", "gzip")
	} else {
		payload = bytes.NewReader(data)
		size = len(data)
	}

	headers.Set("Content-Length", fmt.Sprintf("%d", size))

	log.Debug().Str("url", s.Endpoint).Int("payload size", size).Msg("sending crash report")

	req, err := http.NewRequest("POST", s.Endpoint, payload)
	if err != nil {
		return "", err
	}

	req.Header = headers

	res, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	switch res.StatusCode {
	case 200:
		// ok
	case 429:
		retryAfter := res.Header.Get("Retry-After")
		timeout, err := strconv.ParseInt(retryAfter, 0, 32)
		s.DropUntil = time.Now().Add(time.Duration(timeout) * time.Second)
		if err != nil {
			log.Err(err).Str("retry-after", retryAfter).Time("when", s.DropUntil).Msg("failed to parse Retry-After header")
		}
	default:
		return "", fmt.Errorf("http request failed with status: %s body: %v", res.Status, string(body))
	}

	var resp struct {
		Id string `json:"id"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	return resp.Id, nil
}

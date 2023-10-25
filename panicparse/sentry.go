package panicparse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

type SentryClient struct {
	Endpoint  string
	PublicKey string
	SecretKey string
}

func FromDNS(dsn string) *SentryClient {
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
		Endpoint:  fmt.Sprintf("%s/api/%s/store/", base, project),
		PublicKey: publicKey,
		SecretKey: secretKey,
	}
}

func (s *SentryClient) SendCrashReport(e *Event) (string, error) {
	client := &http.Client{}

	payload, err := e.MarshalJSON()
	if err != nil {
		return "", err
	}

	log.Debug().Str("url", s.Endpoint).Msg("sending crash report")

	req, err := http.NewRequest("POST", s.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	var sentrySecret string
	if s.SecretKey != "" {
		sentrySecret = fmt.Sprintf(", sentry_secret=%s", s.SecretKey)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=panicparse/0.0.1, sentry_key="+s.PublicKey+sentrySecret)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
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

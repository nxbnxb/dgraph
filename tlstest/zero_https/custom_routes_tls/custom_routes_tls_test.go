package custom_routes_tls

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

type testCase struct {
	url        string
	statusCode int
	response   string
}

var testCasesHttp = []testCase{
	{
		url:        "http://localhost:6180/health",
		response:   "OK",
		statusCode: 200,
	},
	{
		url:        "http://localhost:6180/state",
		response:   "Client sent an HTTP request to an HTTPS server.",
		statusCode: 400,
	},
}

func TestZeroWithCustomTLSWithHTTPClient(t *testing.T) {
	client := http.Client{
		Timeout: time.Second * 10,
	}
	defer client.CloseIdleConnections()
	for _, test := range testCasesHttp {
		request, err := http.NewRequest("GET", test.url, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		do, err := client.Do(request)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if do != nil && do.StatusCode != test.statusCode {
			t.Fatalf("status code is not same. Got: %d Expected: %d", do.StatusCode, test.statusCode)
		}

		body := readResponseBody(t, do)
		if !strings.Contains(string(body), test.response) {
			t.Fatalf("response is not same. Got: %s Expected: %s", string(body), test.response)
		}
	}
}

var testCasesHttps = []testCase{
	{
		url:       "https://localhost:6180/health",
		response:  "OK",
		statusCode: 200,
	},
	{
		url:       "https://localhost:6180/state",
		response:  "\"id\":\"1\",\"groupId\":0,\"addr\":\"zero1:5180\",\"leader\":true,\"amDead\":false",
		statusCode: 200,
	},
}

func TestZeroWithCustomTLSWithTLSClient(t *testing.T) {
	pool, err := generateCertPool("/dgraph-tls/ca.crt", true)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	tlsCfg := &tls.Config{RootCAs: pool, ServerName: "localhost", InsecureSkipVerify: true}
	tr := &http.Transport{
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
		TLSClientConfig: tlsCfg,
	}
	client := http.Client{
		Transport: tr,
	}

	defer client.CloseIdleConnections()
	for _, test := range testCasesHttps {
		request, err := http.NewRequest("GET", test.url, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		do, err := client.Do(request)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if do != nil && do.StatusCode != test.statusCode {
			t.Fatalf("status code is not same. Got: %d Expected: %d", do.StatusCode, test.statusCode)
		}
		body := readResponseBody(t, do)
		if !strings.Contains(string(body), test.response) {
			t.Fatalf("response is not same. Got: %s Expected: %s", string(body), test.response)
		}
	}
}

func readResponseBody(t *testing.T, do *http.Response) []byte {
	defer func() { _ = do.Body.Close() }()
	body, err := ioutil.ReadAll(do.Body)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	return body
}

func generateCertPool(certPath string, useSystemCA bool) (*x509.CertPool, error) {
	var pool *x509.CertPool
	if useSystemCA {
		var err error
		if pool, err = x509.SystemCertPool(); err != nil {
			return nil, err
		}
	} else {
		pool = x509.NewCertPool()
	}

	if len(certPath) > 0 {
		caFile, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		if !pool.AppendCertsFromPEM(caFile) {
			return nil, errors.Errorf("error reading CA file %q", certPath)
		}
	}

	return pool, nil
}
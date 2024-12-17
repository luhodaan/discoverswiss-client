package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var env struct {
	HTTP_URL    string
	HTTP_METHOD string `default:"GET"`

	SUBSCRIPTION_KEY string
}

const ENV_HEADER_PREFIX = "HTTP_HEADER_"

func FailOnError(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		panic(err)
	}
}

func lodgingRequest(url *url.URL, httpHeaders http.Header, httpMethod string) string {
	headers := httpHeaders
	u := url
	req, err := http.NewRequest(httpMethod, u.String(), http.NoBody)
	FailOnError(err, "could not create http request")

	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("error during http request:", "err", err)
		return ""
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("http request returned non-Ok status", "statusCode", resp.StatusCode)
		return ""
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("error reading response body:", "err", err)
		return ""
	}

	// Convert body to string
	return string(body)
}

func customHeaders() http.Header {
	headers := http.Header{}
	for _, env := range os.Environ() {
		for i := 1; i < len(env); i++ {
			if env[i] == '=' {
				envk := env[:i]
				if strings.HasPrefix(envk, ENV_HEADER_PREFIX) {
					envv := env[i+1:]
					headerName, headerValue, found := strings.Cut(envv, ":")
					if !found {
						slog.Error("invalid header config", "env", envk, "val", envv)
						panic("invalid header config")
					}
					headers[headerName] = []string{strings.TrimSpace(headerValue)}
				}
				break
			}
		}
	}
	return headers
}

func main() {

	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", "err", err)
	}

	envconfig.MustProcess("", &env)

	headers := customHeaders()
	//httpMethod := env.HTTP_METHOD
	url, err := url.Parse(env.HTTP_URL)
	FailOnError(err, "failed parsing url")

	body := lodgingRequest(url, headers, "GET")

	fmt.Println(body)

}

package clients

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func ExecuteHttpRequest(method string, url string) (string, error) {
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	httpClient := http.Client{Timeout: 5 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("code [%d], %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return string(body), nil
}

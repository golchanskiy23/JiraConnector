package util

import (
	"JiraConnector/connector/application"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

func jitterBackoff(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	delta := max - min
	return min + time.Duration(rand.Int63n(int64(delta)))
}

func GetJiraResponse(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	cfg := application.App.Config()

	minWait := *cfg.ServiceConfig.MinTimeSleep
	maxWait := *cfg.ServiceConfig.MaxTimeSleep

	waitTime := minWait
	for attempt := 1; attempt <= 5; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}

		if attempt == 5 {
			return nil, fmt.Errorf("achieved maximum amount of attempts: %d, %w", 5, err)
		}

		delay := jitterBackoff(waitTime/2, waitTime)
		fmt.Printf("Request failed: %v — retrying in %v\n", err, delay)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, errors.New("cancelling context reached")
		}

		waitTime *= 2
		if waitTime > maxWait {
			waitTime = maxWait
		}
	}
	return nil, errors.New("unreachable code achieved")
}

func GetJSON(ctx context.Context, client *http.Client, url string, target any) error {
	response, err := GetJiraResponse(ctx, client, url)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		limited := io.LimitReader(response.Body, 2048)
		body, _ := io.ReadAll(limited)
		return fmt.Errorf("request %s returned status %d: %s", url, response.StatusCode, string(body))
	}

	return json.NewDecoder(response.Body).Decode(target)
}

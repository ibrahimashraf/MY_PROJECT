package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"integin/internal/storage"
)

func main() {
	values, err := loadEnvironment("C:/MY PROJECT/private/integin-secrets/integin-pilot.env")
	if err != nil {
		fail(err)
	}

	endpoint := strings.TrimRight(values["INTEGIN_S3_ENDPOINT"], "/")
	bucket := values["INTEGIN_S3_BUCKET"]
	accessKey := values["INTEGIN_S3_ACCESS_KEY"]
	secretKey := values["INTEGIN_S3_SECRET_KEY"]
	region := values["INTEGIN_S3_REGION"]
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" || region == "" {
		fail(fmt.Errorf("pilot S3 environment is incomplete"))
	}

	client := &http.Client{Timeout: 15 * time.Second}
	signer := storage.NewAWSSigV4Signer(accessKey, secretKey, region, "s3")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	create, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint+"/"+bucket, nil)
	if err != nil {
		fail(err)
	}
	if err := signer(create); err != nil {
		fail(err)
	}
	createResponse, err := client.Do(create)
	if err != nil {
		fail(err)
	}
	defer createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusOK && createResponse.StatusCode != http.StatusConflict {
		fail(httpFailure("pilot bucket creation", createResponse))
	}

	list, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/"+bucket+"?list-type=2", nil)
	if err != nil {
		fail(err)
	}
	if err := signer(list); err != nil {
		fail(err)
	}
	listResponse, err := client.Do(list)
	if err != nil {
		fail(err)
	}
	defer listResponse.Body.Close()
	if listResponse.StatusCode != http.StatusOK {
		fail(httpFailure("pilot signed bucket listing", listResponse))
	}

	fmt.Println("PILOT_RUSTFS_BUCKET_AND_SIGNED_LIST_VERIFIED")
}

func loadEnvironment(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values, scanner.Err()
}

func httpFailure(operation string, response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
	return fmt.Errorf("%s returned HTTP %d: %s", operation, response.StatusCode, strings.TrimSpace(string(body)))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "pilot bucket initialization failed:", err)
	os.Exit(1)
}

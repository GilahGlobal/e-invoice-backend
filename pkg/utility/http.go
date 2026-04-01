package utility

import (
	"bytes"
	"einvoice-access-point/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var DefaultHTTPClient HTTPClient = &http.Client{}

type RequestConfig struct {
	URL     string
	Headers map[string]string
	Body    interface{}
}

type Response struct {
	StatusCode int
	Body       []byte
}

func GetRequest(client HTTPClient, config RequestConfig, response interface{}) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, config.URL, nil)
	if err != nil {
		return nil, err
	}

	return doRequest(client, req, config.Headers, response)
}

func GetQueryRequest(client HTTPClient, config RequestConfig, response interface{}, query models.PaginationQuery) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, config.URL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("size", strconv.Itoa(query.Size))
	q.Add("page", strconv.Itoa(query.Page))
	q.Add("sort_by", query.SortBy)
	q.Add("sort_direction_desc", strconv.FormatBool(query.SortDirectionDesc))
	if query.Reference != "" {
		q.Add("reference", query.Reference)
	}
	req.URL.RawQuery = q.Encode()

	return doRequest(client, req, config.Headers, response)
}

func GetQueryPullRequest(client HTTPClient, config RequestConfig, response interface{}, query models.PullDataQuery) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, config.URL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("confirmed", query.Confirmed)
	if query.From != "" {
		q.Add("from", query.From)
		q.Add("to", query.To)
	}
	req.URL.RawQuery = q.Encode()

	return doRequest(client, req, config.Headers, response)
}

func PostRequest(client HTTPClient, config RequestConfig, response interface{}) (*Response, error) {
	body, err := json.Marshal(config.Body)
	if err != nil {
		return nil, err
	}

	log.Println("firs url: ", config.URL)
	req, err := http.NewRequest(http.MethodPost, config.URL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return doRequest(client, req, config.Headers, response)
}

func PutRequest(client HTTPClient, config RequestConfig, response interface{}) (*Response, error) {
	body, err := json.Marshal(config.Body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPut, config.URL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return doRequest(client, req, config.Headers, response)
}

func PatchRequest(client HTTPClient, config RequestConfig, response interface{}) (*Response, error) {
	body, err := json.Marshal(config.Body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPatch, config.URL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return doRequest(client, req, config.Headers, response)
}

func doRequest(client HTTPClient, req *http.Request, headers map[string]string, response interface{}) (*Response, error) {

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Response: %s\n", string(body))

	httpResponse := &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
	}

	if response != nil && len(body) > 0 {
		if err := json.Unmarshal(body, response); err != nil {
			fmt.Println("Error unmarshalling response:", err)
			return httpResponse, err
		}
	}

	return httpResponse, nil
}

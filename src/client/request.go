package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Request struct {
	*http.Request
}

type QueryParam string

type PathParam struct {
	K string
	V string
}

// NewRequest constructs a new http request with url
// Returns error if invalid pathParams provided.
// equal to client's baseUrl + the provided endpoint.
func (client *BaseClient) NewRequest(method string, endpoint string, pathParams ...PathParam) (*Request, error) {
	url, err := client.newUrl(endpoint, pathParams...)
	if err != nil {
		return nil, err
	}
	if r, err := http.NewRequest(
		method,
		url,
		nil,
	); err != nil {
		return nil, err
	} else {
		req := &Request{r}
		for k, v := range client.headers {
			req = req.AddHeader(k, v)
		}
		return req, nil
	}
}

// AddBody marshalls the provided interface to json
// and sets it as the request's body.
func (r *Request) AddBody(v interface{}) error {
	jsonValue, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(strings.NewReader(string(jsonValue)))
	return nil
}

func (r *Request) AddHeader(k string, v string) *Request {
	r.Header.Add(k, v)
	return r
}

func (r *Request) AddQueryParam(k string, v string) *Request {
	q := r.URL.Query()
	q.Add(k, v)
	r.URL.RawQuery = q.Encode()
	return r
}

func (r *Request) AddPathParam(k string, v string) *Request {

	u, err := url.QueryUnescape(r.Url())
	if err == nil {
		u = strings.ReplaceAll(
			u,
			fmt.Sprintf("{%s}", k),
			v,
		)
	}
	r.URL.Path = u
	return r
}

// Do sends a http request and returns the response
func (r *Request) Do() (*http.Response, error) {
	return http.DefaultClient.Do(r.Request)
}

// DoAndRead sends a http request and reads the response's body
func (r *Request) DoAndRead() ([]byte, error) {
	resp, err := http.DefaultClient.Do(r.Request)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// DoAndRead sends a http request and unmarshalls
// the response's body to the provided interface.
func (r *Request) DoAndUnmarshall(i interface{}) error {
	body, err := r.DoAndRead()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, i)
}

// Url returns the request's url as a string
func (request *Request) Url() string {
	return request.URL.String()
}

func (client *BaseClient) newUrl(endpoint string, pathParams ...PathParam) (string, error) {
	for _, p := range pathParams {
		endpoint = strings.ReplaceAll(
			endpoint,
			fmt.Sprintf("{%s}", p.K),
			p.V,
		)
	}
	c := strings.Count(endpoint, "{")
	if c > 0 {
		return "", errors.New(
			fmt.Sprintf(
				"Did not get all the required "+
					"path params for the endpoint '%s'",
				endpoint,
			),
		)
	}
	return client.baseUrl + endpoint, nil
}

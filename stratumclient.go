// Package stratumclient implements a client library for the Stratum
// API. The Stratum API is a query based REST API and have no
// traditional resources. The documentation can be found on the API
// server: https://<server>/stratum/docs/
//
//	 package stratumclient
//
//	 import (
//	         "log"
//	         "fmt"
//	         "github.com/stianwa/stratumclient"
//	 )
//
//	 type Platform struct {
//		ID    int    `json:"id"`
//		Name  string `json:"name"`
//	 }
//
//	 func main() {
//	     c := &stratumclient.Client{Username: "myuser",
//	                                Password: "mypassword",
//	                                BaseURL:  "https://server/stratum/v1"}
//	     if err := c.Open(); err != nil {
//	         log.Fatal(err)
//	     }
//
//	     var platforms []*Platform
//		if err := c.Get("platform/?orderby=name&select=id,name&where=name~linux", &platforms); err != nil {
//	               t.Fatalf("get platforms: %v\n", err)
//		}
//
//		for _, platform := range platforms {
//			fmt.Print("[%d] %s\n", platform.Id, platform.Name)
//		}
//	 }
package stratumclient

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client holds client config and token data.
type Client struct {
	Username           string    `yaml:"username" json:"username"`
	Password           string    `yaml:"password" json:"password"`
	BaseURL            string    `yaml:"baseURL" json:"base_url"`
	UserAgent          string    `yaml:"userAgent" json:"user_agent"`
	Timeout            int       `yaml:"timeout" json:"timeout"`
	InsecureSkipVerify bool      `yaml:"insecureSkipVerify" json:"insecure_skip_verify"`
	prefix             string    `yaml:"-" json:"-"`
	url                *url.URL  `yaml:"-" json:"-"`
	token              string    `yaml:"-" json:"-"`
	validUntil         time.Time `yaml:"-" json:"-"`
	opened             bool      `yaml:"-" json:"-"`
}

// LoginResponse holds the response from a successful login
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// Stringer function for LoginResponse fmt.String() compliant.
func (l *LoginResponse) String() string {
	return fmt.Sprintf("%s %d %s", l.AccessToken, l.ExpiresIn, l.TokenType)
}

// ErrorResponse holds connection and/or API errors.
type ErrorResponse struct {
	Backend    *BackendError `json:"backend,omitempty"`
	Message    string        `json:"error,omitempty"`
	Status     string
	StatusCode int
}

// BackendError holds errors from the API backend (PostgreSQL). In
// case such an error occurs, the BackendError will be added to the
// ErrorResponse.
type BackendError struct {
	SQL      string `json:"sql,omitempty"`
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Code     string `json:"code,omitempty"`
}

// Error function for ErrorResponse in compliance with the Error
// interface.
func (e *ErrorResponse) Error() string {
	var ret []string
	if e.Status != "" {
		ret = append(ret, e.Status)
	}
	if e.Message != "" {
		ret = append(ret, e.Message)
	}
	if e.Backend != nil {
		if e.Backend.SQL != "" {
			ret = append(ret, fmt.Sprintf("sql: %s", e.Backend.SQL))
		}
		if e.Backend.Message != "" {
			ret = append(ret, fmt.Sprintf("message: %s", e.Backend.Message))
		}
		if e.Backend.Code != "" {
			ret = append(ret, fmt.Sprintf("code: %s", e.Backend.Code))
		}
		if e.Backend.Severity != "" {
			ret = append(ret, fmt.Sprintf("severity: %s", e.Backend.Severity))
		}
		if e.Backend.Detail != "" {
			ret = append(ret, fmt.Sprintf("detail: %s", e.Backend.Detail))
		}
	}

	return strings.Join(ret, ": ")
}

// Open makes sure all the necessary configuration fields are set,
// sets default values for missing fields, and logs on to the API
// using Basic authentication. Any further API calls will use the JWT
// token for authorization. The library will transparently refresh the
// JWT token when necessary.
func (c *Client) Open() error {
	if c.Username == "" {
		return fmt.Errorf("missing: Username")
	}
	if c.Password == "" {
		return fmt.Errorf("missing: Password")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("missing: BaseURL")
	}
	if c.Timeout == 0 {
		c.Timeout = 30
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	c.url = u
	if c.url.Path == "" {
		return fmt.Errorf("missing: path part in BaseURL")
	}
	c.prefix = c.url.Path
	c.url.Path = ""

	if err := c.login(); err != nil {
		return err
	}

	c.opened = true

	return nil
}

// Get will perform a GET API call to stratum. It takes a query string
// and a response parameter. If the response parameter is nil, no data
// will be returned. Otherwise the response parameter should be a
// pointer to a slice of struct pointers which the response will be
// unmarshalled into. The function returns an error upon errors
// otherwise nil.
func (c *Client) Get(query string, resp any) error {
	return c.Unmarshal("GET", query, nil, resp)
}

// Delete will perform a DELETE API call to stratum. It takes a query
// string, post data, and a response parameter. The post data should
// be a map or JSON text when post data is provided, otherwise nil. If
// the response parameter is nil, no data will be returned. Otherwise
// the response parameter should be a pointer to a slice of struct
// pointers which the response will be unmarshalled into. The
// function returns an error upon errors otherwise nil.
func (c *Client) Delete(query string, post, resp any) error {
	return c.Unmarshal("DELETE", query, post, resp)
}

// Put will perform a PUT API call to stratum. It takes a query
// string, post data, and a response parameter. The post data should
// be a map or JSON text when post data is provided, otherwise nil. If
// the response parameter is nil, no data will be returned. Otherwise
// the response parameter should be a pointer to a slice of struct
// pointers which the response will be unmarshalled into. The
// function returns an error upon errors otherwise nil.
func (c *Client) Put(query string, post, resp any) error {
	return c.Unmarshal("PUT", query, post, resp)
}

// Post will perform a POST API call to stratum. It takes a query
// string, post data, and a response parameter. The post data should
// be a map or JSON text when post data is provided, otherwise nil. If
// the response parameter is nil, no data will be returned. Otherwise
// the response parameter should be a pointer to a slice of struct
// pointers which the response will be unmarshalled into. The
// function returns an error upon errors otherwise nil.
func (c *Client) Post(query string, post, resp any) error {
	return c.Unmarshal("POST", query, post, resp)
}

// Unmarshal will perform an API call to stratum. It takes a method,
// query string, post data, and a response parameter. The post data
// should be a map or JSON text when post data is provided, otherwise
// nil. If the response parameter is nil, no data will be
// returned. Otherwise the response parameter should be a pointer to a
// slice of struct pointers which the response will be unmarshalled
// into. The function returns an error upon errors otherwise nil.
func (c *Client) Unmarshal(method, query string, data, resp any) error {
	content, err := c.Call(method, query, data)
	if err != nil {
		return err
	}

	if resp != nil {
		return json.Unmarshal(content, resp)
	}

	return nil
}

// Call will perform an API call to stratum. It takes a method, query
// string, and post data. The post data should be a map or JSON text
// when post data is provided, otherwise nil. The function returns the
// response body and an error.
func (c *Client) Call(method, query string, data any) ([]byte, error) {
	method = strings.ToUpper(method)

	if data != nil && method == "GET" {
		return nil, fmt.Errorf("post data not allowed with method %s", method)
	}

	cURL := strings.TrimRight(c.url.String(),"/")
	if query == "login/v1" {
		cURL = cURL + "/" + query
	} else if !c.opened {
		return nil, fmt.Errorf("config not opened with Open()")
	} else {
		cURL = cURL + "/" + strings.TrimRight(strings.TrimLeft(c.prefix,"/"),"/") + "/" + strings.TrimLeft(query,"/")
	}

	u, err := url.Parse(cURL)
	if err != nil {
		return nil, err
	}
	
	var post []byte
	if data != nil {
		switch data := data.(type) {
		case []byte:
			post = data
		default:
			d, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			post = d
		}
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewReader(post))
	if err != nil {
		return nil, err
	}

	agent := "StratumClient/1.0"
	if c.UserAgent != "" {
		agent = agent + " (" + c.UserAgent + ")"
	}
	req.Header.Set("User-Agent", agent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if query == "login/v1" && method == "GET" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.Username+":"+c.Password)))
	} else {
		if c.token == "" || time.Now().After(c.validUntil) {
			// token expired or missing: get a fresh one
			c.token = ""
			c.validUntil = time.Time{}
			if err := c.login(); err != nil {
				return nil, err
			}
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}

	client := http.Client{
		Timeout:   time.Duration(c.Timeout) * time.Second,
		Transport: customTransport,
	}

	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	ct := resp.Header.Get("Content-Type")
	if !(resp.StatusCode == 200 || resp.StatusCode == 201) {
		if ct == "application/json" {
			eresp := &ErrorResponse{}
			if err := json.Unmarshal(body, &eresp); err != nil {
				return nil, err
			}
			eresp.Status = resp.Status
			eresp.StatusCode = resp.StatusCode

			return nil, eresp
		}
		return nil, fmt.Errorf("%s", resp.Status)
	}

	if ct != "application/json" {
		return nil, fmt.Errorf("server responded with unknown Content-Type: %s", ct)
	}

	return body, nil
}

// login will perform the initial login API call. The login is using
// Basic authentication to retrieve a Bearer token (JWT). The function
// returns an error if any.
func (c *Client) login() error {
	body, err := c.Call("GET", "login/v1", nil)
	if err != nil {
		return err
	}

	resp := &LoginResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	c.token = resp.AccessToken
	c.validUntil = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	return nil
}

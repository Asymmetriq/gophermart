package config

import (
	"fmt"
	"net/http"
	"net/url"
)

func NewAccrualClient(accrualURL string) AccrualClient {
	return AccrualClient{
		accrualURL: accrualURL,
		Client:     &http.Client{
			// Timeout: 30 * time.Second,
		},
	}
}

type AccrualClient struct {
	accrualURL string
	*http.Client
}

func (ac *AccrualClient) GetOrderInfo(orderNumber string) (*http.Response, error) {
	req, err := ac.buildRequest(orderNumber)
	if err != nil {
		return nil, err
	}
	return ac.Do(req)
}

func (ac *AccrualClient) buildRequest(orderNumber string) (*http.Request, error) {
	serverURL, err := url.Parse(ac.accrualURL)
	if err != nil {
		return nil, err
	}
	queryURL, err := serverURL.Parse(fmt.Sprintf("/api/orders/%s", orderNumber))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

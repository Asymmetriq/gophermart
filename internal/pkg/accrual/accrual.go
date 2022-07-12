package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Asymmetriq/gophermart/internal/pkg/model"
)

func NewlClient(accrualURL string) Client {
	return Client{
		accrualURL: accrualURL,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type Client struct {
	accrualURL string
	*http.Client
}

func (ac *Client) GetOrderInfo(order model.Order) (model.Order, error) {
	req, err := ac.buildRequest(order.Number)
	if err != nil {
		return model.Order{}, err
	}
	resp, err := ac.Do(req)
	if err != nil {
		return model.Order{}, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return model.Order{}, err
	}
	return order, nil
}

func (ac *Client) buildRequest(orderNumber string) (*http.Request, error) {
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

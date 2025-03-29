package eventbrite

import (
	"event_calendar/internal/models"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	apiKey string
	client *resty.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: resty.New(),
	}
}

func (c *Client) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	// Example API call, adjust according to Eventbrite API spec
	url := "https://www.eventbriteapi.com/v3/events/search/"

	resp, err := c.client.R().
		SetQueryParams(map[string]string{
			"location.address":       city,
			"categories":             category, // needs mapping based on Eventbrite category IDs
			"start_date.range_start": time.Now().Format(time.RFC3339),
			"start_date.range_end":   time.Now().Add(period).Format(time.RFC3339),
			"token":                  c.apiKey,
		}).
		Get(url)

	if err != nil {
		return nil, err
	}

	fmt.Println(string(resp.Body())) // Log for debugging

	// TODO: Parse response into models.Event struct
	return []models.Event{}, nil
}

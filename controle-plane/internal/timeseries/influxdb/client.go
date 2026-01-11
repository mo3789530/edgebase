package influxdb

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// Config holds the configuration for InfluxDB connection
type Config struct {
	URL    string
	Token  string
	Org    string
	Bucket string
}

// Client wraps the InfluxDB client and provides connection management
type Client struct {
	client     influxdb2.Client
	writeAPI   api.WriteAPIBlocking
	queryAPI   api.QueryAPI
	bucketsAPI api.BucketsAPI
	config     Config
}

// NewClient creates a new InfluxDB client
func NewClient(cfg Config) (*Client, error) {
	client := influxdb2.NewClient(cfg.URL, cfg.Token)

	// Validate connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		// Just return error, don't panic. The application can decide to retry or fail.
		// Requirement 5.2 says: "WHEN database connection fails, THE system SHALL log the error and continue operation without blocking function execution"
		// However, this NewClient is likely called at startup.
		// For startup, we might want to return error.
		return nil, fmt.Errorf("failed to check influxdb health: %w", err)
	}
	if health.Status != "pass" {
		return nil, fmt.Errorf("influxdb health check failed: %s", *health.Message)
	}

	writeAPI := client.WriteAPIBlocking(cfg.Org, cfg.Bucket)
	queryAPI := client.QueryAPI(cfg.Org)
	bucketsAPI := client.BucketsAPI()

	return &Client{
		client:     client,
		writeAPI:   writeAPI,
		queryAPI:   queryAPI,
		bucketsAPI: bucketsAPI,
		config:     cfg,
	}, nil
}

// Close closes the connection to InfluxDB
func (c *Client) Close() {
	c.client.Close()
}

// WriteAPI returns the blocking write API
func (c *Client) WriteAPI() api.WriteAPIBlocking {
	return c.writeAPI
}

// QueryAPI returns the query API
func (c *Client) QueryAPI() api.QueryAPI {
	return c.queryAPI
}

// BucketsAPI returns the buckets API
func (c *Client) BucketsAPI() api.BucketsAPI {
	return c.bucketsAPI
}

// Config returns the client configuration
func (c *Client) Config() Config {
	return c.config
}

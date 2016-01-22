package status

import (
	"fmt"
	"net"
	"time"
)

type Client struct {
	conn net.Conn
}

func NewClient(addr string) (*Client, error) {
	// Connect to the status server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed contacting status server: %v", err)
	}
	return &Client{conn}, nil
}

func (c *Client) Set(key string, val float64, ts time.Time) error {
	payload := fmt.Sprintf("%d|%s|%f\n", ts.UnixNano(), key, val)
	_, err := c.conn.Write([]byte(payload))
	return err
}

func (c *Client) Close() {
	c.conn.Close()
}

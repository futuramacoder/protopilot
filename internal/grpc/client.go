package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/futuramacoder/protopilot/internal/messages"
)

// TLSConfig holds TLS-related settings.
type TLSConfig struct {
	Plaintext  bool
	CACert     string // file path
	Cert       string // file path
	Key        string // file path
	ServerName string
}

// Client manages the persistent gRPC connection.
type Client struct {
	conn *grpc.ClientConn
	host string
	tls  TLSConfig
}

// NewClient creates a Client (does not connect yet).
func NewClient(host string, tlsCfg TLSConfig) *Client {
	return &Client{
		host: host,
		tls:  tlsCfg,
	}
}

// Host returns the current host.
func (c *Client) Host() string {
	return c.host
}

// TLS returns the current TLS config.
func (c *Client) TLS() TLSConfig {
	return c.tls
}

// Connect establishes the gRPC connection. Returns a tea.Cmd.
func (c *Client) Connect() tea.Cmd {
	return func() tea.Msg {
		opts, err := c.dialOptions()
		if err != nil {
			return messages.ConnectionChangedMsg{
				Host:      c.host,
				Connected: false,
				Err:       fmt.Errorf("TLS config error: %w", err),
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, c.host, opts...)
		if err != nil {
			return messages.ConnectionChangedMsg{
				Host:      c.host,
				Connected: false,
				Err:       err,
			}
		}

		c.conn = conn
		return messages.ConnectionChangedMsg{
			Host:      c.host,
			Connected: true,
		}
	}
}

// Reconnect closes and re-establishes the connection.
func (c *Client) Reconnect() tea.Cmd {
	return func() tea.Msg {
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		msg := c.Connect()()
		return msg
	}
}

// ChangeHost updates the host and reconnects.
func (c *Client) ChangeHost(host string) tea.Cmd {
	c.host = host
	return c.Reconnect()
}

// Conn returns the current connection (may be nil).
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close cleanly shuts down the connection.
func (c *Client) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) dialOptions() ([]grpc.DialOption, error) {
	var opts []grpc.DialOption

	if c.tls.Plaintext {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsCfg, err := buildTLSConfig(c.tls)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	}

	return opts, nil
}

func buildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{}

	if cfg.CACert != "" {
		caCert, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
		tlsCfg.RootCAs = pool
	}

	if cfg.Cert != "" && cfg.Key != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return nil, fmt.Errorf("loading client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if cfg.ServerName != "" {
		tlsCfg.ServerName = cfg.ServerName
	}

	return tlsCfg, nil
}

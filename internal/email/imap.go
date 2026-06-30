package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"

	"github.com/emersion/go-imap/v2/imapclient"
)

// imapConfig holds the resolved (decrypted) connection parameters for a mailbox.
type imapConfig struct {
	host     string
	port     int
	security Security
	username string
	password string
}

// dialAndLogin opens an IMAP connection using the account's transport security
// and authenticates. The caller is responsible for Logout/Close.
func dialAndLogin(cfg imapConfig) (*imapclient.Client, error) {
	addr := net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port))

	var (
		client *imapclient.Client
		err    error
	)
	switch cfg.security {
	case SecuritySSL:
		client, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: cfg.host, MinVersion: tls.VersionTLS12},
		})
	case SecuritySTARTTLS:
		client, err = imapclient.DialStartTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: cfg.host, MinVersion: tls.VersionTLS12},
		})
	case SecurityNone:
		client, err = imapclient.DialInsecure(addr, nil)
	default:
		return nil, fmt.Errorf("unsupported security %q", cfg.security)
	}
	if err != nil {
		return nil, fmt.Errorf("connect to imap server: %w", err)
	}

	if err := client.Login(cfg.username, cfg.password).Wait(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("imap login failed: %w", err)
	}

	return client, nil
}

// verifyIMAPConnection dials and authenticates with the given config, then
// cleanly closes the connection. It returns the underlying error on failure.
func verifyIMAPConnection(cfg imapConfig) error {
	client, err := dialAndLogin(cfg)
	if err != nil {
		return err
	}
	_ = client.Logout().Wait()
	_ = client.Close()
	return nil
}

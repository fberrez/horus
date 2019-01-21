package client

import (
	"net"
	"time"
)

type (
	Client interface {
		Send(dest *net.IP, port string, packet []byte) ([]byte, error)
		SendWithDeadLine(dest *net.IP, port string, packet []byte, deadline time.Duration) ([]byte, error)
	}

	Protocol string
)

const (
	UDP Protocol = "udp"
)

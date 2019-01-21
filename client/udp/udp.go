package udp

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
)

const (
	DEFAULT_DEADLINE = time.Second * 2
)

type UDP struct {
	Name string `json:"name" yaml:"name"`
}

func (u *UDP) Send(dest *net.IP, port string, packet []byte) ([]byte, error) {
	return u.SendWithDeadLine(dest, port, packet, DEFAULT_DEADLINE)
}

// SendWithDeadLine sends packet to dest and gets back one reply from the bulb
// Note that the way it's implemented means it's possible it'll get multiple replies
// (i.e. ack and a response if those are both asked for in the request)
// and only log the first reply
func (u *UDP) SendWithDeadLine(dest *net.IP, port string, packet []byte, deadline time.Duration) ([]byte, error) {
	log := logrus.WithField("from", "clientUDP")
	// Initializes the UDP cient
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", dest.String(), port))
	if err != nil {
		return nil, errors.Annotate(err, "cannot send udp packet")
	}

	conn, err := net.DialUDP("udp", nil, addr)
	defer conn.Close()
	if err != nil {
		return nil, errors.Annotate(err, "cannot send udp packet")
	}

	log.Debugf("Sending packet to %s", addr.String())
	// Sends the UDP packet
	_, err = conn.Write(packet)
	if err != nil {
		return nil, errors.Annotate(err, "cannot send udp packet")
	}
	// Reads the response
	conn.SetReadDeadline(time.Now().Add(deadline))
	p := make([]byte, 2048)
	_, err = conn.Read(p)
	if err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint16(p[0:2])

	return p[0:size], nil
}

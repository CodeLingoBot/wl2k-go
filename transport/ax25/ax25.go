// Copyright 2015 Martin Hebnes Pedersen (LA5NTA). All rights reserved.
// Use of this source code is governed by the MIT-license that can be
// found in the LICENSE file.

// Package ax25 provides net.Conn interface for AX.25 connections
// through TNCs and axports (on Linux).
package ax25

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/la5nta/wl2k-go/transport"
)

const _NETWORK = "AX.25"

var DefaultDialer = &Dialer{Timeout: 45 * time.Second}

func init() {
	transport.RegisterDialer("ax25", DefaultDialer)
	transport.RegisterDialer("serial-tnc", DefaultDialer)
}

type addr interface {
	Address() Address // Callsign
	Digis() []Address // Digipeaters
}

type AX25Addr struct{ addr }

func (a AX25Addr) Network() string { return _NETWORK }
func (a AX25Addr) String() string {
	var buf bytes.Buffer

	fmt.Fprint(&buf, a.Address())
	if len(a.Digis()) > 0 {
		fmt.Fprint(&buf, " via")
	}
	for _, digi := range a.Digis() {
		fmt.Fprintf(&buf, " %s", digi)
	}

	return buf.String()
}

type Address struct {
	Call string
	SSID uint8
}

type Conn struct {
	io.ReadWriteCloser
	localAddr  AX25Addr
	remoteAddr AX25Addr
}

func (c *Conn) LocalAddr() net.Addr {
	if !c.ok() {
		return nil
	}
	return c.localAddr
}

func (c *Conn) RemoteAddr() net.Addr {
	if !c.ok() {
		return nil
	}
	return c.remoteAddr
}

func (c *Conn) ok() bool { return c != nil }

func (c *Conn) SetDeadline(t time.Time) error {
	return errors.New(`SetDeadline not implemented`)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return errors.New(`SetReadDeadline not implemented`)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return errors.New(`SetWriteDeadline not implemented`)
}

type Beacon interface {
	Now() error
	Every(d time.Duration) error

	LocalAddr() net.Addr
	RemoteAddr() net.Addr

	Message() string
}

type Dialer struct {
	Timeout time.Duration
}

func (d Dialer) DialURL(url *transport.URL) (net.Conn, error) {
	target := url.Target
	if len(url.Digis) > 0 {
		target = fmt.Sprintf("%s via %s", target, strings.Join(url.Digis, " "))
	}

	switch url.Scheme {
	case "ax25":
		return DialAX25Timeout(url.Host, url.User.Username(), target, d.Timeout)
	case "serial-tnc":
		//TODO: This is some badly designed legacy stuff. Need to re-think the whole
		//serial-tnc scheme. See issue #34.
		baudrate := Baudrate(1200)
		if i, _ := strconv.Atoi(url.Params.Get("hbaud")); i > 0 {
			baudrate = Baudrate(i)
		}

		return DialKenwood(
			url.Host,
			url.User.Username(),
			target,
			NewConfig(baudrate),
			nil,
		)
	default:
		return nil, transport.ErrUnsupportedScheme
	}
}

func AddressFromString(str string) Address {
	parts := strings.Split(str, "-")
	addr := Address{Call: parts[0]}
	if len(parts) > 1 {
		ssid, err := strconv.ParseInt(parts[1], 10, 32)
		if err == nil && ssid >= 0 && ssid <= 255 {
			addr.SSID = uint8(ssid)
		}
	}
	return addr
}

func (a Address) String() string {
	if a.SSID > 0 {
		return fmt.Sprintf("%s-%d", a.Call, a.SSID)
	} else {
		return a.Call
	}

}

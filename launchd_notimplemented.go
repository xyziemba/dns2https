// +build !darwin

package main

import (
	"errors"
	"net"
)

func bootstrap() (net.PacketConn, net.Listener, error) {
	return nil, nil, errors.New("launchd not implemented on non Mac operating systems")
}

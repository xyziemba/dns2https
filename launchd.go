package main

// #include <launch.h>
// #include <stdlib.h>
// #include <errno.h>
import "C"
import (
	"errors"
	"fmt"
	"net"
	"os"
	"unsafe"
)

func bootstrap() (net.PacketConn, net.Listener, error) {
	udpFile, err := serviceFilePointer("UdpListener")
	if err != nil {
		return nil, nil, err
	}
	udp, err := net.FilePacketConn(udpFile)

	tcpFile, err := serviceFilePointer("TcpListener")
	if err != nil {
		return nil, nil, err
	}
	tcp, err := net.FileListener(tcpFile)

	return udp, tcp, err
}

func serviceFilePointer(name string) (*os.File, error) {
	var fdCount C.size_t
	var fds *C.int
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	eCode := int(C.launch_activate_socket(cName, &fds, &fdCount))
	if eCode == C.ENOENT {
		return nil, errors.New("OS socket activation failed with ENOENT")
	} else if eCode == C.ESRCH {
		return nil, errors.New("OS socket activation failed with ESRCH")
	} else if eCode == C.EALREADY {
		return nil, errors.New("OS socket activation failed with EALREADY")
	} else if eCode != 0 {
		return nil, fmt.Errorf("OS socket activation failed with unknown error %d", eCode)
	}
	defer C.free(unsafe.Pointer(fds))

	if fdCount > 1 {
		return nil, errors.New("multiple fds returned; I don't know how to deal with that")
	} else if fdCount == 0 {
		return nil, errors.New("unable to get FD for service " + name)
	}

	fdPtr := (uintptr)(*fds)
	return os.NewFile(fdPtr, name), nil
}

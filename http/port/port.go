package port

import (
	"errors"
	"net"

	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// AvailablePort finds an available port to use for any TCP connection
func AvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, stacktrace.Wrap(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, stacktrace.Wrap(err)
	}
	defer l.Close()

	if tcpAddr, ok := l.Addr().(*net.TCPAddr); ok {
		return tcpAddr.Port, nil
	}
	return 0, stacktrace.Wrap(errors.New("failed type assertion"))
}

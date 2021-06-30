// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package listeners

import (
	"bytes"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/dogstatsd/packets"
	"github.com/DataDog/datadog-agent/pkg/dogstatsd/replay"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"fmt"

	"os"
)

type UDSStreamListener struct {
	conn           net.Listener
	packetManager  *packets.PacketManager
	connections    *namedPipeConnections
	trafficCapture *replay.TrafficCapture // Currently ignored
}

// address, err := net.ResolveUnixAddr("unix", path)
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		listener, err := net.ListenUnix("unix", address)
// 		if err != nil {
// 			log.Fatal("listen error:", err)
// 		}

func createUDSconn() (net.Listener, error) {
	socketPath := config.Datadog.GetString("dogstatsd_socket_uds")

	address, addrErr := net.ResolveUnixAddr("unix", socketPath)
	if addrErr != nil {
		return nil, fmt.Errorf("dogstatsd-uds: can't ResolveUnixAddr: %v", addrErr)
	}
	fileInfo, err := os.Stat(socketPath)
	// Socket file already exists
	if err == nil {
		// Make sure it's a UNIX socket
		if fileInfo.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("dogstatsd-uds: cannot reuse %s socket path: path already exists and is not a UNIX socket", socketPath)
		}
		err = os.Remove(socketPath)
		if err != nil {
			return nil, fmt.Errorf("dogstatsd-usd: cannot remove stale UNIX socket: %v", err)
		}
	}

	conn, err := net.ListenUnix("unix", address)
	if err != nil {
		return nil, fmt.Errorf("can't listen: %s", err)
	}

	err = os.Chmod(socketPath, 0722)
	if err != nil {
		return nil, fmt.Errorf("can't set the socket at write only: %s", err)
	}
	return conn, nil
}

// NewUDSStreamListener returns an named pipe Statsd listener
func NewUDSStreamListener(packetOut chan packets.Packets,
	sharedPacketPoolManager *packets.PoolManager, capture *replay.TrafficCapture) (*UDSStreamListener, error) {

	bufferSize := config.Datadog.GetInt("dogstatsd_buffer_size")
	return newUDSStreamListener(
		bufferSize,
		packets.NewPacketManagerFromConfig(packetOut, sharedPacketPoolManager),
		capture)
}

func newUDSStreamListener(
	bufferSize int,
	packetManager *packets.PacketManager,
	capture *replay.TrafficCapture) (*UDSStreamListener, error) {

	// pipe, err := winio.ListenPipe(pipePath, &config)

	con, err := createUDSconn()
	if err != nil {
		return nil, err
	}

	listener := &UDSStreamListener{
		conn:          con,
		packetManager: packetManager,
		connections: &namedPipeConnections{
			newConn:         make(chan net.Conn),
			connToClose:     make(chan net.Conn),
			closeAllConns:   make(chan struct{}),
			allConnsClosed:  make(chan struct{}),
			activeConnCount: 0,
		},
		trafficCapture: capture,
	}

	log.Debugf("uds-stream: %s successfully initialized", con.Addr())
	return listener, nil
}

type namedPipeConnections struct {
	newConn         chan net.Conn
	connToClose     chan net.Conn
	closeAllConns   chan struct{}
	allConnsClosed  chan struct{}
	activeConnCount int32
}

func (l *namedPipeConnections) handleConnections() {
	connections := make(map[net.Conn]struct{})
	requestStop := false
	for stop := false; !stop; {
		select {
		case conn := <-l.newConn:
			connections[conn] = struct{}{}
			atomic.AddInt32(&l.activeConnCount, 1)
		case conn := <-l.connToClose:
			conn.Close()
			delete(connections, conn)
			atomic.AddInt32(&l.activeConnCount, -1)
			if requestStop && len(connections) == 0 {
				stop = true
			}
		case <-l.closeAllConns:
			requestStop = true
			if len(connections) == 0 {
				stop = true
			}
			for conn := range connections {
				// Stop the current execution of net.Conn.Read() and exit listen loop.
				conn.SetReadDeadline(time.Now())
			}

		}
	}
	l.allConnsClosed <- struct{}{}
}

// Listen runs the intake loop. Should be called in its own goroutine
func (l *UDSStreamListener) Listen() {
	go l.connections.handleConnections()
	for {
		conn, err := l.conn.Accept()
		switch {
		case err == nil:
			l.connections.newConn <- conn
			buffer := l.packetManager.CreateBuffer()
			go l.listenConnection(conn, buffer)

		case err.Error() == "use of closed network connection":
			{
				// Called when the pipe listener is closed from Stop()
				log.Debug("UDSStreamListener: stop listening")
				return
			}
		default:
			log.Error(err)
		}
	}
}

func (l *UDSStreamListener) listenConnection(conn net.Conn, buffer []byte) {
	log.Infof("UDSStreamListener: start listening a new named pipe client on %s", conn.LocalAddr())
	startWriteIndex := 0
	var t1, t2 time.Time
	for {
		bytesRead, err := conn.Read(buffer[startWriteIndex:])

		t1 = time.Now()

		if err != nil {
			if err == io.EOF {
				log.Debugf("UDSStreamListener: client disconnected from %s", conn.LocalAddr())
				break
			}

			// UDSStreamListener.Stop uses a timeout to stop listening.
			// if err == winio.ErrTimeout {
			// 	log.Debugf("UDSStreamListener: stop listening a named pipe client on %s", conn.LocalAddr())
			// 	break
			// }
			log.Errorf("UDSStreamListener: error reading packet: %v", err.Error())
			//namedPipeTelemetry.onReadError()
		} else {
			endIndex := startWriteIndex + bytesRead

			// When there is no '\n', the message is partial. LastIndexByte returns -1 and messageSize is 0.
			// If there is a '\n', at least one message is completed and '\n' is part of this message.
			messageSize := bytes.LastIndexByte(buffer[:endIndex], '\n') + 1
			if messageSize > 0 {
				//	namedPipeTelemetry.onReadSuccess(messageSize)

				// PacketAssembler merges multiple packets together and sends them when its buffer is full
				l.packetManager.PacketAssembler.AddMessage(buffer[:messageSize])
			}

			startWriteIndex = endIndex - messageSize

			// If the message is bigger than the buffer size, reset startWriteIndex to continue reading next messages.
			if startWriteIndex >= len(buffer) {
				startWriteIndex = 0
			} else {
				copy(buffer, buffer[messageSize:endIndex])
			}
		}

		t2 = time.Now()
		tlmListener.Observe(float64(t2.Sub(t1).Nanoseconds()), "named_pipe")
	}
	l.connections.connToClose <- conn
}

// Stop closes the connection and stops listening
func (l *UDSStreamListener) Stop() {
	// Request closing connections
	l.connections.closeAllConns <- struct{}{}

	// Wait until all connections are closed
	<-l.connections.allConnsClosed

	l.packetManager.Close()
	l.conn.Close()
}

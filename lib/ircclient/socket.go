// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package ircclient

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/goshuirc/irc-go/ircmsg"
)

type Socket struct {
	Host       string
	Port       int
	TLS        bool
	TLSConfig  *tls.Config
	Conn       net.Conn
	ConnLock   sync.Mutex
	Connected  bool
	Connecting bool
	MessagesIn chan ircmsg.IrcMessage
}

func NewSocket() *Socket {
	return &Socket{}
}

func (socket *Socket) Connect() error {
	socket.Connected = false
	socket.Connecting = true

	destination := net.JoinHostPort(socket.Host, strconv.Itoa(socket.Port))

	// TODO: Timeouts
	var conn net.Conn
	var err error
	if socket.TLS {
		conn, err = tls.Dial("tcp", destination, socket.TLSConfig)
	} else {
		conn, err = net.Dial("tcp", destination)
	}

	socket.Connecting = false

	if err != nil {
		return err
	}

	socket.Connected = true
	socket.Conn = conn

	socket.MessagesIn = make(chan ircmsg.IrcMessage)
	go socket.readInput()

	return nil
}

func (socket *Socket) Close() error {
	if socket.Connected {
		return socket.Conn.Close()
	}

	return nil
}

func (socket *Socket) readInput() {
	reader := bufio.NewReader(socket.Conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.Trim(line, "\r\n")
		println("[S " + socket.Host + "] " + line)
		message, parseErr := ircmsg.ParseLine(line)
		if parseErr == nil {
			socket.MessagesIn <- message
		}
	}

	socket.Connected = false
	close(socket.MessagesIn)
}

// WriteLine writes a raw IRC line to the server. Auto appends \n
func (socket *Socket) WriteLine(format string, args ...interface{}) (int, error) {
	if !socket.Connected {
		return 0, fmt.Errorf("not connected")
	}

	line := ""

	if len(args) == 0 {
		line = format
		if !strings.HasSuffix(line, "\n") {
			line += "\n"
		}
	} else {
		if strings.HasSuffix(format, "\n") {
			line = fmt.Sprintf(format, args...)
		} else {
			line = fmt.Sprintf(format+"\n", args...)
		}
	}

	println("[C " + socket.Host + "] " + strings.Trim(line, "\n"))
	return socket.Write([]byte(line))
}

func (socket *Socket) Write(p []byte) (n int, err error) {
	socket.ConnLock.Lock()
	defer socket.ConnLock.Unlock()
	return socket.Conn.Write(p)
}

package main

import (
	"golang.org/x/net/websocket"
)

type uibroadcaster struct {
	sockets  []*websocket.Conn
	messages chan []byte
}

func NewUIBroadcaster() *uibroadcaster {
	ret := &uibroadcaster{
		sockets:  make([]*websocket.Conn, 0),
		messages: make(chan []byte, 1024),
	}
	go ret.writer()
	return ret
}

func (u *uibroadcaster) Send(msg []byte) {
	u.messages <- msg
}

func (u *uibroadcaster) AddSocket(sock *websocket.Conn) {
	u.sockets = append(u.sockets, sock)
}

func (u *uibroadcaster) writer() {
	for {
		msg := <-u.messages
		// Send to all.
		p := make([]*websocket.Conn, 0) // Keep a list of the writeable sockets.
		for _, sock := range u.sockets {
			_, err := sock.Write(msg)
			if err == nil {
				p = append(p, sock)
			}
		}
		u.sockets = p // Save the list of writeable sockets.
	}
}

/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	uibroadcast.go: Helper functions for managementinterface - notification channels for update "subscriptions"
	 (used for weather and traffic websockets).
*/

package main

import (
	"encoding/json"
	"golang.org/x/net/websocket"
	"sync"
	"time"
)

type uibroadcaster struct {
	sockets    []*websocket.Conn
	sockets_mu *sync.Mutex
	messages   chan []byte
}

func NewUIBroadcaster() *uibroadcaster {
	ret := &uibroadcaster{
		sockets:    make([]*websocket.Conn, 0),
		sockets_mu: &sync.Mutex{},
		messages:   make(chan []byte, 1024),
	}
	go ret.writer()
	return ret
}

func (u *uibroadcaster) Send(msg []byte) {
	u.messages <- msg
}

func (u *uibroadcaster) SendJSON(i interface{}) {
	j, _ := json.Marshal(&i)
	u.Send(j)
}

func (u *uibroadcaster) AddSocket(sock *websocket.Conn) {
	u.sockets_mu.Lock()
	u.sockets = append(u.sockets, sock)
	u.sockets_mu.Unlock()
}

func (u *uibroadcaster) writer() {
	for {
		msg := <-u.messages
		// Send to all.
		p := make([]*websocket.Conn, 0) // Keep a list of the writeable sockets.
		u.sockets_mu.Lock()
		for _, sock := range u.sockets {
			err := sock.SetWriteDeadline(time.Now().Add(time.Second))
			_, err2 := sock.Write(msg)
			if err == nil && err2 == nil {
				p = append(p, sock)
			}
		}
		u.sockets = p // Save the list of writeable sockets.
		u.sockets_mu.Unlock()
	}
}

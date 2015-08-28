package main

import (
  "encoding/json"
  "log"
  "time"
  "golang.org/x/net/websocket"
  "io"
  "net/http"
)

type SettingMessage struct {
  Setting string `json:"setting"`
  Value   string `json:"state"`
}

func statusSender(conn *websocket.Conn) {
  timer := time.NewTicker(1 * time.Second)
  for {
    <-timer.C

    resp, _ := json.Marshal(&globalStatus)
    _, err := conn.Write(resp)

    if err != nil {
      log.Printf("Web client disconnected.\n")
      break
    }
  }
}

func handleManagementConnection(conn *websocket.Conn) {
  go statusSender(conn)

  for {
    var msg SettingMessage
    err := websocket.JSON.Receive(conn, &msg)
    if err == io.EOF {
      break
    } else if err != nil {
      log.Printf("handleManagementConnection: %s\n", err.Error())
    } else {
      // TODO: Update specified setting

      // TODO: Send new setting to all the other clients
    }
  }
}

func managementInterface() {
  http.HandleFunc("/",
    func (w http.ResponseWriter, req *http.Request) {
      s := websocket.Server{
        Handler: websocket.Handler(handleManagementConnection)}
      s.ServeHTTP(w, req)
    });

  err := http.ListenAndServe(managementAddr, nil)

  if err != nil {
    log.Printf("managementInterface ListenAndServe: %s\n", err.Error())
  }
}
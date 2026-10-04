package nethost

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// ServeHTTP upgrades the request to a WebSocket and serves protocol messages.
func (host *Host) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if host == nil {
		http.Error(writer, "host unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := &wsClient{conn: conn}
	host.register(client)
	defer host.unregister(client)

	ctx := request.Context()
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if msgType != websocket.MessageText && msgType != websocket.MessageBinary {
			continue
		}
		reply, err := host.dispatch(client, data)
		if err != nil {
			_ = writeBytes(ctx, conn, mustEncodeError("internal", err.Error()))
			continue
		}
		if err := writeBytes(ctx, conn, reply); err != nil {
			return
		}
	}
}

// ListenAndServe starts an HTTP server that only serves this host's WebSocket.
func (host *Host) ListenAndServe(addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           host,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

// Listen starts on addr and returns the bound address (useful when addr ends in :0).
func (host *Host) Listen(addr string) (net.Addr, *http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	server := &http.Server{
		Handler:           host,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = server.Serve(listener)
	}()
	return listener.Addr(), server, nil
}

func writeBytes(ctx context.Context, conn *websocket.Conn, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty websocket write")
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func mustEncodeError(code, message string) []byte {
	raw, err := encodeError(code, message)
	if err != nil {
		return []byte(`{"version":1,"kind":"error","payload":{"code":"internal","message":"encode failed"}}`)
	}
	return raw
}

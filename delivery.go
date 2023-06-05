// Source code file, created by Developer@YANYINGSONG.

package rpcpd

import (
	"fmt"
	"net"
	"rpcpd/seqx"
	"runtime"
	"strings"
	"time"
)

// Connect The client connects to the server.
func Connect(addr string) (*Power, error) {
	channel, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("(net.Dial) %s", err)
	}

	// new Connection.
	conn := newConn()
	conn.Channel = channel
	conn.encryptKey = generateKey(32)

	// new Client.
	cli := &Power{
		serverAddr: strings.Join([]string{"https://", addr}, ""),
		Client:     conn,
	}

	return cli, nil
}

// Listen in server.
func Listen(addr string) (*Power, error) {

	// Server to client channel.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("(net.Listen) %s", err)
	}
	p := &Power{
		Listener: l,
	}

	// Client to server channel.
	// task := func() error {
	// 	server := &http3.Server{Addr: addr, Handler: rpcHandlerByServer()}
	// 	if err = server.ListenAndServeTLS(certFile, keyFile); err != nil {
	// 		return err
	// 	}
	// 	return nil
	// }

	return p, nil
}

// Accept connections from clients or servers.
func (p *Power) Accept() (*Conn, error) {
	channel, err := p.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// new Connection.
	conn := newConn()
	conn.Channel = channel
	conn.encryptKey = generateKey(32)

	// Exchange keys with the client.
	if err = p.exchangeKey(conn, __PACKTYPE_KEY_PING); err != nil {
		return nil, fmt.Errorf("(exchangeKey) %s", err)
	}

	return conn, nil
}

func (p *Power) ConnectionProcessor(conn *Conn) error {
	if conn == nil {
		conn = p.Client
	}

	for {
		if err := p.connectionProcessor(conn); err != nil {
			return fmt.Errorf("(connectionProcessor) %s", err)
		}
	}
}

func (p *Power) Call(conn *Conn, sign, packet []byte) ([]byte, error) {
	if conn == nil {
		conn = p.Client
	}

	// Check sign.
	if sign == nil {
		return nil, fmt.Errorf("call sign is nil")
	}

	// Define the packet header.
	h := &PacketHeader{
		Type: __PACKTYPE_RPC_CALL,
		Sign: int8(len(sign)),
		Seq:  seqx.X.NextID(),
	}

	if p.Client != nil {
		// return p.CallH3(h, sign, packet)
	}

	// Unsigned packets.
	// if sign == nil {
	// 	_, resp, err := p.packetWrite(conn, h, packet)
	// 	return resp, err
	// }

	// There are signed packets.
	data := make([]byte, 0, len(sign)+len(packet))
	data = append(sign, packet...)
	_, _, err := p.packetWrite(conn, h, data)
	if err != nil {
		return nil, fmt.Errorf("(packetWrite) %s", err)
	}

	// Wait for return and timeout processing.
	var t = time.NewTimer(30 * time.Second)
	var q = make(chan []byte)
	var resp []byte
	replyWaitSpace.Store(h.Seq, q)
	defer replyWaitSpace.Delete(h.Seq)
	defer t.Stop()
	select {
	case resp = <-q:
	case <-t.C:
		return nil, fmt.Errorf("reply timeout")
	}

	return resp, nil
}

// func (p *Power) CallH3(h *PacketHeader, sign, packet []byte) (*PacketHeader, []byte, error) {
//
// 	// Logs
// 	if debugMode {
// 		fmt.Printf("UDP.R -> %v P=(%s) %s\n", true, sign, packet)
// 	}
//
// 	r := &http.Request{Header: make(http.Header, 10)}
// 	r.Header.Set("ID", strconv.FormatInt(h.Seq, 10))
// 	r.Header.Set("sign", chars.BytesToString(sign))
// 	httpforward.PayloadSet(r, packet)
// 	_, data, err := httpforward.Http3Do(r, p.serverAddr)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	return h, data, err
// }

var DefaultTaskNum = runtime.NumCPU() * 100

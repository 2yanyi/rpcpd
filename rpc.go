// Source code file, created by Developer@YANYINGSONG.

package rpcpd

import (
	"fmt"
	"io"
	"library/generic/chars"
	"net"
	"net/http"
	"strconv"
	"sync"
	"unsafe"
)

// Packet header structure size.
const packetHeaderSize = int(unsafe.Sizeof(PacketHeader{}))

// PacketHeader Packet header structure.
type PacketHeader struct {
	Type int8  `json:"type,omitempty"` // Packet Type
	Code int8  `json:"code,omitempty"` // RPC Reply Code
	Sign int8  `json:"sign,omitempty"` // RPC Call Sign
	Size int32 `json:"size"`           // Packet Size
	Seq  int64 `json:"seq"`            // Packet ID
}

// WireFormat The method converts a PacketHeader structure to a []byte type for serialization over the network.
func (p *PacketHeader) wireFormat() []byte {
	return (*[packetHeaderSize]byte)(unsafe.Pointer(p))[:]
}

func (p *PacketHeader) isKeyPing(typ int8) bool {
	return typ == __PACKTYPE_KEY_PING
}

func (p *PacketHeader) isCall() bool {
	return p.Type == __PACKTYPE_RPC_CALL
}

func (p *PacketHeader) isReply() bool {
	return p.Type == __PACKTYPE_RPC_REPLY
}

// Packet types.
const (
	__PACKTYPE_KEY_PONG = iota - 1
	__PACKTYPE_KEY_PING
	__PACKTYPE_RPC_CALL
	__PACKTYPE_RPC_REPLY
)

// Status codes.
const (
	statusCodeNoReturn = iota - 1
	statusCodeSuccess
	statusCodeFailed
)

// RPC registry for native functions.
var functionRegistry = make(map[string]functionType)

// RPC reply data waiting space.
var replyWaitSpace = &sync.Map{}

// RPC function generic signature definition.
type functionType func(data []byte) ([]byte, error)

// Conn Long connection structure.
type Conn struct {
	ID         string      // Conn ID
	Channel    net.Conn    // Long connection channel
	syncWrite  *sync.Mutex // write lock
	syncRead   *sync.Mutex // read lock
	encryptKey []byte      // encryption key
	decryptKey []byte      // decryption key
}

// Power main structure.
type Power struct {
	serverAddr string
	Client     *Conn
	Listener   net.Listener
}

// Create a new connection.
func newConn() *Conn {
	return &Conn{
		syncWrite: &sync.Mutex{},
		syncRead:  &sync.Mutex{},
	}
}

// AddFunction Add functionality to the registry.
func AddFunction(sign string, fn functionType) {
	functionRegistry[sign] = fn
}

// RPC function call addressing.
func rpc(header *PacketHeader, fn, data []byte) (h *PacketHeader, sign, packet []byte, err error) {
	h, sign = header, fn

	if sign == nil {
		return
	}

	run, ok := functionRegistry[string(sign)]
	if !ok {
		err = fmt.Errorf("failed: cannot found the function: %s", sign)
		return
	}

	packet, err = run(data)
	if err != nil {
		err = fmt.Errorf("failed: %s", err)
		return
	}

	return
}

// RPC call, server processing.
func rpcHandlerByServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		ID := r.Header.Get("ID")
		fn := r.Header.Get("sign")

		data := make([]byte, r.ContentLength)
		if _, err := io.ReadFull(r.Body, data); err != nil {
			panic(err)
		}

		// Logs
		if debugMode {
			fmt.Printf("UDP.R -> %v P=(%s) %s\n", false, fn, data)
		}

		// RPC call, server processing.
		_, _, packet, err := rpc(nil, chars.StringToBytes(fn), data)
		if err != nil {
			panic(err)
		}

		// reply packet.
		w.Header().Set("ID", ID)
		w.Header().Set("sign", fn)
		w.Header().Set("Content-Length", strconv.Itoa(len(packet)))
		_, _ = w.Write(packet)
	})

	return mux
}

func DebugOn() {
	debugMode = true
}

var debugMode bool

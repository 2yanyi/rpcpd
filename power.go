// Source code file, created by Developer@YANYINGSONG.

package rpcpd

import (
	"errors"
	"fmt"
	"library/generic/chars"
	"math/rand"
	"time"
)

func (p *Power) write(conn *Conn, h *PacketHeader, packet []byte) error {
	conn.syncWrite.Lock()
	defer conn.syncWrite.Unlock()

	// 1). Write packets header.
	h.Size = int32(len(packet))
	if _, err := conn.Channel.Write(h.wireFormat()); err != nil {
		return err
	}
	if h.Size == 0 {
		return nil
	}

	// 2). Write packets.
	if _, err := conn.Channel.Write(packet); err != nil {
		return err
	}

	return nil
}

func (p *Power) read(conn *Conn, h *PacketHeader, packet *[]byte) error {
	conn.syncRead.Lock()
	defer conn.syncRead.Unlock()

	// 1). Read packets header.
	if _, err := conn.Channel.Read(h.wireFormat()); err != nil {
		return fmt.Errorf("(read:header) %s", err)
	}
	if h.Size == 0 {
		return nil
	}

	// 2). Read packets.
	*packet = make([]byte, h.Size)
	if _, err := conn.Channel.Read(*packet); err != nil {
		return fmt.Errorf("(read:data) %s", err)
	}

	return nil
}

func (p *Power) packetWrite(conn *Conn, h *PacketHeader, packet []byte) (*PacketHeader, []byte, error) {

	// Logs
	if debugMode {
		fmt.Printf("SOCK.W -> H=%s P=%s\n", chars.ToJsonBytes(h, ""), packet)
	}

	// Encrypt the data to be sent.
	if !h.isKeyPing(h.Type) {
		p.encrypt(conn.encryptKey, packet)
	}

	// Write packets.
	if err := p.write(conn, h, packet); err != nil {
		h.Code = statusCodeFailed
		return h, nil, err
	}

	h.Code = statusCodeSuccess
	return h, nil, nil
}

func (p *Power) packetRead(conn *Conn) (*PacketHeader, []byte, []byte, error) {

	var h = &PacketHeader{}
	var sign, packet []byte

	// Read packets.
	if err := p.read(conn, h, &packet); err != nil {
		h.Code = statusCodeFailed
		return h, nil, nil, err
	}

	if packet != nil {

		// Decrypt the returned data.
		if !h.isKeyPing(h.Type) {
			p.decrypt(conn.decryptKey, packet)
		}

		// Low-level unpacking.
		sign = make([]byte, 0, h.Sign)
		if h.Sign > 0 {
			sign = packet[:h.Sign]
			packet = packet[h.Sign:]
		}
	}

	// Logs
	if debugMode {
		fmt.Printf("SOCK.R -> H=%s P=%s %s\n", chars.ToJsonBytes(h, ""), sign, packet)
	}

	return p.packetHandler(conn, h, sign, packet)
}

func (p *Power) packetHandler(conn *Conn, h *PacketHeader, sign, packet []byte) (*PacketHeader, []byte, []byte, error) {

	// RPC call.
	if h.isCall() {
		return rpc(h, sign, packet)
	}

	// Exchange Key.
	if h.isKeyPing(h.Type) {

		if packet == nil {
			h.Code = statusCodeFailed
			return h, nil, nil, errors.New("failed: client is unwilling to exchange keys")
		}
		conn.decryptKey = packet

		// Exchange keys with the server.
		if p.Client != nil {
			if err := p.exchangeKey(conn, __PACKTYPE_KEY_PONG); err != nil {
				h.Code = statusCodeFailed
				return h, nil, nil, fmt.Errorf("(exchangeKey) %s", err)
			}
		}

		h.Code = statusCodeNoReturn
		return h, sign, packet, nil
	}

	h.Code = statusCodeSuccess
	return h, sign, packet, nil
}

func (p *Power) exchangeKey(conn *Conn, typ int8) error {
	h := &PacketHeader{Type: typ, Size: int32(len(conn.encryptKey))}
	_, _, err := p.packetWrite(conn, h, conn.encryptKey)
	if err != nil {
		return fmt.Errorf("(packetWrite) %s", err)
	}

	if h.isKeyPing(h.Type) {
		_, _, _, err = p.packetRead(conn)
		if err != nil {
			return fmt.Errorf("(packetRead) %s", err)
		}
		return nil
	}

	return nil
}

func (p *Power) getReply(seqId int64, data []byte) {
	q, ok := replyWaitSpace.Load(seqId)
	if ok {
		q.(chan []byte) <- data
	}
}

func (p *Power) reply(conn *Conn, h *PacketHeader, sign, packet []byte) error {

	// Define the packet header.
	h.Type = __PACKTYPE_RPC_REPLY

	// Unsigned packets.
	if sign == nil {
		_, _, err := p.packetWrite(conn, h, packet)
		return err
	}

	// There are signed packets.
	data := make([]byte, 0, len(sign)+len(packet))
	data = append(sign, packet...)
	_, _, err := p.packetWrite(conn, h, data)

	return err
}

func (p *Power) connectionProcessor(conn *Conn) error {
	h, sign, packet, err := p.packetRead(conn)
	if err != nil {
		packet = []byte(err.Error())
	}

	if h.isReply() {
		p.getReply(h.Seq, packet)
		return nil
	}

	// Do not reply to packets.
	if h.Code == statusCodeNoReturn {
		return nil
	}

	// reply packet.
	if err = p.reply(conn, h, sign, packet); err != nil {
		return err
	}

	return nil
}

func (p *Power) encrypt(key []byte, data []byte) {
	if key == nil || data == nil {
		return
	}
	j := 0
	for i := 0; i < len(data); i++ {
		data[i] = data[i] ^ key[j]
		j += 1
		if j == len(key) {
			j = 0
		}
	}
}

func (p *Power) decrypt(key []byte, data []byte) {
	if key == nil || data == nil {
		return
	}
	j := 0
	for i := 0; i < len(data); i++ {
		data[i] = data[i] ^ key[j]
		j += 1
		if j == len(key) {
			j = 0
		}
	}
}

// Randomly generate keys.
func generateKey(length int) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := make([]byte, length)
	for i := 0; i < length; i++ {
		c := r.Intn(127)
		if c < 33 {
			c += 33
		}
		s[i] = byte(c)
	}

	return s
}

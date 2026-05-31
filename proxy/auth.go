package proxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"net"

	"github.com/pozii/minegate/internal"
	"github.com/pozii/minegate/packet"
)

// ForwardingMode represents a proxy forwarding mode.
type ForwardingMode int

const (
	ForwardNone      ForwardingMode = iota // No forwarding
	ForwardLegacy                           // BungeeCord legacy (via host:port)
	ForwardModern                           // BungeeCord modern (UUID+IP+profile)
	ForwardVelocity                         // Velocity modern forwarding
)

// ForwardingData holds the data to be appended during forwarding.
type ForwardingData struct {
	Mode     ForwardingMode
	UUID     [16]byte
	IP       net.IP
	Username string
	Profile  []byte // Base64 skin profile (BungeeCord modern)
	Secret   []byte // Velocity secret key
}

// AppendLegacyForwarding appends the client IP to the host for legacy BungeeCord forwarding.
func AppendLegacyForwarding(host string, clientIP net.IP) string {
	return host + "\x00" + clientIP.String() + "\x00"
}

// AppendModernForwarding appends UUID+IP+empty properties to the LoginSuccess
// packet for modern BungeeCord forwarding.
//
// The appended format (after normal LoginSuccess fields):
//
//	UUID (16 bytes)
//	IP   (VarInt-length-prefixed UTF-8 string)
//	Properties (VarInt 0 — empty)
func AppendModernForwarding(pkt packet.Packet, data ForwardingData) (packet.Packet, error) {
	extra := make([]byte, 0, 32)
	extra = append(extra, data.UUID[:]...)

	ipStr := data.IP.String()
	buf := make([]byte, packet.MaxVarIntLen)
	n := packet.PutVarInt(buf, int32(len(ipStr)))
	extra = append(extra, buf[:n]...)
	extra = append(extra, []byte(ipStr)...)

	n = packet.PutVarInt(buf, 0)
	extra = append(extra, buf[:n]...)

	pkt.Data = append(pkt.Data, extra...)
	return pkt, nil
}

// ParseModernForwarding extracts BungeeCord modern forwarding data from
// the extra bytes appended to a LoginSuccess packet. The caller must know
// where normal LoginSuccess fields end; pass everything after that point.
func ParseModernForwarding(extra []byte) (*ForwardingData, error) {
	if len(extra) < 16 {
		return nil, internal.ErrPacketTooShort
	}

	var fd ForwardingData
	copy(fd.UUID[:], extra[:16])
	extra = extra[16:]

	ipLen, remaining, err := packet.ReadVarIntFromBytes(extra)
	if err != nil {
		return nil, err
	}
	if int(ipLen) > len(remaining) || ipLen < 0 {
		return nil, internal.ErrPacketTooShort
	}
	fd.IP = net.ParseIP(string(remaining[:ipLen]))
	// ParseModernForwarding is best-effort; ignore properties count

	return &fd, nil
}

// buildVelocityPlaintext builds the plaintext payload for Velocity forwarding.
func buildVelocityPlaintext(data ForwardingData) []byte {
	var payload []byte
	payload = append(payload, data.UUID[:]...)
	payload = append(payload, []byte(data.Username)...)
	payload = append(payload, 0x00)

	ipStr := data.IP.String()
	buf := make([]byte, packet.MaxVarIntLen)
	n := packet.PutVarInt(buf, int32(len(ipStr)))
	payload = append(payload, buf[:n]...)
	payload = append(payload, []byte(ipStr)...)

	portBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(portBuf, 0)
	payload = append(payload, portBuf...)

	return payload
}

// Velocity channel name used for forwarding.
const VelocityChannel = "velocity:player_info"

// AppendVelocityForwarding wraps an existing plugin message with Velocity
// forwarding data, creating a new plugin message packet.
func AppendVelocityForwarding(pkt packet.Packet, data ForwardingData) (packet.Packet, error) {
	plaintext := buildVelocityPlaintext(data)
	payload := signVelocityPayload(plaintext, data.Secret)

	channelBytes := []byte(VelocityChannel)
	channelLen := make([]byte, packet.MaxVarIntLen)
	cl := packet.PutVarInt(channelLen, int32(len(channelBytes)))

	result := make([]byte, 0, cl+len(channelBytes)+len(payload))
	result = append(result, channelLen[:cl]...)
	result = append(result, channelBytes...)
	result = append(result, payload...)

	return packet.Packet{ID: 0x00, Data: result}, nil
}

// CreateVelocityForwardingPacket creates a Velocity forwarding plugin message packet.
// The packet ID 0x00 corresponds to the identifier packet in the login/play state.
func CreateVelocityForwardingPacket(data ForwardingData) (packet.Packet, error) {
	plaintext := buildVelocityPlaintext(data)
	payload := signVelocityPayload(plaintext, data.Secret)

	channelBytes := []byte(VelocityChannel)
	channelLen := make([]byte, packet.MaxVarIntLen)
	cl := packet.PutVarInt(channelLen, int32(len(channelBytes)))

	result := make([]byte, 0, cl+len(channelBytes)+len(payload))
	result = append(result, channelLen[:cl]...)
	result = append(result, channelBytes...)
	result = append(result, payload...)

	return packet.Packet{ID: 0x00, Data: result}, nil
}

// signVelocityPayload computes HMAC-SHA256 of the plaintext and prepends it.
func signVelocityPayload(plaintext, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(plaintext)
	signature := mac.Sum(nil)

	result := make([]byte, 0, len(signature)+len(plaintext))
	result = append(result, signature...)
	result = append(result, plaintext...)
	return result
}

// ValidateVelocityForwarding verifies the HMAC signature on a Velocity
// forwarding payload. Returns the plaintext data if valid.
func ValidateVelocityForwarding(data, secret []byte) ([]byte, bool) {
	if len(data) < 32 {
		return nil, false
	}
	signature := data[:32]
	plaintext := data[32:]

	mac := hmac.New(sha256.New, secret)
	mac.Write(plaintext)
	expected := mac.Sum(nil)

	if !hmac.Equal(signature, expected) {
		return nil, false
	}
	return plaintext, true
}

var _ = internal.ErrHandshakeFailed

package proxy

import (
	"net"
	"testing"

	"github.com/pozii/minegate/packet"
)

func TestBuildAndParseHandshake(t *testing.T) {
	pkt := BuildHandshake(766, "mc.example.com", 25565, StateLogin)

	host, port, state, err := ParseHandshake(pkt)
	if err != nil {
		t.Fatal(err)
	}

	if host != "mc.example.com" {
		t.Errorf("host: got %q, want %q", host, "mc.example.com")
	}
	if port != 25565 {
		t.Errorf("port: got %d, want %d", port, 25565)
	}
	if state != StateLogin {
		t.Errorf("state: got %d, want %d", state, StateLogin)
	}
}

func TestBuildAndParseHandshakeIPv6(t *testing.T) {
	pkt := BuildHandshake(766, "::1", 25565, StateStatus)

	host, port, state, err := ParseHandshake(pkt)
	if err != nil {
		t.Fatal(err)
	}

	if host != "::1" {
		t.Errorf("host: got %q, want %q", host, "::1")
	}
	if port != 25565 {
		t.Errorf("port: got %d, want %d", port, 25565)
	}
	if state != StateStatus {
		t.Errorf("state: got %d, want %d", state, StateStatus)
	}
}

func TestParseHandshakeTooShort(t *testing.T) {
	_, _, _, err := ParseHandshake(packet.Packet{Data: []byte{0x00}})
	if err == nil {
		t.Fatal("expected error for too-short packet")
	}
}

func TestParseHandshakeEmptyData(t *testing.T) {
	_, _, _, err := ParseHandshake(packet.Packet{Data: nil})
	if err == nil {
		t.Fatal("expected error for nil data")
	}
}

func TestParseHandshakeMinimal(t *testing.T) {
	// Protocol version 0, empty host, port 0, state 1
	data := []byte{0x00}             // protoVer=0
	data = append(data, 0x00)        // empty string length
	data = append(data, 0x00, 0x00)  // port=0
	data = append(data, 0x01)        // state=1

	host, port, state, err := ParseHandshake(packet.Packet{Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		t.Errorf("host: got %q, want empty", host)
	}
	if port != 0 {
		t.Errorf("port: got %d, want 0", port)
	}
	if state != StateStatus {
		t.Errorf("state: got %d, want %d", state, StateStatus)
	}
}

func TestModifyHandshakeHost(t *testing.T) {
	pkt := BuildHandshake(766, "old.example.com", 25565, StateLogin)

	modified, err := ModifyHandshakeHost(pkt, "new.example.com")
	if err != nil {
		t.Fatal(err)
	}

	host, port, state, err := ParseHandshake(modified)
	if err != nil {
		t.Fatal(err)
	}

	if host != "new.example.com" {
		t.Errorf("host: got %q, want %q", host, "new.example.com")
	}
	if port != 25565 {
		t.Errorf("port: got %d, want %d", port, 25565)
	}
	if state != StateLogin {
		t.Errorf("state: got %d, want %d", state, StateLogin)
	}
}

func TestModifyHandshakeHostShorter(t *testing.T) {
	pkt := BuildHandshake(766, "long-hostname.example.com", 25565, StateLogin)

	modified, err := ModifyHandshakeHost(pkt, "short")
	if err != nil {
		t.Fatal(err)
	}

	host, port, state, err := ParseHandshake(modified)
	if err != nil {
		t.Fatal(err)
	}

	if host != "short" {
		t.Errorf("host: got %q, want %q", host, "short")
	}
	if port != 25565 {
		t.Errorf("port: got %d, want %d", port, 25565)
	}
	if state != StateLogin {
		t.Errorf("state: got %d, want %d", state, StateLogin)
	}
}

func TestModifyHandshakeHostInvalid(t *testing.T) {
	_, err := ModifyHandshakeHost(packet.Packet{Data: []byte{0x01}}, "newhost")
	if err == nil {
		t.Fatal("expected error for invalid packet")
	}
}

func TestAppendLegacyForwarding(t *testing.T) {
	ip := net.ParseIP("192.168.1.100")
	result := AppendLegacyForwarding("mc.example.com", ip)
	expected := "mc.example.com\x00192.168.1.100\x00"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestAppendLegacyForwardingIPv6(t *testing.T) {
	ip := net.ParseIP("::1")
	result := AppendLegacyForwarding("mc.example.com", ip)
	expected := "mc.example.com\x00::1\x00"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestAppendModernForwarding(t *testing.T) {
	uuid := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	ip := net.ParseIP("10.0.0.1")

	data := ForwardingData{
		Mode:     ForwardModern,
		UUID:     uuid,
		IP:       ip,
		Username: "Player1",
	}

	pkt := packet.Packet{ID: 0x02, Data: []byte("fake-login-success-data")}
	origLen := len(pkt.Data)

	result, err := AppendModernForwarding(pkt, data)
	if err != nil {
		t.Fatal(err)
	}

	if result.ID != 0x02 {
		t.Errorf("packet ID changed: got 0x%x, want 0x02", result.ID)
	}
	if len(result.Data) <= origLen {
		t.Fatal("forwarding data was not appended")
	}

	// UUID should start right after original data
	gotUUID := result.Data[origLen : origLen+16]
	var expectedUUID [16]byte
	copy(expectedUUID[:], gotUUID)
	if expectedUUID != uuid {
		t.Errorf("UUID mismatch: got %v, want %v", expectedUUID, uuid)
	}
}

func TestParseModernForwarding(t *testing.T) {
	uuid := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	ip := net.ParseIP("10.0.0.1")

	extra := make([]byte, 0, 32)
	extra = append(extra, uuid[:]...)

	ipStr := ip.String()
	buf := make([]byte, packet.MaxVarIntLen)
	n := packet.PutVarInt(buf, int32(len(ipStr)))
	extra = append(extra, buf[:n]...)
	extra = append(extra, []byte(ipStr)...)
	n = packet.PutVarInt(buf, 0)
	extra = append(extra, buf[:n]...)

	fd, err := ParseModernForwarding(extra)
	if err != nil {
		t.Fatal(err)
	}
	if fd.UUID != uuid {
		t.Errorf("UUID: got %v, want %v", fd.UUID, uuid)
	}
}

func TestParseModernForwardingTooShort(t *testing.T) {
	_, err := ParseModernForwarding([]byte{0, 1, 2})
	if err == nil {
		t.Fatal("expected error for too-short data")
	}
}

func TestCreateVelocityForwardingPacket(t *testing.T) {
	uuid := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	ip := net.ParseIP("10.0.0.1")
	secret := []byte("my-velocity-secret")

	data := ForwardingData{
		Mode:     ForwardVelocity,
		UUID:     uuid,
		IP:       ip,
		Username: "Player1",
		Secret:   secret,
	}

	pkt, err := CreateVelocityForwardingPacket(data)
	if err != nil {
		t.Fatal(err)
	}

	if pkt.ID != 0x00 {
		t.Errorf("packet ID: got 0x%x, want 0x00", pkt.ID)
	}

	if len(pkt.Data) == 0 {
		t.Fatal("forwarding packet data should not be empty")
	}

	// Verify channel prefix: "velocity:player_info"
	chLen, afterCh, err := packet.ReadVarIntFromBytes(pkt.Data)
	if err != nil {
		t.Fatal(err)
	}
	channel := string(afterCh[:chLen])
	if channel != "velocity:player_info" {
		t.Errorf("channel: got %q, want %q", channel, "velocity:player_info")
	}

	// After channel: HMAC(32) + plaintext
	payload := afterCh[chLen:]
	if len(payload) < 32 {
		t.Fatal("payload too short, missing HMAC")
	}

	// Verify HMAC signature
	plaintext, valid := ValidateVelocityForwarding(payload, secret)
	if !valid {
		t.Fatal("HMAC signature validation failed")
	}

	// Verify UUID is at start of plaintext
	var gotUUID [16]byte
	copy(gotUUID[:], plaintext[:16])
	if gotUUID != uuid {
		t.Errorf("UUID mismatch: got %v, want %v", gotUUID, uuid)
	}
}

func TestNewProxy(t *testing.T) {
	p := NewProxy(nil, nil)
	if p == nil {
		t.Fatal("NewProxy should not return nil")
	}
}

func TestProxySetHandler(t *testing.T) {
	p := NewProxy(nil, nil)
	p.SetHandler(func(hc HandlerContext) {
	})
	if p.handler == nil {
		t.Fatal("handler should not be nil after SetHandler")
	}
}

func TestForwardingModeValues(t *testing.T) {
	if ForwardNone != 0 {
		t.Errorf("ForwardNone should be 0, got %d", ForwardNone)
	}
	if ForwardLegacy != 1 {
		t.Errorf("ForwardLegacy should be 1, got %d", ForwardLegacy)
	}
	if ForwardModern != 2 {
		t.Errorf("ForwardModern should be 2, got %d", ForwardModern)
	}
	if ForwardVelocity != 3 {
		t.Errorf("ForwardVelocity should be 3, got %d", ForwardVelocity)
	}
}

func TestHandshakeStateConstants(t *testing.T) {
	if StateStatus != 1 {
		t.Errorf("StateStatus should be 1, got %d", StateStatus)
	}
	if StateLogin != 2 {
		t.Errorf("StateLogin should be 2, got %d", StateLogin)
	}
}

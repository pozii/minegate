package packet

import (
	"bytes"
	"net"
	"testing"
)

func TestPacketRoundTrip(t *testing.T) {
	tests := []Packet{
		{ID: 0x00, Data: []byte{0x01, 0x02, 0x03}},
		{ID: 0x7F, Data: []byte{}},
		{ID: 0xFF, Data: make([]byte, 100)},
		{ID: 0x80, Data: []byte("hello minecraft")},
	}

	for _, pkt := range tests {
		client, server := net.Pipe()

		go func() {
			w := NewPacketWriter(client, -1)
			w.WritePacket(pkt)
			client.Close()
		}()

		r := NewPacketReader(server, -1)
		got, err := r.ReadPacket()
		if err != nil {
			t.Fatal(err)
		}
		server.Close()

		if got.ID != pkt.ID {
			t.Errorf("packet ID = %d, want %d", got.ID, pkt.ID)
		}
		if !bytes.Equal(got.Data, pkt.Data) {
			t.Errorf("packet data mismatch: got %x, want %x", got.Data, pkt.Data)
		}
	}
}

func TestReadWriteRawPacket(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		w := NewPacketWriter(client, -1)
		w.WritePacket(Packet{ID: 0x01, Data: []byte{0xAA, 0xBB, 0xCC}})
		client.Close()
	}()

	r := NewPacketReader(server, -1)
	rp, err := r.ReadRawPacket()
	if err != nil {
		t.Fatal(err)
	}
	defer rp.Release()
	server.Close()

	if rp.Len() == 0 {
		t.Fatal("raw packet is empty")
	}

	id, err := rp.PacketID()
	if err != nil {
		t.Fatal(err)
	}
	if id != 0x01 {
		t.Errorf("raw packet ID = %d, want 1", id)
	}
}

func TestPacketCodec(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		codec := NewPacketCodec(client, -1)
		codec.writer.WritePacket(Packet{ID: 1, Data: []byte{0x01}})
		codec.writer.WritePacket(Packet{ID: 2, Data: []byte{0x02}})
		client.Close()
	}()

	codec := NewPacketCodec(server, -1)
	p1, err := codec.reader.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if p1.ID != 1 {
		t.Errorf("first packet ID = %d, want 1", p1.ID)
	}

	p2, err := codec.reader.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if p2.ID != 2 {
		t.Errorf("second packet ID = %d, want 2", p2.ID)
	}
	server.Close()
}

func TestPacketTooLarge(t *testing.T) {
	r := NewPacketReader(bytes.NewReader([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0x7F}), -1)
	_, err := r.ReadPacket()
	if err == nil {
		t.Error("expected error for oversized packet, got nil")
	}
}

func BenchmarkPacketWrite(b *testing.B) {
	p := Packet{ID: 0x00, Data: make([]byte, 128)}
	buf := &bytes.Buffer{}

	for i := 0; i < b.N; i++ {
		w := NewPacketWriter(buf, -1)
		w.WritePacket(p)
	}
}

func BenchmarkPacketRead(b *testing.B) {
	p := Packet{ID: 0x00, Data: make([]byte, 128)}
	buf := &bytes.Buffer{}
	w := NewPacketWriter(buf, -1)
	w.WritePacket(p)
	data := buf.Bytes()

	for i := 0; i < b.N; i++ {
		r := NewPacketReader(bytes.NewReader(data), -1)
		r.ReadPacket()
	}
}

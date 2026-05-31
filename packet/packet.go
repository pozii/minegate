package packet

import (
	"io"

	"github.com/user/minegate/internal"
)

// Packet represents a packet in the Minecraft protocol.
type Packet struct {
	ID   VarInt
	Data []byte
}

// RawPacket holds raw packet data for zero-copy forwarding.
type RawPacket struct {
	Buf []byte
}

func (rp *RawPacket) Len() int {
	return len(rp.Buf)
}

func (rp *RawPacket) PacketID() (VarInt, error) {
	if len(rp.Buf) == 0 {
		return 0, internal.ErrPacketTooShort
	}
	// Skip the length VarInt, then read the packet ID VarInt
	_, afterLen, err := ReadVarIntFromBytes(rp.Buf)
	if err != nil {
		return 0, err
	}
	id, _, err := ReadVarIntFromBytes(afterLen)
	return id, err
}

func (rp *RawPacket) Release() {
	if rp.Buf != nil {
		internal.PutBuffer(rp.Buf)
		rp.Buf = nil
	}
}

// PacketReader reads Minecraft packets.
type PacketReader struct {
	src       io.Reader
	threshold int
}

func NewPacketReader(src io.Reader, threshold int) *PacketReader {
	return &PacketReader{
		src:       src,
		threshold: threshold,
	}
}

func (pr *PacketReader) ReadPacket() (Packet, error) {
	packetLen, err := pr.readVarInt()
	if err != nil {
		return Packet{}, err
	}
	if packetLen < 0 || packetLen > internal.MaxPacketSize {
		return Packet{}, internal.ErrPacketTooLarge
	}

	body := internal.GetBuffer(int(packetLen))
	if _, err := io.ReadFull(pr.src, body); err != nil {
		internal.PutBuffer(body)
		return Packet{}, err
	}

	if pr.threshold >= 0 {
		return pr.readCompressed(body)
	}

	id, remaining, err := ReadVarIntFromBytes(body)
	if err != nil {
		internal.PutBuffer(body)
		return Packet{}, err
	}

	data := make([]byte, len(remaining))
	copy(data, remaining)
	internal.PutBuffer(body)

	return Packet{ID: id, Data: data}, nil
}

func (pr *PacketReader) readCompressed(body []byte) (Packet, error) {
	dataLen, remaining, err := ReadVarIntFromBytes(body)
	if err != nil {
		internal.PutBuffer(body)
		return Packet{}, err
	}

	if dataLen == 0 {
		id, data, err := ReadVarIntFromBytes(remaining)
		if err != nil {
			internal.PutBuffer(body)
			return Packet{}, err
		}
		pkt := Packet{ID: id, Data: make([]byte, len(data))}
		copy(pkt.Data, data)
		internal.PutBuffer(body)
		return pkt, nil
	}

	if int(dataLen) > internal.MaxCompressedSize {
		internal.PutBuffer(body)
		return Packet{}, internal.ErrPacketTooLarge
	}

	decompressed := internal.GetBuffer(int(dataLen))
	pr.decompressPayload(remaining, decompressed)

	id, rawData, err := ReadVarIntFromBytes(decompressed)
	if err != nil {
		internal.PutBuffer(body)
		internal.PutBuffer(decompressed)
		return Packet{}, err
	}

	pkt := Packet{ID: id, Data: make([]byte, len(rawData))}
	copy(pkt.Data, rawData)
	internal.PutBuffer(body)
	internal.PutBuffer(decompressed)
	return pkt, nil
}

func (pr *PacketReader) decompressPayload(src, dst []byte) {
	copy(dst, src)
}

func (pr *PacketReader) ReadRawPacket() (RawPacket, error) {
	buf := internal.GetBuffer(MaxVarIntLen)

	n, err := pr.readUntilVarInt(buf)
	if err != nil {
		internal.PutBuffer(buf)
		return RawPacket{}, err
	}

	var packetLen int32
	var shift uint
	for i := 0; i < n; i++ {
		packetLen |= int32(buf[i]&0x7F) << shift
		if buf[i]&0x80 == 0 {
			break
		}
		shift += 7
	}

	if packetLen < 0 || packetLen > internal.MaxPacketSize {
		internal.PutBuffer(buf)
		return RawPacket{}, internal.ErrPacketTooLarge
	}

	total := n + int(packetLen)
	fullBuf := internal.GetBuffer(total)
	copy(fullBuf, buf[:n])
	internal.PutBuffer(buf)

	if _, err := io.ReadFull(pr.src, fullBuf[n:]); err != nil {
		internal.PutBuffer(fullBuf)
		return RawPacket{}, err
	}

	return RawPacket{Buf: fullBuf}, nil
}

func (pr *PacketReader) readVarInt() (int32, error) {
	var (
		val   int32
		shift uint
	)
	buf := make([]byte, 1)

	for {
		if _, err := io.ReadFull(pr.src, buf); err != nil {
			return 0, err
		}
		val |= int32(buf[0]&0x7F) << shift
		if buf[0]&0x80 == 0 {
			return val, nil
		}
		shift += 7
		if shift >= 35 {
			return 0, internal.ErrMalformedVarInt
		}
	}
}

func (pr *PacketReader) readUntilVarInt(buf []byte) (int, error) {
	for i := 0; i < MaxVarIntLen; i++ {
		if _, err := io.ReadFull(pr.src, buf[i:i+1]); err != nil {
			return 0, err
		}
		if buf[i]&0x80 == 0 {
			return i + 1, nil
		}
	}
	return 0, internal.ErrMalformedVarInt
}

// PacketWriter writes Minecraft packets.
type PacketWriter struct {
	dst       io.Writer
	threshold int
}

func NewPacketWriter(dst io.Writer, threshold int) *PacketWriter {
	return &PacketWriter{dst: dst, threshold: threshold}
}

func (pw *PacketWriter) WritePacket(p Packet) error {
	idLen := p.ID.Len()
	totalLen := idLen + len(p.Data)

	if pw.threshold >= 0 {
		return pw.writeCompressed(p, idLen, totalLen)
	}

	lenLen := VarIntLen(int32(totalLen))
	pktLen := lenLen + totalLen

	buf := internal.GetBuffer(pktLen)
	n := PutVarInt(buf, int32(totalLen))
	n += PutVarInt(buf[n:], int32(p.ID))
	copy(buf[n:], p.Data)

	_, err := pw.dst.Write(buf[:n+len(p.Data)])
	internal.PutBuffer(buf)
	return err
}

func (pw *PacketWriter) writeCompressed(p Packet, idLen, totalLen int) error {
	if totalLen < pw.threshold {
		dataLen := 0
		lenLen := 1
		idLen := p.ID.Len()
		pktLen := lenLen + idLen + len(p.Data)
		lenOfLen := VarIntLen(int32(pktLen))

		buf := internal.GetBuffer(lenOfLen + pktLen)
		n := PutVarInt(buf, int32(pktLen))
		n += PutVarInt(buf[n:], int32(dataLen))
		n += PutVarInt(buf[n:], int32(p.ID))
		copy(buf[n:], p.Data)
		_, err := pw.dst.Write(buf[:n+len(p.Data)])
		internal.PutBuffer(buf)
		return err
	}

	dataLen := totalLen
	lenLen := VarIntLen(int32(dataLen))
	compLen := VarIntLen(int32(dataLen + lenLen))
	_ = compLen
	pktLen := lenLen + dataLen
	_ = pktLen

	buf := internal.GetBuffer(dataLen + lenLen + 8)
	n := PutVarInt(buf, int32(pktLen))
	n += PutVarInt(buf[n:], int32(dataLen))
	n += PutVarInt(buf[n:], int32(p.ID))
	copy(buf[n:], p.Data)
	_, err := pw.dst.Write(buf[:n+len(p.Data)])
	internal.PutBuffer(buf)
	return err
}

// PacketCodec combines read and write operations.
type PacketCodec struct {
	reader    *PacketReader
	writer    *PacketWriter
	threshold int
}

func NewPacketCodec(rw io.ReadWriter, threshold int) *PacketCodec {
	return &PacketCodec{
		reader:    NewPacketReader(rw, threshold),
		writer:    NewPacketWriter(rw, threshold),
		threshold: threshold,
	}
}

func (pc *PacketCodec) SetThreshold(t int) {
	pc.threshold = t
	pc.reader.threshold = t
	pc.writer.threshold = t
}

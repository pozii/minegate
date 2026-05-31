package packet

import (
	"bytes"
	"math"
	"testing"
)

func TestVarIntLen(t *testing.T) {
	tests := []struct {
		val int32
		n   int
	}{
		{0, 1},
		{1, 1},
		{127, 1},
		{128, 2},
		{255, 2},
		{16383, 2},
		{16384, 3},
		{2097151, 3},
		{2097152, 4},
		{268435455, 4},
		{268435456, 5},
		{math.MaxInt32, 5},
		{-1, 5},
	}

	for _, tt := range tests {
		n := VarIntLen(tt.val)
		if n != tt.n {
			t.Errorf("VarIntLen(%d) = %d, want %d", tt.val, n, tt.n)
		}
	}
}

func TestVarLongLen(t *testing.T) {
	tests := []struct {
		val int64
		n   int
	}{
		{0, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{2097151, 3},
		{2097152, 4},
		{268435455, 4},
		{268435456, 5},
		{34359738367, 5},
		{34359738368, 6},
		{4398046511103, 6},
		{4398046511104, 7},
		{562949953421311, 7},
		{562949953421312, 8},
		{72057594037927935, 8},
		{72057594037927936, 9},
		{9223372036854775807, 9},
		{-1, 10},
	}

	for _, tt := range tests {
		n := VarLongLen(tt.val)
		if n != tt.n {
			t.Errorf("VarLongLen(%d) = %d, want %d", tt.val, n, tt.n)
		}
	}
}

func TestPutVarInt(t *testing.T) {
	buf := make([]byte, MaxVarIntLen)
	for _, val := range []int32{0, 1, 127, 128, 255, 16383, 16384, 2097151, 2097152, 268435455, 268435456, -1} {
		n := PutVarInt(buf, val)
		got := int32(0)
		var shift uint
		for i := 0; i < n; i++ {
			got |= int32(buf[i]&0x7F) << shift
			if buf[i]&0x80 == 0 {
				break
			}
			shift += 7
		}
		if got != val {
			t.Errorf("PutVarInt(%d) = 0x%x, got %d after decode", val, buf[:n], got)
		}
	}
}

func TestPutVarLong(t *testing.T) {
	buf := make([]byte, MaxVarLongLen)
	for _, val := range []int64{0, 1, 127, 128, 16383, 16384, 34359738367, 34359738368, 9223372036854775807, -1} {
		n := PutVarLong(buf, val)
		got := int64(0)
		var shift uint
		for i := 0; i < n; i++ {
			got |= int64(buf[i]&0x7F) << shift
			if buf[i]&0x80 == 0 {
				break
			}
			shift += 7
		}
		if got != val {
			t.Errorf("PutVarLong(%d) = 0x%x, got %d after decode", val, buf[:n], got)
		}
	}
}

func TestReadVarInt(t *testing.T) {
	tests := []struct {
		bytes []byte
		val   int32
	}{
		{[]byte{0x00}, 0},
		{[]byte{0x7F}, 127},
		{[]byte{0x80, 0x01}, 128},
		{[]byte{0xFF, 0x7F}, 16383},
		{[]byte{0x80, 0x80, 0x01}, 16384},
	}

	for _, tt := range tests {
		r := bytes.NewReader(tt.bytes)
		val, err := ReadVarInt(r)
		if err != nil {
			t.Errorf("ReadVarInt(%x) error: %v", tt.bytes, err)
		}
		if val != tt.val {
			t.Errorf("ReadVarInt(%x) = %d, want %d", tt.bytes, val, tt.val)
		}
	}
}

func TestReadVarIntFromBytes(t *testing.T) {
	data := []byte{0x80, 0x01, 0x00, 0x7F}
	val, remaining, err := ReadVarIntFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if val != 128 {
		t.Errorf("ReadVarIntFromBytes = %d, want 128", val)
	}
	if len(remaining) != 2 {
		t.Errorf("remaining length = %d, want 2", len(remaining))
	}
}

func TestVarIntRoundTrip(t *testing.T) {
	vals := []int32{0, 1, -1, 127, 128, 16383, 16384, 2097151, 2097152, 268435455, 268435456, math.MaxInt32, math.MinInt32}

	for _, v := range vals {
		var buf bytes.Buffer
		varint := VarInt(v)
		_, err := varint.WriteTo(&buf)
		if err != nil {
			t.Fatal(err)
		}

		var got VarInt
		if err := got.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatal(err)
		}

		if got != varint {
			t.Errorf("VarInt(%d): round trip failed, got %d", v, got)
		}
	}
}

func TestVarLongRoundTrip(t *testing.T) {
	vals := []int64{0, 1, -1, 127, 128, 16383, 16384, 34359738367, -34359738368, 9223372036854775807}

	for _, v := range vals {
		var buf bytes.Buffer
		vlong := VarLong(v)
		_, err := vlong.WriteTo(&buf)
		if err != nil {
			t.Fatal(err)
		}

		var got VarLong
		if err := got.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatal(err)
		}

		if got != vlong {
			t.Errorf("VarLong(%d): round trip failed, got %d", v, got)
		}
	}
}

func BenchmarkVarIntEncode(b *testing.B) {
	buf := make([]byte, MaxVarIntLen)
	for i := 0; i < b.N; i++ {
		PutVarInt(buf, int32(i))
	}
}

func BenchmarkVarIntDecode(b *testing.B) {
	buf := make([]byte, MaxVarIntLen)
	PutVarInt(buf, 268435455)
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(buf)
		ReadVarInt(r)
	}
}

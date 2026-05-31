package packet

import (
	"bytes"
	"math"
	"testing"
)

func TestStringRoundTrip(t *testing.T) {
	s := String("hello, minecraft!")

	var buf bytes.Buffer
	_, err := s.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	var got String
	if err := got.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}

	if got != s {
		t.Errorf("String round trip: got %q, want %q", got, s)
	}
}

func TestByteRoundTrip(t *testing.T) {
	b := Byte(-128)
	var buf bytes.Buffer
	b.WriteTo(&buf)

	var got Byte
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != b {
		t.Errorf("Byte round trip: got %d, want %d", got, b)
	}
}

func TestUnsignedByteRoundTrip(t *testing.T) {
	ub := UnsignedByte(255)
	var buf bytes.Buffer
	ub.WriteTo(&buf)

	var got UnsignedByte
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != ub {
		t.Errorf("UnsignedByte round trip: got %d, want %d", got, ub)
	}
}

func TestShortRoundTrip(t *testing.T) {
	s := Short(-32768)
	var buf bytes.Buffer
	s.WriteTo(&buf)

	var got Short
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != s {
		t.Errorf("Short round trip: got %d, want %d", got, s)
	}
}

func TestUnsignedShortRoundTrip(t *testing.T) {
	us := UnsignedShort(65535)
	var buf bytes.Buffer
	us.WriteTo(&buf)

	var got UnsignedShort
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != us {
		t.Errorf("UnsignedShort round trip: got %d, want %d", got, us)
	}
}

func TestIntRoundTrip(t *testing.T) {
	i := Int(math.MaxInt32)
	var buf bytes.Buffer
	i.WriteTo(&buf)

	var got Int
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != i {
		t.Errorf("Int round trip: got %d, want %d", got, i)
	}
}

func TestLongRoundTrip(t *testing.T) {
	l := Long(math.MaxInt64)
	var buf bytes.Buffer
	l.WriteTo(&buf)

	var got Long
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != l {
		t.Errorf("Long round trip: got %d, want %d", got, l)
	}
}

func TestFloatRoundTrip(t *testing.T) {
	f := Float(3.14159)
	var buf bytes.Buffer
	f.WriteTo(&buf)

	var got Float
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if math.Abs(float64(got)-float64(f)) > 0.0001 {
		t.Errorf("Float round trip: got %f, want %f", got, f)
	}
}

func TestDoubleRoundTrip(t *testing.T) {
	d := Double(math.Pi)
	var buf bytes.Buffer
	d.WriteTo(&buf)

	var got Double
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if math.Abs(float64(got)-float64(d)) > 0.0001 {
		t.Errorf("Double round trip: got %f, want %f", got, d)
	}
}

func TestBooleanRoundTrip(t *testing.T) {
	for _, val := range []Boolean{true, false} {
		var buf bytes.Buffer
		val.WriteTo(&buf)
		var got Boolean
		got.ReadFrom(bytes.NewReader(buf.Bytes()))
		if got != val {
			t.Errorf("Boolean round trip: got %v, want %v", got, val)
		}
	}
}

func TestPositionRoundTrip(t *testing.T) {
	pos := Position{X: 100, Y: 64, Z: -200}
	var buf bytes.Buffer
	pos.WriteTo(&buf)

	var got Position
	got.ReadFrom(bytes.NewReader(buf.Bytes()))

	if got.X != pos.X || got.Y != pos.Y || got.Z != pos.Z {
		t.Errorf("Position round trip: got (%d,%d,%d), want (%d,%d,%d)",
			got.X, got.Y, got.Z, pos.X, pos.Y, pos.Z)
	}
}

func TestPositionMaxValues(t *testing.T) {
	pos := Position{X: 33554431, Y: 2047, Z: 33554431}
	var buf bytes.Buffer
	pos.WriteTo(&buf)

	var got Position
	got.ReadFrom(bytes.NewReader(buf.Bytes()))

	if got.X != pos.X || got.Y != pos.Y || got.Z != pos.Z {
		t.Errorf("Position max round trip: got (%d,%d,%d), want (%d,%d,%d)",
			got.X, got.Y, got.Z, pos.X, pos.Y, pos.Z)
	}
}

func TestUUIDRoundTrip(t *testing.T) {
	hi, lo := int64(0x0123456789ABCDEF), int64(0x7EDCBA9876543210)
	u := UUID(IntsToUUID(hi, lo))
	var buf bytes.Buffer
	u.WriteTo(&buf)

	var got UUID
	got.ReadFrom(bytes.NewReader(buf.Bytes()))

	gotHi, gotLo := UuidToInts(got)
	if gotHi != hi || gotLo != lo {
		t.Errorf("UUID round trip: got (0x%X, 0x%X), want (0x%X, 0x%X)", gotHi, gotLo, hi, lo)
	}
}

func TestSlotRoundTrip(t *testing.T) {
	slot := Slot{
		Present: true,
		ItemID:  42,
		Count:   10,
	}

	var buf bytes.Buffer
	slot.WriteTo(&buf)

	var got Slot
	got.ReadFrom(bytes.NewReader(buf.Bytes()))

	if got.Present != slot.Present || got.ItemID != slot.ItemID || got.Count != slot.Count {
		t.Errorf("Slot round trip: got {%v,%d,%d}, want {%v,%d,%d}",
			got.Present, got.ItemID, got.Count, slot.Present, slot.ItemID, slot.Count)
	}
}

func TestSlotEmpty(t *testing.T) {
	slot := Slot{Present: false}
	var buf bytes.Buffer
	slot.WriteTo(&buf)

	var got Slot
	got.ReadFrom(bytes.NewReader(buf.Bytes()))

	if got.Present {
		t.Error("empty slot should have Present=false")
	}
}

func TestAngleRoundTrip(t *testing.T) {
	a := Angle(128)
	var buf bytes.Buffer
	a.WriteTo(&buf)

	var got Angle
	got.ReadFrom(bytes.NewReader(buf.Bytes()))
	if got != a {
		t.Errorf("Angle round trip: got %d, want %d", got, a)
	}
}

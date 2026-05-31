package compress

import (
	"bytes"
	"testing"
)

func TestCompressDecompress(t *testing.T) {
	data := []byte("Hello, Minecraft! This is a test of zlib compression.")

	compressed, err := Compress(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(compressed) >= len(data) {
		t.Logf("compressed size %d >= original %d (expected for small data)", len(compressed), len(data))
	}

	decompressed, err := Decompress(compressed, len(data)*2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Errorf("compress/decompress round trip failed:\n  got:  %q\n  want: %q", decompressed, data)
	}
}

func TestCompressLargeData(t *testing.T) {
	data := make([]byte, 65536)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}

	compressed, err := Compress(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(compressed) >= len(data) {
		t.Logf("compressed size %d >= original %d", len(compressed), len(data))
	}

	decompressed, err := Decompress(compressed, len(data)*2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("large data compress/decompress round trip failed")
	}
}

func TestCompressEmpty(t *testing.T) {
	compressed, err := Compress([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := Decompress(compressed, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(decompressed) != 0 {
		t.Errorf("empty data decompressed to %d bytes, want 0", len(decompressed))
	}
}

func TestCompressBuffer(t *testing.T) {
	data := []byte("buffer based compression test")
	var buf bytes.Buffer

	err := CompressBuffer(data, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if buf.Len() == 0 {
		t.Fatal("compressed buffer is empty")
	}

	var decompressed bytes.Buffer
	err = DecompressBuffer(buf.Bytes(), &decompressed)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, decompressed.Bytes()) {
		t.Errorf("CompressBuffer/DecompressBuffer round trip failed")
	}
}

func TestThreshold(t *testing.T) {
	tm := NewThresholdManager(256)
	if !tm.IsEnabled() {
		t.Error("threshold should be enabled")
	}
	if tm.Threshold() != 256 {
		t.Errorf("threshold = %d, want 256", tm.Threshold())
	}

	if tm.ShouldCompress(100) {
		t.Error("100 should not be compressed with threshold 256")
	}
	if !tm.ShouldCompress(300) {
		t.Error("300 should be compressed with threshold 256")
	}

	tm.SetThreshold(-1)
	if tm.IsEnabled() {
		t.Error("threshold -1 should disable compression")
	}
}

func BenchmarkCompress(b *testing.B) {
	data := make([]byte, 8192)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compress(data)
	}
}

func BenchmarkDecompress(b *testing.B) {
	data := make([]byte, 8192)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}
	compressed, _ := Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decompress(compressed, len(data)*2)
	}
}

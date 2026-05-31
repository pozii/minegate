package compress

import (
	"bytes"
	"compress/zlib"
	"io"
	"sync"

	kzlib "github.com/klauspost/compress/zlib"
	"github.com/user/minegate/internal"
)

var (
	writerPool sync.Pool
	readerPool sync.Pool
)

func getWriter(w io.Writer) (*kzlib.Writer, error) {
	if v := writerPool.Get(); v != nil {
		zw := v.(*kzlib.Writer)
		zw.Reset(w)
		return zw, nil
	}
	return kzlib.NewWriterLevel(w, kzlib.DefaultCompression)
}

func putWriter(zw *kzlib.Writer) {
	writerPool.Put(zw)
}

func getReader(r io.Reader) (io.ReadCloser, error) {
	if v := readerPool.Get(); v != nil {
		zr := v.(io.ReadCloser)
		if rst, ok := zr.(zlib.Resetter); ok {
			if err := rst.Reset(r, nil); err != nil {
				return nil, err
			}
			return zr, nil
		}
	}
	return zlib.NewReader(r)
}

func putReader(zr io.ReadCloser) {
	zr.Close()
	readerPool.Put(zr)
}

// Compress compresses the given data using zlib.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := getWriter(&buf)
	if err != nil {
		return nil, err
	}
	defer putWriter(w)

	if _, err := w.Write(data); err != nil {
		return nil, internal.ErrCompressionFailed
	}
	if err := w.Close(); err != nil {
		return nil, internal.ErrCompressionFailed
	}

	return buf.Bytes(), nil
}

// Decompress decompresses the given zlib-compressed data.
func Decompress(data []byte, maxSize int) ([]byte, error) {
	r, err := getReader(bytes.NewReader(data))
	if err != nil {
		return nil, internal.ErrCompressionFailed
	}
	defer putReader(r)

	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, r, int64(maxSize)); err != nil && err != io.EOF {
		return nil, internal.ErrCompressionFailed
	}

	return buf.Bytes(), nil
}

// CompressBuffer is like Compress but takes a destination buffer as parameter.
func CompressBuffer(src []byte, dst *bytes.Buffer) error {
	w, err := getWriter(dst)
	if err != nil {
		return err
	}
	defer putWriter(w)

	if _, err := w.Write(src); err != nil {
		return internal.ErrCompressionFailed
	}
	if err := w.Close(); err != nil {
		return internal.ErrCompressionFailed
	}

	return nil
}

// DecompressBuffer is like Decompress but takes a destination buffer as parameter.
func DecompressBuffer(src []byte, dst *bytes.Buffer) error {
	r, err := getReader(bytes.NewReader(src))
	if err != nil {
		return internal.ErrCompressionFailed
	}
	defer putReader(r)

	if _, err := io.Copy(dst, r); err != nil {
		return internal.ErrCompressionFailed
	}

	return nil
}

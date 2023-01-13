package simple

import (
	"encoding/binary"
	"fmt"

	"github.com/pierrec/lz4/v4"
	"github.com/sirupsen/logrus"
)

type Compressor struct{}

func NewCompressor() *Compressor {
	return &Compressor{}
}

const sizeBytesLen = 8

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data))+sizeBytesLen)
	n, err := lz4.CompressBlock(data, buf[sizeBytesLen:], nil)
	if err != nil {
		return nil, fmt.Errorf("failed to compress the passed data: %w", err)
	}
	sizeToStore := len(data)
	if n == 0 {
		logrus.WithError(err).Warn("failed to compress data")
		n = copy(buf[sizeBytesLen:], data)
		sizeToStore = 0
	}
	storeSizeSlice(buf[:sizeBytesLen], uint64(sizeToStore))
	return buf[:(n + sizeBytesLen)], nil
}

func (c *Compressor) Decompress(compressedData []byte) (pt []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	dataLen := readSizeFromSlice(compressedData[:sizeBytesLen])
	if dataLen == 0 {
		return compressedData[sizeBytesLen:], nil
	}
	data := make([]byte, dataLen)
	if _, err := lz4.UncompressBlock(compressedData[sizeBytesLen:], data); err != nil {
		return nil, fmt.Errorf("failed to uncompress the data block: %w", err)
	}
	return data, nil
}

func storeSizeSlice(dst []byte, size uint64) {
	binary.LittleEndian.PutUint64(dst, size)
}

func readSizeFromSlice(sizeSlice []byte) uint64 {
	return binary.LittleEndian.Uint64(sizeSlice)
}

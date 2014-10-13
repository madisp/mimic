package h264

import (
	"bufio"
	"bytes"
	"io"
)

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// check for eof
	if atEOF {
		if len(data) == 0 {
			return 0, nil, nil
		}
		// last token
		return len(data), data, nil
	}
	// skip leading zeroes
	prefixByteCount := 0
	start := 0
	for ; start+4 < len(data); start++ {
		if bytes.Equal(data[start:3], []byte{0x00, 0x00, 0x01}) {
			prefixByteCount = 3
			break
		} else if bytes.Equal(data[start:4], []byte{0x00, 0x00, 0x00, 0x01}) {
			prefixByteCount = 4
			break
		}
	}

	if prefixByteCount > 0 {
		// look for end
		for i := start + prefixByteCount; i+4 < len(data); i++ {
			if bytes.Equal(data[i:i+3], []byte{0x00, 0x00, 0x01}) {
				return i, data[0:i], nil
			} else if bytes.Equal(data[i:i+4], []byte{0x00, 0x00, 0x00, 0x01}) {
				return i, data[0:i], nil
			}
		}
	}

	// moar data!
	return 0, nil, nil
}

func wrapSplit(buf []byte, atEOF bool, f func([]byte) error) ([]byte, error) {
	if advance, unit, err := split(buf, atEOF); err != nil {
		return nil, err
	} else {
		if unit != nil {
			if err := f(unit); err != nil {
				return nil, err
			}
		}
		if advance > 0 {
			return buf[advance:len(buf)], nil
		}
	}
	return buf, nil
}

/* takes a reader and emits h264 NAL units in a channel */
func Scan(r io.Reader, f func([]byte) error) (err error) {
	reader := bufio.NewReaderSize(r, 4096)
	buf := make([]byte, 4096)
	token := make([]byte, 0)
	for err == nil {
		var n int
		if n, err = reader.Read(buf); err == nil {
			token = append(token, buf[0:n]...)
			if token, err = wrapSplit(token, false, f); err != nil {
				return err
			}
		}
	}
	if err == io.EOF {
		_, err = wrapSplit(token, true, f)
	}
	return err
}

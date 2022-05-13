package diff

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// https://stackoverflow.com/a/64498218

type compareReader struct {
	a    io.Reader
	b    io.Reader
	bBuf []byte
}

func NewCompareReader(a, b io.Reader) io.Reader {
	return &compareReader{
		a: a,
		b: b,
	}
}

func min(a, b int) int {
	if a > b {
		return b
	}

	return a
}

func (c *compareReader) Read(p []byte) (int, error) {
	if c.bBuf == nil {
		// assuming p's len() stays the same, so we can optimize for both of their buffer
		// sizes to be equal
		c.bBuf = make([]byte, len(p))
	}

	// read only as much data as we can fit in both p and bBuf
	readA, errA := c.a.Read(p[0:min(len(p), len(c.bBuf))])
	if readA > 0 {
		// bBuf is guaranteed to have at least readA space
		if _, errB := io.ReadFull(c.b, c.bBuf[0:readA]); errB != nil { // docs: "EOF only if no bytes were read"
			if errB == io.ErrUnexpectedEOF {
				return readA, errors.New("compareReader: A had more data than B")
			} else {
				return readA, fmt.Errorf("compareReader: read error from B: %w", errB)
			}
		}

		if !bytes.Equal(p[0:readA], c.bBuf[0:readA]) {
			return readA, errors.New("compareReader: bytes not equal")
		}
	}
	if errA == io.EOF {
		// in happy case expecting EOF from B as well. might be extraneous call b/c we might've
		// got it already from the for loop above, but it's easier to check here
		readB, errB := c.b.Read(c.bBuf)
		if readB > 0 {
			return readA, errors.New("compareReader: B had more data than A")
		}

		if errB != io.EOF {
			return readA, fmt.Errorf("compareReader: got EOF from A but not from B: %w", errB)
		}
	}

	return readA, errA
}

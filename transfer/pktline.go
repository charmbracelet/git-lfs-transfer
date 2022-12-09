package transfer

import (
	"fmt"
	"io"

	"github.com/git-lfs/pktline"
)

const (
	// Flush is the flush packet.
	Flush = '\x00'
	// Delim is the delimiter packet.
	Delim = '\x01'
)

// List of Git LFS commands.
const (
	versionCommand      = "version"
	batchCommand        = "batch"
	putObjectCommand    = "put-object"
	verifyObjectCommand = "verify-object"
	getObjectCommand    = "get-object"
	lockCommand         = "lock"
	listLockCommand     = "list-lock"
	unlockCommand       = "unlock"
	quitCommand         = "quit"
)

// PktLine is a Git packet line handler.
type Pktline struct {
	*pktline.Pktline
}

// NewPktline creates a new Git packet line handler.
func NewPktline(r io.Reader, w io.Writer) *Pktline {
	return &Pktline{pktline.NewPktline(r, w)}
}

// SendError sends an error msg.
func (p *Pktline) SendError(status uint32, message string) error {
	if err := p.WritePacketText(fmt.Sprintf("status: %03d\n", status)); err != nil {
		return err
	}
	if err := p.WriteDelim(); err != nil {
		return err
	}
	if err := p.WritePacketText(message); err != nil {
		return err
	}
	return p.WriteFlush()
}

// SendStatus sends a status message.
func (p *Pktline) SendStatus(status Status) error {
	if err := p.WritePacketText(fmt.Sprintf("status: %03d\n", status.Code())); err != nil {
		return err
	}
	if args := status.Args(); len(args) > 0 {
		for _, arg := range args {
			if err := p.WritePacketText(arg); err != nil {
				return err
			}
		}
	}
	if msgs := status.Messages(); len(msgs) > 0 {
		if err := p.WriteDelim(); err != nil {
			return err
		}
		for _, msg := range msgs {
			if err := p.WritePacketText(msg); err != nil {
				return err
			}
		}
	} else if r := status.Reader(); r != nil {
		// Close reader if it implements io.Closer.
		if c, ok := r.(io.Closer); ok {
			defer c.Close()
		}
		if err := p.WriteDelim(); err != nil {
			return err
		}
		w := pktline.NewPktlineWriterFromPktline(p.Pktline, 0)
		if _, err := io.Copy(w, r); err != nil {
			return err
		}

	}
	return p.WriteFlush()
}

// Reader returns a reader for the packet line.
func (p *Pktline) Reader() io.Reader {
	return p.ReaderWithSize(0)
}

// ReaderWithSize returns a reader for the packet line with the given size.
func (p *Pktline) ReaderWithSize(size int) io.Reader {
	return pktline.NewPktlineReaderFromPktline(p.Pktline, size)
}

// Writer returns a writer for the packet line.
func (p *Pktline) Writer() io.Writer {
	return p.WriterWithSize(0)
}

// WriterWithSize returns a writer for the packet line with the given size.
func (p *Pktline) WriterWithSize(size int) io.Writer {
	return pktline.NewPktlineWriterFromPktline(p.Pktline, size)
}

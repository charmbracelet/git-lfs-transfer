package transfer

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// Processor is a transfer processor.
type Processor struct {
	handler *Pktline
	backend Backend
}

// NewProcessor creates a new transfer processor.
func NewProcessor(line *Pktline, backend Backend) *Processor {
	return &Processor{
		handler: line,
		backend: backend,
	}
}

// Version returns the version of the transfer protocol.
func (p *Processor) Version() (Status, error) {
	_, err := p.handler.ReadPacketText()
	if err != nil {
		return nil, err
	}
	return NewSuccessStatus([]string{}), nil
}

// Error returns a transfer protocol error.
func (p *Processor) Error(code uint32, message string) (Status, error) {
	return NewFailureStatus(code, message), nil
}

// ReadBatch reads a batch request.
func (p *Processor) ReadBatch(op Operation) ([]BatchItem, error) {
	ar, err := p.handler.ReadPacketListToDelim()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	args, err := ParseArgs(ar)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	data, err := p.handler.ReadPacketListToFlush()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	hashAlgo := args[HashAlgoKey]
	switch hashAlgo {
	case "", "sha256":
	default:
		return nil, fmt.Errorf("%w: %s", ErrNotAllowed, fmt.Sprintf("unsupported hash algorithm: %s", hashAlgo))
	}
	Logf("data: %d %v", len(data), data)
	Logf("batch: %s args: %d %v data: %d %v", op, len(args), args, len(data), data)
	items := make([]OidWithSize, 0)
	for _, line := range data {
		if line == "" {
			return nil, ErrInvalidPacket
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 || parts[1] == "" {
			return nil, ErrParseError
		}
		size, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid integer, got: %q", ErrParseError, parts[1])
		}
		item := OidWithSize{
			Oid(parts[0]),
			int64(size),
		}
		items = append(items, item)
	}
	Logf("items %v", items)
	its, err := p.backend.Batch(op, items)
	if err != nil {
		return nil, err
	}
	Logf("batch items: %v", its)
	return its, nil
}

// BatchData writes batch data to the transfer protocol.
func (p *Processor) BatchData(op Operation, presentAction string, missingAction string) (Status, error) {
	batch, err := p.ReadBatch(op)
	if err != nil {
		return p.Error(StatusBadRequest, err.Error())
	}
	oids := make([]string, 0)
	for _, item := range batch {
		action := missingAction
		if item.Present {
			action = presentAction
		}
		oids = append(oids, fmt.Sprintf("%s %d %s", item.Oid, item.Size, action))
	}
	return NewSuccessStatus(oids), nil
}

// UploadBatch writes upload data to the transfer protocol.
func (p *Processor) UploadBatch() (Status, error) {
	return p.BatchData(UploadOperation, "noop", "upload")
}

// DownloadBatch writes download data to the transfer protocol.
func (p *Processor) DownloadBatch() (Status, error) {
	return p.BatchData(DownloadOperation, "download", "noop")
}

// SizeFromArgs returns the size from the given args.
func (Processor) SizeFromArgs(args map[string]string) (int64, error) {
	size, ok := args[SizeKey]
	if !ok {
		return 0, fmt.Errorf("missing required size header")
	}
	n, err := strconv.Atoi(size)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %w", err)
	}
	return int64(n), nil
}

// PutObject writes an object ID to the transfer protocol.
func (p *Processor) PutObject(oid Oid) (Status, error) {
	ar, err := p.handler.ReadPacketListToDelim()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	args, err := ParseArgs(ar)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	expectedSize, err := p.SizeFromArgs(args)
	if err != nil {
		return nil, err
	}
	r := p.handler.ReaderWithSize(int(expectedSize))
	rdr := NewHashingReader(r, sha256.New())
	state, err := p.backend.StartUpload(oid, rdr)
	if err != nil {
		return nil, err
	}
	actualSize := rdr.Size()
	if actualSize != expectedSize {
		err := fmt.Errorf("invalid size, expected %d, got %d", expectedSize, actualSize)
		if actualSize > expectedSize {
			err = fmt.Errorf("%w: %s", ErrExtraData, err)
		} else {
			err = fmt.Errorf("%w: %s", ErrMissingData, err)
		}
		return nil, err
	}
	actualOid, err := rdr.Oid()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCorruptData, err)
	}
	if actualOid != oid {
		return nil, fmt.Errorf("%w: %s", ErrCorruptData, fmt.Sprintf("invalid object ID, expected %s, got %s", oid, actualOid))
	}
	if err := p.backend.FinishUpload(state); err != nil {
		return nil, err
	}
	return SuccessStatus(), nil
}

// VerifyObject verifies an object ID.
func (p *Processor) VerifyObject(oid Oid) (Status, error) {
	ar, err := p.handler.ReadPacketListToFlush()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	args, err := ParseArgs(ar)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	return p.backend.Verify(oid, args)
}

// GetObject writes an object ID to the transfer protocol.
func (p *Processor) GetObject(oid Oid) (Status, error) {
	ar, err := p.handler.ReadPacketListToFlush()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	args, err := ParseArgs(ar)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	r, err := p.backend.Download(oid, ArgsToList(args)...)
	if errors.Is(err, fs.ErrNotExist) {
		return NewFailureStatus(StatusNotFound, fmt.Sprintf("object %s not found", oid)), nil
	}
	if err != nil {
		return nil, err
	}
	return NewSuccessStatusWithReader(r, fmt.Sprintf("size=%d", r.Size)), nil
}

// Lock writes a lock to the transfer protocol.
func (p *Processor) Lock() (Status, error) {
	data, err := p.handler.ReadPacketListToFlush()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	args, err := ParseArgs(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	path := args[PathKey]
	// refname := args[RefnameKey]
	if path == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingData, "path and refname are required")
	}
	lockBackend := p.backend.LockBackend()
	retried := false
	for !retried {
		lock, err := lockBackend.Create(path)
		if errors.Is(err, ErrConflict) {
			if lock == nil {
				retried = true
				continue
			}
			return NewFailureStatusWithArgs(StatusConflict, err.Error(), lock.AsArguments()...), nil
		}
		if err != nil {
			return nil, err
		}
		return NewSuccessStatusWithCode(StatusCreated, lock.AsArguments()...), nil
	}
	// unreachable
	panic("unreachable")
}

// ListLocksForPath lists locks for a path. cursor can be empty.
func (p *Processor) ListLocksForPath(path string, cursor string, useOwnerID bool) (Status, error) {
	lock, err := p.backend.LockBackend().FromPath(path)
	if err != nil {
		return nil, err
	}
	if (lock == nil && cursor == "") ||
		(lock.ID() < cursor) {
		return p.Error(StatusNotFound, fmt.Sprintf("lock for path %s not found", path))
	}
	spec, err := lock.AsLockSpec(useOwnerID)
	if err != nil {
		return nil, err
	}
	return NewSuccessStatus(spec), nil
}

// ListLocks lists locks.
func (p *Processor) ListLocks(useOwnerID bool) (Status, error) {
	ar, err := p.handler.ReadPacketListToFlush()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	args, err := ParseArgs(ar)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseError, err)
	}
	limit, err := strconv.Atoi(args[LimitKey])
	if err != nil {
		return NewSuccessStatusWithCode(StatusContinue), nil
	}
	if limit == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotAllowed, "request has no limit")
	} else if limit > 100 {
		// Try to avoid DoS attacks.
		limit = 100
	}
	cursor := args[CursorKey]
	path := args[PathKey]
	if path != "" {
		return p.ListLocksForPath(path, cursor, useOwnerID)
	}
	locks := make([]Lock, 0)
	lb := p.backend.LockBackend()
	lb.Range(func(lock Lock) error {
		if len(locks) >= limit+1 {
			// stop iterating when limit is reached.
			return io.EOF
		}
		if lock == nil {
			// skip nil locks
			return nil
		}
		if lock.ID() < cursor {
			return nil
		}
		locks = append(locks, lock)
		return nil
	})
	msgs := make([]string, 0, len(locks))
	for _, item := range locks {
		specs, err := item.AsLockSpec(useOwnerID)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, specs...)
	}
	nextCursor := ""
	if len(locks) == limit+1 {
		nextCursor = fmt.Sprintf("next-cursor=%s", locks[limit].ID())
	}
	return NewSuccessStatusWithData(StatusAccepted, msgs, nextCursor), nil
}

// Unlock unlocks a lock.
func (p *Processor) Unlock(id string) (Status, error) {
	_, _ = p.handler.ReadPacket()
	lock, err := p.backend.LockBackend().FromID(id)
	if err != nil {
		return nil, err
	}
	if lock == nil {
		return p.Error(StatusNotFound, fmt.Sprintf("lock %s not found", id))
	}
	if err := lock.Unlock(); err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			return p.Error(StatusNotFound, fmt.Sprintf("lock %s not found", id))
		case errors.Is(err, os.ErrPermission):
			return p.Error(StatusForbidden, fmt.Sprintf("lock %s not owned by you", id))
		default:
			return nil, err
		}
	}
	return NewSuccessStatusWithCode(StatusOK, lock.AsArguments()...), nil
}

// ProcessCommands processes commands from the transfer protocol.
func (p *Processor) ProcessCommands(op Operation) error {
	Log("processing commands")
	for {
		pkt, err := p.handler.ReadPacketText()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		Logf("received packet: %s", pkt)
		if pkt == "" {
			if err := p.handler.SendError(StatusBadRequest, "unknown command"); err != nil {
				Logf("error pktline sending error: %v", err)
			}
			continue
		}
		msgs := strings.Split(pkt, " ")
		if len(msgs) < 1 {
			if err := p.handler.SendError(StatusBadRequest, "no command provided"); err != nil {
				Logf("error pktline sending error: %v", err)
			}
			continue
		}
		Logf("received command: %s %v", msgs[0], msgs[1:])
		var status Status
		switch msgs[0] {
		case versionCommand:
			if len(msgs) > 0 && msgs[1] == "1" {
				status, err = p.Version()
			} else {
				err = p.handler.SendError(StatusBadRequest, "unknown version")
			}
		case batchCommand:
			switch op {
			case UploadOperation:
				Logf("upload batch command received")
				status, err = p.UploadBatch()
			case DownloadOperation:
				Logf("download batch command received")
				status, err = p.DownloadBatch()
			}
		case putObjectCommand:
			if len(msgs) > 1 {
				status, err = p.PutObject(Oid(msgs[1]))
			} else {
				err = p.handler.SendError(StatusForbidden, "not allowed")
			}
		case verifyObjectCommand:
			if len(msgs) > 1 {
				status, err = p.VerifyObject(Oid(msgs[1]))
			} else {
				err = p.handler.SendError(StatusForbidden, "not allowed")
			}
		case getObjectCommand:
			if len(msgs) > 1 {
				status, err = p.GetObject(Oid(msgs[1]))
			} else {
				err = p.handler.SendError(StatusForbidden, "not allowed")
			}
		case lockCommand:
			status, err = p.Lock()
		case listLockCommand, "list-locks":
			switch op {
			case UploadOperation:
				status, err = p.ListLocks(true)
			case DownloadOperation:
				status, err = p.ListLocks(false)
			}
			Logf("list lock status: %v %v", status, err)
		case unlockCommand:
			if len(msgs) > 1 {
				status, err = p.Unlock(msgs[1])
			} else {
				err = p.handler.SendError(StatusBadRequest, "unknown command")
			}
		case quitCommand:
			p.handler.SendStatus(SuccessStatus())
			return nil
		default:
			err = p.handler.SendError(StatusBadRequest, "unknown command")
		}
		if err != nil {
			switch {
			case errors.Is(err, ErrExtraData),
				errors.Is(err, ErrNotAllowed),
				errors.Is(err, ErrInvalidPacket),
				errors.Is(err, ErrCorruptData):
				if err := p.handler.SendError(StatusBadRequest, fmt.Errorf("error: %w", err).Error()); err != nil {
					Logf("error pktline sending error: %v", err)
				}
			default:
				Logf("error processing command: %v", err)
				if err := p.handler.SendError(StatusInternalServerError, "internal error"); err != nil {
					Logf("error pktline sending error: %v", err)
				}
			}
		}
		if status != nil {
			if err := p.handler.SendStatus(status); err != nil {
				Logf("error pktline sending status: %v", err)
			}
		}
	}
}

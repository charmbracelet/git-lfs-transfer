package transfer

// BatchItem is a Git LFS batch item.
type BatchItem struct {
	Pointer

	// Present is used to determine the action to take for the batch item.
	Present bool

	// Args is an optional oid-line key-value pairs.
	Args Args
<<<<<<< HEAD
=======

	// Error is an optional error message.
	Error string
>>>>>>> 29d6979f6467 (feat: support batch response error message)
}

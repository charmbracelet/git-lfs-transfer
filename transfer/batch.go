package transfer

// BatchItem is a Git LFS batch item.
type BatchItem struct {
	Pointer
	Args    Args
	Present bool
}

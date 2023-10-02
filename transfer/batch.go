package transfer

// BatchItemArg is a Git LFS batch item argument.
type BatchItemArg struct {
	Key   string
	Value string
}

// BatchItem is a Git LFS batch item.
type BatchItem struct {
	Pointer
	Present bool
	Args    []BatchItemArg
}

package main

// BatchItem is a Git LFS batch item.
type BatchItem struct {
	Oid     Oid
	Size    int64
	Present bool
}

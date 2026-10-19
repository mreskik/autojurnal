package types

import (
	"bytes"
	"encoding/json"
)

// Nullable membedakan field JSON yang tidak dikirim, dikirim null, dan dikirim berisi nilai.
// Dipakai di request update sebagian (PATCH-like) untuk field yang boleh dikosongkan.
type Nullable[T any] struct {
	Set   bool // field ada di JSON
	Value *T   // nil kalau dikirim null
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	if bytes.Equal(data, []byte("null")) {
		n.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

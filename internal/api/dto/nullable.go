package dto

import "encoding/json"

// NullableField represents a JSON field with three states:
//  1. Absent — field was not present in JSON (Set=false)
//  2. Explicit null — field was present and set to null (Set=true, Null=true)
//  3. Has value — field was present with a non-null value (Set=true, Null=false)
//
// Use this type for PATCH DTO fields that correspond to nullable DB columns,
// so the handler can distinguish "don't change" from "clear to NULL".
type NullableField[T any] struct {
	Value T
	Set   bool
	Null  bool
}

func (n *NullableField[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Null = true
		return nil
	}
	return json.Unmarshal(data, &n.Value)
}

func (n NullableField[T]) MarshalJSON() ([]byte, error) {
	if !n.Set || n.Null {
		return []byte("null"), nil
	}
	return json.Marshal(n.Value)
}

// Ptr returns a pointer to the value if set and not null, nil otherwise.
// Convenient for assigning to domain model pointer fields:
//
//	if req.Description.Set {
//	    existing.Description = req.Description.Ptr()
//	}
func (n NullableField[T]) Ptr() *T {
	if n.Set && !n.Null {
		v := n.Value
		return &v
	}
	return nil
}

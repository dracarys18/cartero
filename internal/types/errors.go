package types

import (
	"fmt"
)

type FilteredError struct {
	ProcessorName string
	ItemID        string
	Reason        string
	Details       map[string]interface{}
}

func (e *FilteredError) Error() string {
	return fmt.Sprintf("filtered by %s: %s (item: %s)", e.ProcessorName, e.Reason, e.ItemID)
}

func IsFiltered(err error) bool {
	_, ok := err.(*FilteredError)
	return ok
}

func NewFilteredError(processorName, itemID, reason string) *FilteredError {
	return &FilteredError{
		ProcessorName: processorName,
		ItemID:        itemID,
		Reason:        reason,
		Details:       make(map[string]interface{}),
	}
}

func (e *FilteredError) WithDetail(key string, value interface{}) *FilteredError {
	e.Details[key] = value
	return e
}

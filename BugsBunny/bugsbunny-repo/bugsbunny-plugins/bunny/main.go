package main

import (
    "unsafe"
)

func main() {}

//export malloc
func malloc(size uint32) *byte {
    buf := make([]byte, size)
    return &buf[0]
}

// THE WORKFLOW GRAPH
// Key = Current Status | Values = Allowed Next Statuses
var workflow = map[string][]string{
    "open":        {"accepted", "rejected", "closed", "deferred"},
    "accepted":    {"in_progress", "on_hold"},
    "in_progress": {"resolved", "on_hold"},
    "on_hold":     {"in_progress", "closed", "deferred"},
    "deferred":    {"open", "closed"},
    "resolved":    {"closed", "in_progress"}, // Can move back to in_progress if QA fails
    "rejected":    {"closed"},
    "closed":      {}, // Terminal state - an empty list means no escape!
}

//export can_transition
func can_transition(currPtr, currSize, newPtr, newSize uint32) uint32 {
    currentStatus := ptrToString(currPtr, currSize)
    newStatus := ptrToString(newPtr, newSize)

    // 1. Look up the current status in our workflow graph
    allowedNextStates, exists := workflow[currentStatus]
    if !exists {
        return 0 // Unknown starting state, reject
    }

    // 2. Check if the new status is in the allowed list
    for _, allowedState := range allowedNextStates {
        if newStatus == allowedState {
            return 1 // 1 = True (Approved)
        }
    }

    // 3. If not found in the allowed list, reject
    return 0 
}

// Helper to read memory safely
func ptrToString(ptr uint32, size uint32) string {
    b := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
    return string(b)
}

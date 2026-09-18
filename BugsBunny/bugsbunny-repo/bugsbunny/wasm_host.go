package main

import (
    "context"
    "fmt"
    "os"

    "github.com/tetratelabs/wazero"
    "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// ValidateTransition boots a WASM sandbox and asks it if a status change is allowed.
func ValidateTransition(projectKey, currentStatus, newStatus string) (bool, error) {
    // 1. If no plugin exists for this project, allow the transition by default
    wasmPath := fmt.Sprintf("plugins/%s.wasm", projectKey)
    wasmBytes, err := os.ReadFile(wasmPath)
    if err != nil {
        return true, nil 
    }

    ctx := context.Background()
    r := wazero.NewRuntime(ctx)
    defer r.Close(ctx)
    wasi_snapshot_preview1.MustInstantiate(ctx, r)

    // 2. Instantiate the WASM module
    mod, err := r.Instantiate(ctx, wasmBytes)
    if err != nil {
        return false, fmt.Errorf("failed to instantiate WASM: %v", err)
    }

    // 3. Get the exported memory and functions
    malloc := mod.ExportedFunction("malloc")
    canTransition := mod.ExportedFunction("can_transition")
    if malloc == nil || canTransition == nil {
        return false, fmt.Errorf("missing required WASM functions")
    }

    // 4. Helper function to write strings into WASM memory
    writeString := func(s string) (uint64, uint64, error) {
        size := uint64(len(s))
        results, err := malloc.Call(ctx, size)
        if err != nil {
            return 0, 0, err
        }
        ptr := results[0]
        if !mod.Memory().Write(uint32(ptr), []byte(s)) {
            return 0, 0, fmt.Errorf("failed to write to WASM memory")
        }
        return ptr, size, nil
    }

    // 5. Write both status strings to the sandbox
    currPtr, currSize, err := writeString(currentStatus)
    if err != nil { return false, err }
    
    newPtr, newSize, err := writeString(newStatus)
    if err != nil { return false, err }

    // 6. Execute the business logic!
    results, err := canTransition.Call(ctx, currPtr, currSize, newPtr, newSize)
    if err != nil { return false, err }

    // Returns 1 if approved, 0 if rejected
    return results[0] == 1, nil
}

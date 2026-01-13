# SUB-PHASE IMPLEMENTATION TEMPLATE
## Reusable Template for AI Agent Instructions

---

## HOW TO USE THIS TEMPLATE

1. **Copy this entire template** for each sub-phase
2. **Fill in all [BRACKETED] sections** with specific details
3. **Keep the structure intact** - the agent expects this format
4. **Provide to the AI agent** as a complete, self-contained prompt
5. **Wait for 100% completion** before moving to next sub-phase

---

## TEMPLATE START

---

### PHASE [1.1.3]: [SUB-PHASE NAME]

**Phase Context**: hase 1 builds the communication foundation between browser and backend. By the end, you'll have a working WebSocket server that can exchange structured binary messages with a JavaScript client.

**Sub-Phase Objective**: [One sentence describing what this sub-phase accomplishes]

**Prerequisites**: 
- [List all previous sub-phases that must be complete]
- [Include specific packages/functions that must exist]
- [Note any required knowledge or context]

**Integration Point**: [Explain how this integrates with existing code]

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing [detailed description of the component/feature]. This [explain purpose and role in the larger system].

**Key Constraints**:
- ✅ Use ONLY Go standard library (no external dependencies)
- ✅ All code must be production-ready and fully tested
- ✅ Must integrate with existing [list relevant packages]
- ✅ Must handle all error cases explicitly
- ✅ Must include comprehensive documentation

---

### Directory Structure

Create the following structure:

```
[PROJECT_ROOT]/
├── pkg/
│   └── [package_name]/
│       ├── [file1].go          # [Description]
│       ├── [file2].go          # [Description]
│       ├── [file3].go          # [Description]
│       ├── [package]_test.go   # Comprehensive tests
│       └── doc.go              # Package documentation
├── cmd/
│   └── [demo_name]/
│       └── main.go             # Demonstration program
└── [other relevant directories]
```

---

### Core Types and Interfaces

**Define the following types**:

```go
package [package_name]

// [TypeName] [description of what this type represents]
type [TypeName] struct {
    [Field1] [type]  // [Field description]
    [Field2] [type]  // [Field description]
    // ... more fields
}

// [InterfaceName] [description of what this interface defines]
type [InterfaceName] interface {
    [Method1]([params]) ([returns], error)
    [Method2]([params]) ([returns], error)
    // ... more methods
}

// Constants
const (
    [ConstName1] [type] = [value]  // [Description]
    [ConstName2] [type] = [value]  // [Description]
)

// Errors
var (
    [ErrName1] = errors.New("[error message]")
    [ErrName2] = errors.New("[error message]")
)
```

---

### Implementation Steps

Follow these steps in order:

#### STEP 1: [Step Name]

**Purpose**: [What this step accomplishes]

**Implementation**:

Create `[file_path]`:

```go
package [package_name]

// [Detailed code example or pseudocode]
// [Include all necessary imports]
// [Include all function implementations]
// [Include all error handling]
```

**Requirements**:
- [Specific requirement 1]
- [Specific requirement 2]
- [etc.]

**Validation**:
```bash
# Commands to verify this step
[command 1]
[command 2]
```

---

#### STEP 2: [Step Name]

[Repeat structure from Step 1]

---

#### STEP 3: [Step Name]

[Repeat structure from Step 1]

---

[Continue with as many steps as needed]

---

### Testing Requirements

**Test Coverage**: Minimum 85% for this sub-phase

**Required Test Cases**:

1. **[Test Category 1]**:
   - Test: [Specific test case]
   - Expected: [Expected behavior]
   - Test: [Another specific test case]
   - Expected: [Expected behavior]

2. **[Test Category 2]**:
   - [Continue pattern]

3. **Error Cases**:
   - Test: [Error condition 1]
   - Expected: [Error type/behavior]
   - Test: [Error condition 2]
   - Expected: [Error type/behavior]

4. **Edge Cases**:
   - [Edge case 1]
   - [Edge case 2]

5. **Concurrency** (if applicable):
   - Test concurrent access to [shared resource]
   - Test race conditions with `-race` flag
   - Test deadlock scenarios

**Test Implementation Template**:

```go
package [package_name]

import (
    "testing"
    // other imports
)

func Test[FunctionName](t *testing.T) {
    tests := []struct {
        name    string
        input   [type]
        want    [type]
        wantErr bool
    }{
        {
            name:    "[test case name]",
            input:   [input value],
            want:    [expected output],
            wantErr: false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := [FunctionCall](tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}

func Benchmark[FunctionName](b *testing.B) {
    [setup code]
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        [function call]
    }
}
```

---

### Demonstration Program

Create `cmd/[demo-name]/main.go` that demonstrates:

1. **Basic Usage**: [What to demonstrate]
2. **Advanced Features**: [What to demonstrate]
3. **Error Handling**: [What to demonstrate]
4. **Performance**: [What to show]

**Demo Requirements**:
- Must be runnable with `go run cmd/[demo-name]/main.go`
- Must produce clear, formatted output
- Must demonstrate ALL key functionality
- Must show successful and error cases
- Must include timing/performance metrics if relevant

**Expected Output Format**:
```
[Demo Title]
==================

[Section 1 Title]
-----------------
[Expected output]

[Section 2 Title]
-----------------
[Expected output]

Summary
-------
[Summary statistics]
```

---

### Integration Requirements

**This sub-phase must integrate with**:
- [Package/component 1]: [How it integrates]
- [Package/component 2]: [How it integrates]

**Integration Tests**:

Create `[integration_test_file.go]` that tests:
1. [Integration scenario 1]
2. [Integration scenario 2]

```go
func TestIntegration[Scenario](t *testing.T) {
    // [Integration test code]
}
```

---

### Documentation Requirements

**Package Documentation** (`doc.go`):
```go
// Package [package_name] [comprehensive description]
//
// [Detailed explanation of what this package does]
//
// # Usage
//
// [Usage examples]
//
// # Architecture
//
// [Architectural notes]
//
// # Examples
//
//   [code example]
//
package [package_name]
```

**Function Documentation**:
- Every exported function must have a doc comment
- Format: `// [FunctionName] [what it does]`
- Include parameter descriptions if non-obvious
- Include example usage for complex functions
- Document error return conditions

**Type Documentation**:
- Every exported type must have a doc comment
- Explain what the type represents
- Document zero value behavior
- Document thread-safety properties

---

### Validation Checklist

Before marking this sub-phase complete, verify:

**Code Quality**:
- [ ] All code compiles without warnings
- [ ] `go vet ./...` passes with no issues
- [ ] `go fmt` has been run on all files
- [ ] No TODO comments remain in code
- [ ] All exported symbols are documented
- [ ] Code follows Go conventions (effective Go)
- [ ] Maximum function length: 50 lines (excluding comments)
- [ ] No panics in production code paths

**Testing**:
- [ ] All tests pass: `go test ./[package]/ -v`
- [ ] No race conditions: `go test ./[package]/ -race`
- [ ] Test coverage ≥85%: `go test ./[package]/ -cover`
- [ ] Benchmarks run: `go test ./[package]/ -bench=.`
- [ ] All error paths tested
- [ ] Edge cases covered

**Integration**:
- [ ] Integrates with [prerequisite package 1]
- [ ] Integrates with [prerequisite package 2]
- [ ] Integration tests pass
- [ ] Does not break existing functionality

**Demonstration**:
- [ ] Demo program compiles
- [ ] Demo runs without errors
- [ ] Demo output is clear and correct
- [ ] Demo shows all key features

**Documentation**:
- [ ] Package doc.go exists and is comprehensive
- [ ] All exported symbols documented
- [ ] README.md updated if needed
- [ ] Examples compile and run

**Performance**:
- [ ] No obvious performance issues
- [ ] Benchmarks show acceptable performance
- [ ] Memory usage is reasonable
- [ ] No memory leaks (tested with profiler)

---

### Acceptance Criteria

This sub-phase is considered complete when:

1. ✅ **All validation checklist items are checked**
2. ✅ **Code review reveals no issues** (self-review minimum)
3. ✅ **Demonstration program runs successfully** and shows expected output
4. ✅ **All tests pass** including race detector
5. ✅ **Coverage meets minimum** 85% threshold
6. ✅ **Integration verified** with previous sub-phases
7. ✅ **Documentation is complete** and clear
8. ✅ **No external dependencies** introduced

**Success Metrics**:
- Test coverage: [X]%
- Benchmarks: [metric] = [acceptable value]
- Memory usage: [acceptable value]
- [Other relevant metrics]

---

### Troubleshooting Guide

**Common Issues**:

**Issue**: [Common problem 1]
- **Symptom**: [How you'll know]
- **Cause**: [Why it happens]
- **Solution**: [How to fix]

**Issue**: [Common problem 2]
- **Symptom**: [How you'll know]
- **Cause**: [Why it happens]
- **Solution**: [How to fix]

---

### Deliverables Summary

Upon completion, you must have:

1. **Source Code**:
   - [ ] `pkg/[package_name]/[file1].go`
   - [ ] `pkg/[package_name]/[file2].go`
   - [ ] `pkg/[package_name]/doc.go`
   - [ ] [List all files]

2. **Tests**:
   - [ ] `pkg/[package_name]/[package]_test.go`
   - [ ] [Integration test files]

3. **Demonstration**:
   - [ ] `cmd/[demo-name]/main.go`

4. **Documentation**:
   - [ ] Package documentation in doc.go
   - [ ] README.md updates (if needed)
   - [ ] Code comments on all exports

5. **Verification**:
   - [ ] Test output logs showing all passing
   - [ ] Coverage report showing ≥85%
   - [ ] Demo execution showing success

---

### Next Sub-Phase

After completing this sub-phase, proceed to:
**[NEXT_PHASE_NUMBER]**: [Next phase name]

**What it will build on**:
- [How next phase uses this one]
- [Dependencies from this phase]

---

## TEMPLATE END

---

## EXAMPLE: FILLED TEMPLATE

Here's an example of the template filled out for a real sub-phase:

---

### PHASE 1.2.1: WebSocket Frame Implementation

**Phase Context**: Phase 1 builds the foundation communication layer between browser and backend.

**Sub-Phase Objective**: Implement RFC 6455 compliant WebSocket frame parsing and generation.

**Prerequisites**: 
- Phase 1.1 (Binary Protocol) must be complete
- `pkg/protocol` package must exist with Message types
- Understanding of WebSocket protocol (RFC 6455)

**Integration Point**: This will be used by the WebSocket connection handler to read/write frames over TCP connections.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing WebSocket frame parsing and generation according to RFC 6455. This will handle the low-level framing protocol that wraps all WebSocket messages. Frames can contain text, binary data, or control messages (ping, pong, close).

**Key Constraints**:
- ✅ Use ONLY Go standard library (no external dependencies)
- ✅ Must be RFC 6455 compliant
- ✅ Must handle fragmented messages
- ✅ Must support all frame types
- ✅ Must validate frame structure

---

### Directory Structure

Create the following structure:

```
webos/
├── pkg/
│   └── websocket/
│       ├── frame.go            # Frame type and constants
│       ├── frame_reader.go     # Frame reading logic
│       ├── frame_writer.go     # Frame writing logic
│       ├── frame_test.go       # Comprehensive tests
│       └── doc.go              # Package documentation
├── cmd/
│   └── frame-demo/
│       └── main.go             # Demonstration program
```

---

### Core Types and Interfaces

```go
package websocket

import "io"

// Frame represents a WebSocket frame.
type Frame struct {
    Fin    bool      // Final fragment flag
    RSV1   bool      // Reserved bit 1
    RSV2   bool      // Reserved bit 2
    RSV3   bool      // Reserved bit 3
    Opcode Opcode    // Frame opcode
    Masked bool      // Mask flag
    Mask   [4]byte   // Masking key (if masked)
    Payload []byte   // Frame payload
}

// Opcode represents WebSocket frame opcodes.
type Opcode uint8

const (
    OpcodeContinuation Opcode = 0x0
    OpcodeText         Opcode = 0x1
    OpcodeBinary       Opcode = 0x2
    OpcodeClose        Opcode = 0x8
    OpcodePing         Opcode = 0x9
    OpcodePong         Opcode = 0xA
)

// FrameReader reads WebSocket frames.
type FrameReader interface {
    ReadFrame() (*Frame, error)
}

// FrameWriter writes WebSocket frames.
type FrameWriter interface {
    WriteFrame(frame *Frame) error
}

// Errors
var (
    ErrInvalidFrame = errors.New("invalid frame")
    ErrInvalidOpcode = errors.New("invalid opcode")
    ErrFrameTooLarge = errors.New("frame too large")
)
```

---

### Implementation Steps

#### STEP 1: Frame Constants and Types

**Purpose**: Define the frame structure and all constants

**Implementation**:

Create `pkg/websocket/frame.go`:

```go
package websocket

import (
    "errors"
    "fmt"
)

// Opcode represents WebSocket frame opcodes per RFC 6455.
type Opcode uint8

const (
    OpcodeContinuation Opcode = 0x0 // Continuation frame
    OpcodeText         Opcode = 0x1 // Text frame
    OpcodeBinary       Opcode = 0x2 // Binary frame
    OpcodeClose        Opcode = 0x8 // Close frame
    OpcodePing         Opcode = 0x9 // Ping frame
    OpcodePong         Opcode = 0xA // Pong frame
)

// IsControl returns true if the opcode is a control frame.
func (o Opcode) IsControl() bool {
    return o >= 0x8
}

// IsData returns true if the opcode is a data frame.
func (o Opcode) IsData() bool {
    return o == OpcodeText || o == OpcodeBinary || o == OpcodeContinuation
}

// String returns the opcode name.
func (o Opcode) String() string {
    switch o {
    case OpcodeContinuation:
        return "CONTINUATION"
    case OpcodeText:
        return "TEXT"
    case OpcodeBinary:
        return "BINARY"
    case OpcodeClose:
        return "CLOSE"
    case OpcodePing:
        return "PING"
    case OpcodePong:
        return "PONG"
    default:
        return fmt.Sprintf("UNKNOWN(0x%X)", uint8(o))
    }
}

// Frame represents a WebSocket frame.
type Frame struct {
    Fin     bool     // Final fragment in message
    RSV1    bool     // Reserved bit 1
    RSV2    bool     // Reserved bit 2
    RSV3    bool     // Reserved bit 3
    Opcode  Opcode   // Frame opcode
    Masked  bool     // Is payload masked?
    Mask    [4]byte  // Masking key
    Payload []byte   // Frame payload
}

// Constants
const (
    MaxControlFramePayloadSize = 125
    MaxFrameHeaderSize         = 14
)

// Errors
var (
    ErrInvalidFrame        = errors.New("invalid frame")
    ErrInvalidOpcode       = errors.New("invalid opcode")
    ErrFrameTooLarge       = errors.New("frame too large")
    ErrControlFrameTooLong = errors.New("control frame payload too long")
    ErrFragmentedControl   = errors.New("control frames cannot be fragmented")
)

// Validate checks if the frame is valid per RFC 6455.
func (f *Frame) Validate() error {
    // Control frames cannot be fragmented
    if f.Opcode.IsControl() && !f.Fin {
        return ErrFragmentedControl
    }
    
    // Control frames have max payload of 125 bytes
    if f.Opcode.IsControl() && len(f.Payload) > MaxControlFramePayloadSize {
        return ErrControlFrameTooLong
    }
    
    // Reserved bits must be 0 unless extension negotiated
    if f.RSV1 || f.RSV2 || f.RSV3 {
        return ErrInvalidFrame
    }
    
    return nil
}
```

**Requirements**:
- Define all opcodes per RFC 6455
- Create Frame struct with all required fields
- Implement validation logic
- Define error types

**Validation**:
```bash
go build ./pkg/websocket/
go vet ./pkg/websocket/
```

---

#### STEP 2: Frame Reading

[Continue with detailed implementation...]

---

[And so on...]

---

## USAGE INSTRUCTIONS FOR AI AGENTS

When you receive a filled template like this:

1. **Read the entire prompt first** - don't start coding until you understand the complete picture
2. **Follow steps sequentially** - complete each step before moving to the next
3. **Run validation after each step** - ensure code compiles and works
4. **Write tests alongside code** - don't leave testing until the end
5. **Verify integration** - ensure it works with previous sub-phases
6. **Create the demo** - prove it works end-to-end
7. **Check the completion criteria** - mark each item as done
8. **Review your work** - self-review before declaring completion

**Do NOT proceed to next sub-phase until**:
- All tests pass
- Demo runs successfully  
- Coverage ≥ 85%
- Integration verified
- All checklist items checked
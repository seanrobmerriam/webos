## CRITICAL GUIDELINES FOR ALL PHASES

**Before Starting ANY Sub-Phase**:

1. **Read the ENTIRE prompt before writing any code**
2. **Use ONLY Go standard library** - zero external dependencies
3. **Write tests FIRST** - TDD approach ensures correctness
4. **Run tests after every function** - ensure 100% working code
5. **Include demo program** - prove it works end-to-end
6. **Document all public APIs** - clear comments required
7. **Verify integration** - test with previous phases

**Code Quality Requirements**:
- All exported functions/types must have doc comments
- Error handling must be explicit (no ignored errors)
- Use meaningful variable names
- Maximum function length: 50 lines
- Test coverage: minimum 85%
- No panics in production code (use errors)

**Validation Before Moving to Next Sub-Phase**:
```bash
# Must all pass:
go test ./... -v              # All tests pass
go test ./... -race           # No race conditions  
go vet ./...                  # No vet warnings
go build ./cmd/...            # All demos build
# Run demo and verify output
```

---

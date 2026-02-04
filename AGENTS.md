# Agents Guide for FFI Convertor

This document helps AI coding agents work effectively with the FFI Convertor project.

## Project Overview

FFI Convertor generates Go bindings for C libraries using libffi (no CGo). It parses C headers and outputs Go code.

## Project Structure

```
.
├── main.go                 # CLI entry point, flag parsing
├── parser/
│   ├── types.go            # AST types: Struct, Function, TypeDef, Enum, CType
│   └── parser.go           # Regex-based C header parser
├── generator/
│   └── generator.go        # Go code generator, templates, type mapping
├── testdata/
│   ├── calculator.h        # Example C header for testing
│   └── out/                # Generated output directory
└── go.mod
```

## Build and Test Commands

```bash
# Build the tool
go build .

# Run with test header
./ffi-convertor -header testdata/calculator.h -output testdata/out -package calculator -lib calculator

# Check generated code compiles (from testdata/out, needs go.mod)
cd testdata/out && go build .
```

## Type Mapping Reference

When modifying the generator, use these mappings:

| C Type | Go Type | FFI Type |
|--------|---------|----------|
| void | (none) | `&ffi.TypeVoid` |
| bool | bool | `&ffi.TypeUint8` |
| char | int8 | `&ffi.TypeSint8` |
| unsigned char / uint8_t | uint8 | `&ffi.TypeUint8` |
| short / int16_t | int16 | `&ffi.TypeSint16` |
| int / int32_t | int32 | `&ffi.TypeSint32` |
| long / int64_t | int64 | `&ffi.TypeSint64` |
| size_t / uint64_t | uint64 | `&ffi.TypeUint64` |
| float | float32 | `&ffi.TypeFloat` |
| double | float64 | `&ffi.TypeDouble` |
| char* | string | `&ffi.TypePointer` |
| T* (any pointer) | uintptr | `&ffi.TypePointer` |
| struct by value | StructName | `&FFITypeStructName` |

## Critical FFI Patterns

### Return values for integers < 8 bytes

Always use `ffi.Arg` and cast afterward:

```go
var result ffi.Arg  // NOT int32
myFunc.Call(unsafe.Pointer(&result))
return int32(result)
```

### String parameters

Convert with `unix.BytePtrFromString`:

```go
namePtr, _ := unix.BytePtrFromString(name)
myFunc.Call(..., unsafe.Pointer(&namePtr))
```

### String return values

```go
var ptr *byte
myFunc.Call(unsafe.Pointer(&ptr))
return unix.BytePtrToString(ptr)
```

### Struct by value

Pass struct pointer directly (not wrapped in unsafe.Pointer):

```go
myFunc.Call(..., &config)  // config is a struct
```

## Verification Checklist

After making changes:

1. **Build succeeds**: `go build .`
2. **Generate test output**: `./ffi-convertor -header testdata/calculator.h -output testdata/out -package calculator -lib calculator`
3. **Verify generated files exist**: `ls testdata/out/` should show `loader.go`, `types.go`, `functions.go`
4. **Check generated code syntax**: Create a temporary go.mod in testdata/out and run `go build .`

## Adding New C Type Support

1. Add parsing logic in `parser/parser.go` (regex patterns)
2. Add type mapping in `generator/generator.go`:
   - `cTypeToGoType()` - C to Go type
   - `cTypeToFFIType()` - C to FFI type descriptor
3. Update function wrapper generation if needed in `generateFunctionWrapper()`
4. Add test case to `testdata/calculator.h`
5. Regenerate and verify

## Common Issues

### Parser misses a construct
The parser uses regex, not a full C parser. Complex macros, nested structs, or unusual syntax may not parse. Add specific regex patterns to handle new cases.

### Generated code doesn't compile
Check that:
- FFI types match Go struct field order exactly
- Pointer vs value semantics are correct
- `ffi.Arg` is used for small integer returns

### Wrong function signature
Verify the C function declaration is on a single line (after preprocessing). Multi-line declarations may not parse correctly.

## Dependencies

The tool itself has no external dependencies beyond the standard library.

Generated code requires:
- `github.com/jupiterrider/ffi v0.3.0`
- `golang.org/x/sys v0.28.0`

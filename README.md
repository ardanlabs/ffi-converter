# FFI Convertor

Automatically generate Go bindings for C libraries without CGo.

## What This Does

FFI Convertor reads a C header file and generates Go code that calls the native library at runtime using [libffi](https://github.com/libffi/libffi). This means:

- **No CGo required** - Your Go code compiles with `go build`, no C compiler needed
- **Cross-compilation works** - Build for any platform from any platform
- **Faster builds** - No CGo compilation overhead

## Quick Start

```bash
# Build the tool
go build .

# Generate bindings from a C header
./ffi-convertor -header mylib.h -output ./mylib -package mylib

# Use the generated bindings
```

```go
package main

import "myproject/mylib"

func main() {
    if err := mylib.Load("/path/to/libs"); err != nil {
        panic(err)
    }
    
    result := mylib.SomeFunction(42)
}
```

## Command-Line Options

| Flag | Required | Description |
|------|----------|-------------|
| `-header` | Yes | Path to the C header file |
| `-output` | No | Output directory (default: current directory) |
| `-package` | No | Go package name (default: "bindings") |
| `-lib` | No | Library name, e.g., "mylib" becomes libmylib.so/dylib (default: header filename) |

## What Gets Generated

Given this C header:

```c
typedef struct {
    double value;
    int32_t precision;
} Config;

typedef struct Context_s* Context;

Config get_default_config(void);
Context create_context(Config cfg);
void destroy_context(Context ctx);
double calculate(Context ctx, double input);
const char* get_version(void);
```

The tool generates three files:

### loader.go
Loads the shared library with platform detection (`.so`, `.dylib`, `.dll`).

### types.go
Go structs matching the C structs, plus FFI type descriptors:

```go
type Config struct {
    Value     float64
    Precision int32
}

type Context uintptr  // Opaque handle

var FFITypeConfig = ffi.NewType(
    &ffi.TypeDouble,
    &ffi.TypeSint32,
)
```

### functions.go
Go functions that call into the native library:

```go
func GetDefaultConfig() Config
func CreateContext(cfg Config) Context
func DestroyContext(ctx Context)
func Calculate(ctx Context, input float64) float64
func GetVersion() string
```

## Supported C Features

- Primitive types: `int`, `float`, `double`, `char`, etc.
- Fixed-width types: `int32_t`, `uint64_t`, `size_t`
- Structs passed by value or pointer
- Opaque handles (`typedef struct X_s* X`)
- Enums
- String parameters and return values (`char*`, `const char*`)
- Pointer parameters

## Requirements

### Build Time
Just Go 1.18+

### Runtime
- **Linux/FreeBSD**: Install libffi (`apt install libffi8` or `dnf install libffi`)
- **macOS**: libffi is bundled
- **Windows (AMD64)**: libffi is bundled

### Generated Code Dependencies

Add these to your `go.mod`:

```
require (
    github.com/jupiterrider/ffi v0.3.0
    golang.org/x/sys v0.28.0
)
```

## Limitations

- Variadic functions need manual adjustment
- Callbacks require additional manual setup using `ffi.Closure`
- Complex preprocessor macros are not parsed
- Bitfields are not supported

## How It Works

1. **Parse**: Regex-based parser extracts structs, functions, typedefs, and enums from the header
2. **Map Types**: C types are mapped to Go types and FFI type descriptors
3. **Generate**: Templates produce idiomatic Go code following FFI best practices

The generated code uses the [jupiterrider/ffi](https://github.com/jupiterrider/ffi) library which wraps libffi for Go.

## Example

```bash
./ffi-convertor -header calculator.h -output ./calculator -package calculator -lib calculator
```

Then in your application:

```go
package main

import (
    "fmt"
    "myapp/calculator"
)

func main() {
    if err := calculator.Load("/usr/local/lib"); err != nil {
        panic(err)
    }
    
    cfg := calculator.CalcDefaultConfig()
    cfg.Precision = 4
    
    calc := calculator.CalcCreate(cfg)
    defer calculator.CalcFree(calc)
    
    result := calculator.CalcAdd(calc, 1.5, 2.5)
    fmt.Printf("Result: %.4f\n", result)
}
```

## License

Apache 2.0

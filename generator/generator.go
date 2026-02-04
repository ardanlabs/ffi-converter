package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/ardanlabs/ffi-converter/parser"
)

type Generator struct {
	packageName string
	libName     string
	header      *parser.Header
}

func New(packageName, libName string, header *parser.Header) *Generator {
	return &Generator{
		packageName: packageName,
		libName:     libName,
		header:      header,
	}
}

func (g *Generator) Generate() (map[string]string, error) {
	files := make(map[string]string)

	loaderCode, err := g.generateLoader()
	if err != nil {
		return nil, fmt.Errorf("generating loader: %w", err)
	}
	files["loader.go"] = loaderCode

	typesCode, err := g.generateTypes()
	if err != nil {
		return nil, fmt.Errorf("generating types: %w", err)
	}
	files["types.go"] = typesCode

	funcsCode, err := g.generateFunctions()
	if err != nil {
		return nil, fmt.Errorf("generating functions: %w", err)
	}
	files["functions.go"] = funcsCode

	return files, nil
}

func (g *Generator) generateLoader() (string, error) {
	tmpl := `package {{.Package}}

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jupiterrider/ffi"
)

var lib ffi.Lib

func Load(path string) error {
	var err error
	lib, err = ffi.Load(getLibraryPath(path))
	if err != nil {
		return fmt.Errorf("failed to load library: %w", err)
	}

	if err := loadFuncs(); err != nil {
		return err
	}

	return nil
}

func getLibraryPath(basePath string) string {
	var filename string
	switch runtime.GOOS {
	case "linux", "freebsd":
		filename = "lib{{.LibName}}.so"
	case "darwin":
		filename = "lib{{.LibName}}.dylib"
	case "windows":
		filename = "{{.LibName}}.dll"
	default:
		filename = "lib{{.LibName}}.so"
	}
	return filepath.Join(basePath, filename)
}
`

	t, err := template.New("loader").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]string{
		"Package": g.packageName,
		"LibName": g.libName,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (g *Generator) generateTypes() (string, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)
	fmt.Fprintf(&buf, "import \"github.com/jupiterrider/ffi\"\n\n")

	for _, s := range g.header.Structs {
		if s.IsOpaque {
			fmt.Fprintf(&buf, "type %s uintptr\n\n", toGoName(s.Name))
			continue
		}

		fmt.Fprintf(&buf, "type %s struct {\n", toGoName(s.Name))
		for _, f := range s.Fields {
			goType := cTypeToGoType(f.Type, g.header)
			fmt.Fprintf(&buf, "\t%s %s\n", toGoName(f.Name), goType)
		}
		fmt.Fprintf(&buf, "}\n\n")

		fmt.Fprintf(&buf, "var FFIType%s = ffi.NewType(\n", toGoName(s.Name))
		for _, f := range s.Fields {
			ffiType := cTypeToFFIType(f.Type, g.header)
			fmt.Fprintf(&buf, "\t%s,\n", ffiType)
		}
		fmt.Fprintf(&buf, ")\n\n")
	}

	for _, e := range g.header.Enums {
		fmt.Fprintf(&buf, "type %s int32\n\n", toGoName(e.Name))
		fmt.Fprintf(&buf, "const (\n")
		for i, v := range e.Values {
			if v.Value != "" {
				fmt.Fprintf(&buf, "\t%s %s = %s\n", toGoEnumName(e.Name, v.Name), toGoName(e.Name), v.Value)
			} else if i == 0 {
				fmt.Fprintf(&buf, "\t%s %s = iota\n", toGoEnumName(e.Name, v.Name), toGoName(e.Name))
			} else {
				fmt.Fprintf(&buf, "\t%s\n", toGoEnumName(e.Name, v.Name))
			}
		}
		fmt.Fprintf(&buf, ")\n\n")
	}

	return buf.String(), nil
}

func (g *Generator) generateFunctions() (string, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)
	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"fmt\"\n")
	fmt.Fprintf(&buf, "\t\"unsafe\"\n\n")
	fmt.Fprintf(&buf, "\t\"github.com/jupiterrider/ffi\"\n")
	fmt.Fprintf(&buf, "\t\"golang.org/x/sys/unix\"\n")
	fmt.Fprintf(&buf, ")\n\n")

	fmt.Fprintf(&buf, "var _ = unix.BytePtrFromString\n\n")

	fmt.Fprintf(&buf, "var (\n")
	for _, fn := range g.header.Functions {
		funcVarName := toLowerCamel(fn.Name) + "Func"
		fmt.Fprintf(&buf, "\t%s ffi.Fun\n", funcVarName)
	}
	fmt.Fprintf(&buf, ")\n\n")

	fmt.Fprintf(&buf, "func loadFuncs() error {\n")
	fmt.Fprintf(&buf, "\tvar err error\n\n")

	for _, fn := range g.header.Functions {
		funcVarName := toLowerCamel(fn.Name) + "Func"
		retFFI := cTypeToFFIType(fn.ReturnType, g.header)

		var argFFIs []string
		for _, p := range fn.Params {
			argFFIs = append(argFFIs, cTypeToFFIType(p.Type, g.header))
		}

		if len(argFFIs) == 0 {
			fmt.Fprintf(&buf, "\tif %s, err = lib.Prep(\"%s\", %s); err != nil {\n",
				funcVarName, fn.Name, retFFI)
		} else {
			fmt.Fprintf(&buf, "\tif %s, err = lib.Prep(\"%s\", %s, %s); err != nil {\n",
				funcVarName, fn.Name, retFFI, strings.Join(argFFIs, ", "))
		}
		fmt.Fprintf(&buf, "\t\treturn fmt.Errorf(\"%s: %%w\", err)\n", fn.Name)
		fmt.Fprintf(&buf, "\t}\n\n")
	}

	fmt.Fprintf(&buf, "\treturn nil\n")
	fmt.Fprintf(&buf, "}\n\n")

	for _, fn := range g.header.Functions {
		code := g.generateFunctionWrapper(fn)
		fmt.Fprintf(&buf, "%s\n", code)
	}

	return buf.String(), nil
}

func (g *Generator) generateFunctionWrapper(fn parser.Function) string {
	var buf bytes.Buffer

	goFuncName := toGoName(fn.Name)
	funcVarName := toLowerCamel(fn.Name) + "Func"

	var params []string
	for _, p := range fn.Params {
		goType := cTypeToGoType(p.Type, g.header)
		paramName := toLowerCamel(p.Name)
		if paramName == "" {
			paramName = "arg"
		}
		params = append(params, fmt.Sprintf("%s %s", paramName, goType))
	}
	paramsStr := strings.Join(params, ", ")

	retGoType := cTypeToGoType(fn.ReturnType, g.header)
	hasReturn := fn.ReturnType.Name != "void"

	if hasReturn {
		fmt.Fprintf(&buf, "func %s(%s) %s {\n", goFuncName, paramsStr, retGoType)
	} else {
		fmt.Fprintf(&buf, "func %s(%s) {\n", goFuncName, paramsStr)
	}

	for _, p := range fn.Params {
		if isStringType(p.Type) {
			paramName := toLowerCamel(p.Name)
			if paramName == "" {
				paramName = "arg"
			}
			fmt.Fprintf(&buf, "\t%sPtr, _ := unix.BytePtrFromString(%s)\n", paramName, paramName)
		}
	}

	if hasReturn {
		if needsFFIArg(fn.ReturnType) {
			fmt.Fprintf(&buf, "\tvar result ffi.Arg\n")
		} else if isStringReturnType(fn.ReturnType) {
			fmt.Fprintf(&buf, "\tvar resultPtr *byte\n")
		} else if isOpaqueHandle(fn.ReturnType, g.header) {
			fmt.Fprintf(&buf, "\tvar result %s\n", retGoType)
		} else {
			fmt.Fprintf(&buf, "\tvar result %s\n", retGoType)
		}
	}

	var callArgs []string
	if hasReturn {
		if isStringReturnType(fn.ReturnType) {
			callArgs = append(callArgs, "unsafe.Pointer(&resultPtr)")
		} else {
			callArgs = append(callArgs, "unsafe.Pointer(&result)")
		}
	} else {
		callArgs = append(callArgs, "nil")
	}

	for _, p := range fn.Params {
		paramName := toLowerCamel(p.Name)
		if paramName == "" {
			paramName = "arg"
		}
		if isStringType(p.Type) {
			callArgs = append(callArgs, fmt.Sprintf("unsafe.Pointer(&%sPtr)", paramName))
		} else if isStructByValue(p.Type, g.header) {
			callArgs = append(callArgs, fmt.Sprintf("&%s", paramName))
		} else {
			callArgs = append(callArgs, fmt.Sprintf("unsafe.Pointer(&%s)", paramName))
		}
	}

	fmt.Fprintf(&buf, "\t%s.Call(%s)\n", funcVarName, strings.Join(callArgs, ", "))

	if hasReturn {
		if needsFFIArg(fn.ReturnType) {
			if fn.ReturnType.Name == "bool" || (fn.ReturnType.Name == "uint8" && !fn.ReturnType.IsPointer) {
				fmt.Fprintf(&buf, "\treturn result.Bool()\n")
			} else {
				fmt.Fprintf(&buf, "\treturn %s(result)\n", retGoType)
			}
		} else if isStringReturnType(fn.ReturnType) {
			fmt.Fprintf(&buf, "\tif resultPtr == nil {\n")
			fmt.Fprintf(&buf, "\t\treturn \"\"\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\treturn unix.BytePtrToString(resultPtr)\n")
		} else {
			fmt.Fprintf(&buf, "\treturn result\n")
		}
	}

	fmt.Fprintf(&buf, "}\n")

	return buf.String()
}

func cTypeToGoType(ct parser.CType, header *parser.Header) string {
	if ct.IsPointer && (ct.Name == "char" || ct.Name == "char *") {
		return "string"
	}

	if ct.IsPointer {
		for _, s := range header.Structs {
			if s.Name == ct.Name && !s.IsOpaque {
				return "*" + toGoName(ct.Name)
			}
		}
		return "uintptr"
	}

	switch ct.Name {
	case "void":
		return ""
	case "bool":
		return "bool"
	case "char":
		if ct.IsUnsigned {
			return "uint8"
		}
		return "int8"
	case "short":
		if ct.IsUnsigned {
			return "uint16"
		}
		return "int16"
	case "int":
		if ct.IsUnsigned {
			return "uint32"
		}
		return "int32"
	case "long":
		if ct.IsUnsigned {
			return "uint64"
		}
		return "int64"
	case "int8_t":
		return "int8"
	case "uint8_t":
		return "uint8"
	case "int16_t":
		return "int16"
	case "uint16_t":
		return "uint16"
	case "int32_t":
		return "int32"
	case "uint32_t":
		return "uint32"
	case "int64_t":
		return "int64"
	case "uint64_t":
		return "uint64"
	case "size_t":
		return "uint64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		for _, s := range header.Structs {
			if s.Name == ct.Name {
				return toGoName(ct.Name)
			}
		}
		for _, e := range header.Enums {
			if e.Name == ct.Name {
				return toGoName(ct.Name)
			}
		}
		return toGoName(ct.Name)
	}
}

func cTypeToFFIType(ct parser.CType, header *parser.Header) string {
	if ct.IsPointer {
		return "&ffi.TypePointer"
	}

	switch ct.Name {
	case "void":
		return "&ffi.TypeVoid"
	case "bool":
		return "&ffi.TypeUint8"
	case "char":
		if ct.IsUnsigned {
			return "&ffi.TypeUint8"
		}
		return "&ffi.TypeSint8"
	case "short":
		if ct.IsUnsigned {
			return "&ffi.TypeUint16"
		}
		return "&ffi.TypeSint16"
	case "int":
		if ct.IsUnsigned {
			return "&ffi.TypeUint32"
		}
		return "&ffi.TypeSint32"
	case "long":
		if ct.IsUnsigned {
			return "&ffi.TypeUint64"
		}
		return "&ffi.TypeSint64"
	case "int8_t":
		return "&ffi.TypeSint8"
	case "uint8_t":
		return "&ffi.TypeUint8"
	case "int16_t":
		return "&ffi.TypeSint16"
	case "uint16_t":
		return "&ffi.TypeUint16"
	case "int32_t":
		return "&ffi.TypeSint32"
	case "uint32_t":
		return "&ffi.TypeUint32"
	case "int64_t":
		return "&ffi.TypeSint64"
	case "uint64_t":
		return "&ffi.TypeUint64"
	case "size_t":
		return "&ffi.TypeUint64"
	case "float":
		return "&ffi.TypeFloat"
	case "double":
		return "&ffi.TypeDouble"
	default:
		for _, s := range header.Structs {
			if s.Name == ct.Name && !s.IsOpaque {
				return "&FFIType" + toGoName(ct.Name)
			}
		}
		return "&ffi.TypePointer"
	}
}

func toGoName(name string) string {
	if name == "" {
		return ""
	}

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_'
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(string(part[0])))
			if len(part) > 1 {
				lower := strings.ToLower(part[1:])
				for _, acronym := range []string{"id", "url", "api", "http", "json", "xml", "sql", "io", "ip", "tcp", "udp"} {
					if strings.ToLower(part) == acronym {
						result.Reset()
						result.WriteString(strings.ToUpper(part))
						break
					}
				}
				if result.Len() > 0 && !strings.HasSuffix(result.String(), strings.ToUpper(part)) {
					result.WriteString(lower)
				}
			}
		}
	}

	return result.String()
}

func toLowerCamel(name string) string {
	goName := toGoName(name)
	if goName == "" {
		return ""
	}
	runes := []rune(goName)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func toGoEnumName(enumName, valueName string) string {
	return toGoName(valueName)
}

func isStringType(ct parser.CType) bool {
	return ct.IsPointer && ct.Name == "char"
}

func isStringReturnType(ct parser.CType) bool {
	return ct.IsPointer && ct.Name == "char"
}

func needsFFIArg(ct parser.CType) bool {
	if ct.IsPointer || ct.Name == "void" {
		return false
	}

	switch ct.Name {
	case "int8_t", "uint8_t", "char", "bool":
		return true
	case "int16_t", "uint16_t", "short":
		return true
	case "int32_t", "uint32_t", "int":
		return true
	case "float", "double":
		return false
	default:
		return false
	}
}

func isOpaqueHandle(ct parser.CType, header *parser.Header) bool {
	if !ct.IsPointer {
		return false
	}
	for _, s := range header.Structs {
		if s.Name == ct.Name && s.IsOpaque {
			return true
		}
	}
	return false
}

func isStructByValue(ct parser.CType, header *parser.Header) bool {
	if ct.IsPointer {
		return false
	}
	for _, s := range header.Structs {
		if s.Name == ct.Name && !s.IsOpaque {
			return true
		}
	}
	return false
}

package parser

import (
	"regexp"
	"strings"
)

var blockCommentRe = regexp.MustCompile(`/\*[\s\S]*?\*/`)
var lineCommentRe = regexp.MustCompile(`//[^\n]*`)
var multiSpaceRe = regexp.MustCompile(`[ \t]+`)
var typedefRe = regexp.MustCompile(`typedef\s+(?:struct\s+)?(\w+(?:\s*\*)?)\s+(\w+)\s*;`)
var opaqueRe = regexp.MustCompile(`typedef\s+struct\s+(\w+)_s\s*\*\s*(\w+)\s*;`)
var structRe = regexp.MustCompile(`typedef\s+struct\s*(?:\w+)?\s*\{([^}]+)\}\s*(\w+)\s*;`)
var enumRe = regexp.MustCompile(`typedef\s+enum\s*(?:\w+)?\s*\{([^}]+)\}\s*(\w+)\s*;`)
var funcRe = regexp.MustCompile(`(?m)^[ \t]*((?:const\s+)?(?:unsigned\s+)?(?:struct\s+)?\w+(?:\s*\*)?)\s+(\w+)\s*\(([^)]*)\)\s*;`)

func Parse(content string) (*Header, error) {
	content = removeComments(content)
	content = normalizeWhitespace(content)

	header := &Header{}

	parseTypeDefs(content, header)
	parseStructs(content, header)
	parseEnums(content, header)
	parseFunctions(content, header)

	return header, nil
}

func removeComments(s string) string {
	s = blockCommentRe.ReplaceAllString(s, "")
	s = lineCommentRe.ReplaceAllString(s, "")

	return s
}

func normalizeWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = multiSpaceRe.ReplaceAllString(s, " ")

	return s
}

func parseTypeDefs(content string, header *Header) {
	matches := typedefRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		srcType := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])

		ctype := parseCType(srcType)

		header.TypeDefs = append(header.TypeDefs, TypeDef{
			Name:       name,
			SourceType: ctype,
		})
	}

	matches = opaqueRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		name := strings.TrimSpace(m[2])

		header.Structs = append(header.Structs, Struct{
			Name:     name,
			IsOpaque: true,
		})
	}
}

func parseStructs(content string, header *Header) {
	matches := structRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		body := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])

		fields := parseStructFields(body)

		header.Structs = append(header.Structs, Struct{
			Name:   name,
			Fields: fields,
		})
	}
}

func parseStructFields(body string) []StructField {
	var fields []StructField

	lines := strings.Split(body, ";")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[len(parts)-1]
		name = strings.TrimPrefix(name, "*")
		typeParts := parts[:len(parts)-1]

		if strings.HasPrefix(parts[len(parts)-1], "*") {
			typeParts = append(typeParts, "*")
		}

		typeStr := strings.Join(typeParts, " ")
		ctype := parseCType(typeStr)

		fields = append(fields, StructField{
			Name: name,
			Type: ctype,
		})
	}

	return fields
}

func parseCType(typeStr string) CType {
	typeStr = strings.TrimSpace(typeStr)

	ct := CType{}

	if strings.Contains(typeStr, "const") {
		ct.IsConst = true
		typeStr = strings.ReplaceAll(typeStr, "const", "")
		typeStr = strings.TrimSpace(typeStr)
	}

	if strings.Contains(typeStr, "unsigned") {
		ct.IsUnsigned = true
		typeStr = strings.ReplaceAll(typeStr, "unsigned", "")
		typeStr = strings.TrimSpace(typeStr)
	}

	if strings.HasSuffix(typeStr, "*") || strings.Contains(typeStr, "* ") {
		ct.IsPointer = true
		typeStr = strings.ReplaceAll(typeStr, "*", "")
		typeStr = strings.TrimSpace(typeStr)
	}

	ct.Name = strings.TrimSpace(typeStr)

	return ct
}

func parseEnums(content string, header *Header) {
	matches := enumRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		body := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])

		values := parseEnumValues(body)

		header.Enums = append(header.Enums, Enum{
			Name:   name,
			Values: values,
		})
	}
}

func parseEnumValues(body string) []EnumValue {
	var values []EnumValue

	parts := strings.Split(body, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if idx := strings.Index(part, "="); idx != -1 {
			name := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			values = append(values, EnumValue{Name: name, Value: value})
		} else {
			values = append(values, EnumValue{Name: part})
		}
	}

	return values
}

func parseFunctions(content string, header *Header) {
	matches := funcRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) < 4 {
			continue
		}

		retType := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])
		paramsStr := strings.TrimSpace(m[3])

		fn := Function{
			Name:       name,
			ReturnType: parseCType(retType),
		}

		if paramsStr == "void" || paramsStr == "" {
			fn.Params = nil
		} else {
			fn.Params, fn.IsVariadic = parseParams(paramsStr)
		}

		header.Functions = append(header.Functions, fn)
	}
}

func parseParams(paramsStr string) ([]FunctionParam, bool) {
	var params []FunctionParam
	isVariadic := false

	parts := strings.Split(paramsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if part == "..." {
			isVariadic = true
			continue
		}

		tokens := strings.Fields(part)
		if len(tokens) < 2 {
			params = append(params, FunctionParam{
				Name: "",
				Type: parseCType(part),
			})
			continue
		}

		name := tokens[len(tokens)-1]
		name = strings.TrimPrefix(name, "*")
		typeParts := tokens[:len(tokens)-1]

		if strings.HasPrefix(tokens[len(tokens)-1], "*") {
			typeParts = append(typeParts, "*")
		}

		typeStr := strings.Join(typeParts, " ")

		params = append(params, FunctionParam{
			Name: name,
			Type: parseCType(typeStr),
		})
	}

	return params, isVariadic
}

package parser

type CType struct {
	Name       string
	IsPointer  bool
	IsConst    bool
	IsUnsigned bool
	IsArray    bool
	ArraySize  int
}

type StructField struct {
	Name string
	Type CType
}

type Struct struct {
	Name     string
	TypeDef  string
	Fields   []StructField
	IsOpaque bool
}

type FunctionParam struct {
	Name string
	Type CType
}

type Function struct {
	Name       string
	ReturnType CType
	Params     []FunctionParam
	IsVariadic bool
}

type TypeDef struct {
	Name       string
	SourceType CType
}

type EnumValue struct {
	Name  string
	Value string
}

type Enum struct {
	Name   string
	Values []EnumValue
}

type Header struct {
	Structs   []Struct
	Functions []Function
	TypeDefs  []TypeDef
	Enums     []Enum
}

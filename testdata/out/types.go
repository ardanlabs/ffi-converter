package calculator

import "github.com/jupiterrider/ffi"

type Calc uintptr

type Calcconfig struct {
	Value float64
	Precision int32
	UseCache uint8
}

var FFITypeCalcconfig = ffi.NewType(
	&ffi.TypeDouble,
	&ffi.TypeSint32,
	&ffi.TypeUint8,
)


package calculator

import (
	"fmt"
	"unsafe"

	"github.com/jupiterrider/ffi"
	"golang.org/x/sys/unix"
)

var _ = unix.BytePtrFromString

var (
	calcDefaultConfigFunc ffi.Fun
	calcCreateFunc ffi.Fun
	calcFreeFunc ffi.Fun
	calcAddFunc ffi.Fun
	calcGetVersionFunc ffi.Fun
	calcFormatFunc ffi.Fun
)

func loadFuncs() error {
	var err error

	if calcDefaultConfigFunc, err = lib.Prep("calc_default_config", &FFITypeCalcconfig); err != nil {
		return fmt.Errorf("calc_default_config: %w", err)
	}

	if calcCreateFunc, err = lib.Prep("calc_create", &ffi.TypePointer, &FFITypeCalcconfig); err != nil {
		return fmt.Errorf("calc_create: %w", err)
	}

	if calcFreeFunc, err = lib.Prep("calc_free", &ffi.TypeVoid, &ffi.TypePointer); err != nil {
		return fmt.Errorf("calc_free: %w", err)
	}

	if calcAddFunc, err = lib.Prep("calc_add", &ffi.TypeDouble, &ffi.TypePointer, &ffi.TypeDouble, &ffi.TypeDouble); err != nil {
		return fmt.Errorf("calc_add: %w", err)
	}

	if calcGetVersionFunc, err = lib.Prep("calc_get_version", &ffi.TypePointer); err != nil {
		return fmt.Errorf("calc_get_version: %w", err)
	}

	if calcFormatFunc, err = lib.Prep("calc_format", &ffi.TypeSint32, &ffi.TypePointer, &ffi.TypePointer, &ffi.TypeUint64); err != nil {
		return fmt.Errorf("calc_format: %w", err)
	}

	return nil
}

func CalcDefaultConfig() Calcconfig {
	var result Calcconfig
	calcDefaultConfigFunc.Call(unsafe.Pointer(&result))
	return result
}

func CalcCreate(config Calcconfig) Calc {
	var result Calc
	calcCreateFunc.Call(unsafe.Pointer(&result), &config)
	return result
}

func CalcFree(calc Calc) {
	calcFreeFunc.Call(nil, unsafe.Pointer(&calc))
}

func CalcAdd(calc Calc, a float64, b float64) float64 {
	var result float64
	calcAddFunc.Call(unsafe.Pointer(&result), unsafe.Pointer(&calc), unsafe.Pointer(&a), unsafe.Pointer(&b))
	return result
}

func CalcGetVersion() string {
	var resultPtr *byte
	calcGetVersionFunc.Call(unsafe.Pointer(&resultPtr))
	if resultPtr == nil {
		return ""
	}
	return unix.BytePtrToString(resultPtr)
}

func CalcFormat(calc Calc, buf string, bufSize uint64) int32 {
	bufPtr, _ := unix.BytePtrFromString(buf)
	var result ffi.Arg
	calcFormatFunc.Call(unsafe.Pointer(&result), unsafe.Pointer(&calc), unsafe.Pointer(&bufPtr), unsafe.Pointer(&bufSize))
	return int32(result)
}


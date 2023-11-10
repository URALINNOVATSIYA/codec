package codec

import "fmt"

const (
	StructCodingModeDefault = 0
	StructCodingModeIndex   = 1
	StructCodingModeName    = 2
)

var structCodingMode int

func SetStructCodingMode(mode int) {
	if mode < StructCodingModeDefault && mode > StructCodingModeName {
		panic(fmt.Sprintf("Invalid structure coding mode %d", mode))
	}
	structCodingMode = mode
}

func GetStructCodingMode() int {
	return structCodingMode
}

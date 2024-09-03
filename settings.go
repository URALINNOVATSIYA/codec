package codec

import "fmt"

const (
	StructCodingModeDefault byte = 0
	StructCodingModeIndex   byte = 1
	StructCodingModeName    byte = 2
)

var structCodingMode byte

func SetDefaultStructCodingMode(mode byte) {
	if mode < StructCodingModeDefault && mode > StructCodingModeName {
		panic(fmt.Errorf("Invalid structure coding mode %d", mode))
	}
	structCodingMode = mode
}

func GetDefaultStructCodingMode() byte {
	return structCodingMode
}

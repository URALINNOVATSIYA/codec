package codec

var defaultTypeReg *TypeRegistry

func init() {
	determineReflectValueFlagOffset()
	defaultTypeReg = NewTypeRegistry(true)
	defaultTypeReg.RegisterBaseTypes()
}

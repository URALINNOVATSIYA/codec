package codec

var defaultTypeReg *TypeRegistry

func init() {
	defaultTypeReg = NewTypeRegistry(true)
	defaultTypeReg.RegisterBaseTypes()
}

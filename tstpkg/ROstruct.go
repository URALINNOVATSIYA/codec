package tstpkg

type privateFuncType func() int

type TstStruct struct {
	privateField privateFuncType
	publicField  string
}

func privateFunc() int {
	return 42
}

func SetPrivateValue(s *TstStruct) {
	s.privateField = privateFunc
}

func Unser() *TstStruct {
	tStr := &TstStruct{
		privateField: nil,
		publicField:  "abs",
	}

	SetPrivateValue(tStr)
	return tStr
}

func (t *TstStruct) GetPublicField() string {
	return t.publicField
}

func (t *TstStruct) PrivateField() int {
	return t.privateField()
}

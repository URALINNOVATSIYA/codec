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

func NewTestStruct() TstStruct {
	return TstStruct{
		privateField: privateFunc,
		publicField:  "abs",
	}
}

func (t TstStruct) AsInterface() interface{} {
	return t
}

func CheckPrivateFunctionality(s interface{}) bool {
	if ts, ok := s.(TstStruct); ok {
		return ts.privateField() == privateFunc()
	}
	return false
}

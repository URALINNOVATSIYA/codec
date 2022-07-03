package codec

type Serializable interface {
	Serialize() []byte
	Unserialize([]byte) (any, error)
}

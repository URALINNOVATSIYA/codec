package codec

import (
	"bytes"
	"testing"
)

func TestSerializerReusesValuesBuffer(t *testing.T) {
	serializer := NewSerializer()
	value := &struct {
		Name  string
		Items []string
	}{
		Name:  "record",
		Items: []string{"first", "second"},
	}

	first := serializer.Encode(value)
	if cap(serializer.values) == 0 {
		t.Fatal("Encode did not retain the values buffer")
	}

	storage := &serializer.values[:cap(serializer.values)][0]
	for i, v := range serializer.values[:cap(serializer.values)] {
		if v.IsValid() {
			t.Fatalf("values buffer retains a reflect.Value at index %d", i)
		}
	}

	second := serializer.Encode(value)
	if !bytes.Equal(first, second) {
		t.Fatal("reusing the serializer changed the encoded output")
	}
	if storage != &serializer.values[:cap(serializer.values)][0] {
		t.Fatal("Encode did not reuse the values buffer")
	}
}

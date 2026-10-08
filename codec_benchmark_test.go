package codec

import "testing"

type benchmarkRecord struct {
	Name    string
	Count   int
	Enabled bool
	Payload []byte
	Values  []int
}

var (
	benchmarkEncoded []byte
	benchmarkDecoded any
)

func benchmarkGraphValue() any {
	node := &graphNode{Value: 42}
	node.Next = node
	node.Prev = node
	return graphMixed{
		Node:  node,
		Items: []any{node, []byte{1, 2, 3, 4}},
		Table: map[string]any{"node": node, "value": 99},
		Box:   node,
	}
}

func BenchmarkCodec(b *testing.B) {
	payload := make([]byte, 64<<10)
	for i := range payload {
		payload[i] = byte(i)
	}
	largeRecord := benchmarkRecord{
		Name:    "large payload",
		Count:   len(payload),
		Enabled: true,
		Payload: payload,
		Values:  []int{1, 2, 3, 5, 8, 13, 21},
	}
	smallRecord := benchmarkRecord{
		Name:    "small payload",
		Count:   7,
		Enabled: true,
		Payload: []byte{1, 2, 3, 4, 5, 6, 7},
		Values:  []int{1, 2, 3, 5, 8, 13, 21},
	}
	items := []struct {
		name  string
		value any
	}{
		{"Scalar", int64(123456789)},
		{"Struct", smallRecord},
		{"ByteSlice64KiB", payload},
		{"Map", map[string][]int{
			"fibonacci": {1, 2, 3, 5, 8, 13, 21},
			"powers":    {1, 2, 4, 8, 16, 32},
		}},
		{"GraphWithAliases", benchmarkGraphValue()},
		{"LargeStruct", largeRecord},
	}

	b.Run("Encode", func(b *testing.B) {
		for _, item := range items {
			b.Run(item.name, func(b *testing.B) {
				serializer := NewSerializer()
				encoded := serializer.Encode(item.value)
				b.SetBytes(int64(len(encoded)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					benchmarkEncoded = serializer.Encode(item.value)
				}
			})
		}
	})

	b.Run("Decode", func(b *testing.B) {
		for _, item := range items {
			b.Run(item.name, func(b *testing.B) {
				data := Serialize(item.value)
				unserializer := NewUnserializer()
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					value, err := unserializer.Decode(data)
					if err != nil {
						b.Fatal(err)
					}
					benchmarkDecoded = value
				}
			})
		}
	})

	b.Run("RoundTrip", func(b *testing.B) {
		for _, item := range items {
			b.Run(item.name, func(b *testing.B) {
				serializer := NewSerializer()
				unserializer := NewUnserializer()
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					data := serializer.Encode(item.value)
					value, err := unserializer.Decode(data)
					if err != nil {
						b.Fatal(err)
					}
					benchmarkDecoded = value
				}
			})
		}
	})
}

/*
=== RUN   BenchmarkCodec/Encode
BenchmarkCodec/Encode

=== RUN   BenchmarkCodec/Encode/Scalar
BenchmarkCodec/Encode/Scalar
BenchmarkCodec/Encode/Scalar-16                           4173872            271.7 ns/op          22.08 MB/s       24 B/op           3 allocs/op
=== RUN   BenchmarkCodec/Encode/Struct
BenchmarkCodec/Encode/Struct
BenchmarkCodec/Encode/Struct-16                           150823             8238 ns/op           5.10 MB/s        1944 B/op         73 allocs/op
=== RUN   BenchmarkCodec/Encode/ByteSlice64KiB
BenchmarkCodec/Encode/ByteSlice64KiB
BenchmarkCodec/Encode/ByteSlice64KiB-16                   21                 51103028 ns/op       1.28 MB/s        4888253 B/op      131116 allocs/op
=== RUN   BenchmarkCodec/Encode/Map
BenchmarkCodec/Encode/Map
BenchmarkCodec/Encode/Map-16                              140082             7352 ns/op           6.26 MB/s        1801 B/op         67 allocs/op
=== RUN   BenchmarkCodec/Encode/GraphWithAliases
BenchmarkCodec/Encode/GraphWithAliases
BenchmarkCodec/Encode/GraphWithAliases-16                 115999             10170 ns/op          5.80 MB/s        2816 B/op         104 allocs/op

=== RUN   BenchmarkCodec/Decode
BenchmarkCodec/Decode

OPTIMIZED
=== RUN   BenchmarkCodec/Decode/Scalar
BenchmarkCodec/Decode/Scalar
BenchmarkCodec/Decode/Scalar-16                          5956699               178.6 ns/op        33.60 MB/s          16 B/op              2 allocs/op
=== RUN   BenchmarkCodec/Decode/Scalar
BenchmarkCodec/Decode/Scalar
BenchmarkCodec/Decode/Scalar-16                          7429437               151.8 ns/op        39.52 MB/s          40 B/op              3 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/Decode/Struct
BenchmarkCodec/Decode/Struct
BenchmarkCodec/Decode/Struct-16                           771172              1394 ns/op          30.13 MB/s         296 B/op              7 allocs/op
=== RUN   BenchmarkCodec/Decode/Struct
BenchmarkCodec/Decode/Struct
BenchmarkCodec/Decode/Struct-16                           556574              2201 ns/op          19.53 MB/s        3728 B/op             14 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/Decode/ByteSlice64KiB
BenchmarkCodec/Decode/ByteSlice64KiB
BenchmarkCodec/Decode/ByteSlice64KiB-16                      415           2803381 ns/op          23.38 MB/s       65608 B/op              4 allocs/op
=== RUN   BenchmarkCodec/Decode/ByteSlice64KiB
BenchmarkCodec/Decode/ByteSlice64KiB
BenchmarkCodec/Decode/ByteSlice64KiB-16                       75          13405214 ns/op           4.89 MB/s    16468039 B/op             33 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/Decode/Map
BenchmarkCodec/Decode/Map
BenchmarkCodec/Decode/Map-16                              759634              1529 ns/op          30.09 MB/s         664 B/op             13 allocs/op
=== RUN   BenchmarkCodec/Decode/Map
BenchmarkCodec/Decode/Map
BenchmarkCodec/Decode/Map-16                              466612              2187 ns/op          21.49 MB/s        2304 B/op             19 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/Decode/GraphWithAliases
BenchmarkCodec/Decode/GraphWithAliases
BenchmarkCodec/Decode/GraphWithAliases-16                 288762              3593 ns/op          16.42 MB/s        1144 B/op             31 allocs/op
=== RUN   BenchmarkCodec/Decode/GraphWithAliases
BenchmarkCodec/Decode/GraphWithAliases
BenchmarkCodec/Decode/GraphWithAliases-16                 288790              4125 ns/op          14.06 MB/s        4528 B/op             37 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/Decode/LargeStruct
BenchmarkCodec/Decode/LargeStruct
BenchmarkCodec/Decode/LargeStruct-16                         379           2742096 ns/op          23.91 MB/s       65824 B/op              7 allocs/op
=== RUN   BenchmarkCodec/Decode/LargeStruct
BenchmarkCodec/Decode/LargeStruct
BenchmarkCodec/Decode/LargeStruct-16                         100          13581276 ns/op           4.83 MB/s    16468282 B/op             36 allocs/op

=== RUN   BenchmarkCodec/RoundTrip
BenchmarkCodec/RoundTrip

OPTIMIZED
=== RUN   BenchmarkCodec/RoundTrip/Scalar
BenchmarkCodec/RoundTrip/Scalar
BenchmarkCodec/RoundTrip/Scalar-16                       2438754               501.7 ns/op            40 B/op          5 allocs/op
=== RUN   BenchmarkCodec/RoundTrip/Scalar
BenchmarkCodec/RoundTrip/Scalar
BenchmarkCodec/RoundTrip/Scalar-16                       2397532               507.2 ns/op            64 B/op          6 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/RoundTrip/Struct
BenchmarkCodec/RoundTrip/Struct
BenchmarkCodec/RoundTrip/Struct-16                        120147              9972 ns/op            2240 B/op         80 allocs/op
=== RUN   BenchmarkCodec/RoundTrip/Struct
BenchmarkCodec/RoundTrip/Struct
BenchmarkCodec/RoundTrip/Struct-16                        110139             11477 ns/op            5672 B/op         87 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/RoundTrip/ByteSlice64KiB
BenchmarkCodec/RoundTrip/ByteSlice64KiB
BenchmarkCodec/RoundTrip/ByteSlice64KiB-16                    21          56483327 ns/op         8132948 B/op     131296 allocs/op
=== RUN   BenchmarkCodec/RoundTrip/ByteSlice64KiB
BenchmarkCodec/RoundTrip/ByteSlice64KiB
BenchmarkCodec/RoundTrip/ByteSlice64KiB-16                    18          64934525 ns/op        25065484 B/op     131355 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/RoundTrip/Map
BenchmarkCodec/RoundTrip/Map
BenchmarkCodec/RoundTrip/Map-16                           133342              9099 ns/op            2466 B/op         80 allocs/op
=== RUN   BenchmarkCodec/RoundTrip/Map
BenchmarkCodec/RoundTrip/Map
BenchmarkCodec/RoundTrip/Map-16                           129178              9637 ns/op            4107 B/op         86 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/RoundTrip/GraphWithAliases
BenchmarkCodec/RoundTrip/GraphWithAliases
BenchmarkCodec/RoundTrip/GraphWithAliases-16               84841             14580 ns/op            3961 B/op        135 allocs/op
=== RUN   BenchmarkCodec/RoundTrip/GraphWithAliases
BenchmarkCodec/RoundTrip/GraphWithAliases
BenchmarkCodec/RoundTrip/GraphWithAliases-16               81938             15125 ns/op            7345 B/op        141 allocs/op

OPTIMIZED
=== RUN   BenchmarkCodec/RoundTrip/LargeStruct
BenchmarkCodec/RoundTrip/LargeStruct
BenchmarkCodec/RoundTrip/LargeStruct-16                       21          55043844 ns/op         8134472 B/op     131346 allocs/op
=== RUN   BenchmarkCodec/RoundTrip/LargeStruct
BenchmarkCodec/RoundTrip/LargeStruct
BenchmarkCodec/RoundTrip/LargeStruct-16                       18          63579910 ns/op        25067006 B/op     131405 allocs/op

26.355s
*/

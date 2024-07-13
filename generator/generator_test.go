package main

import "testing"

func BenchmarkMain(b *testing.B) {
	b.ReportAllocs()
	args = []string{}
	for i := 0; i < b.N; i++ {
		main()
	}
}

func BenchmarkFormatTemp(b *testing.B) {
	b.ReportAllocs()

	var res []byte
	for i := 0; i < b.N; i++ {
		res = formatTemp(-906)
	}
	_ = res
}

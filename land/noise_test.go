// Copyright © 2014-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package land

import (
	"testing"
)

func TestSameSeed(t *testing.T) {
	n0 := newNoise(123)
	n1 := newNoise(123)
	if n0.generate2D(0, 0) != n1.generate2D(0, 0) {
		t.Error("Two generators with the same seed should produce the same result")
	}
}

// ============================================================================
// benchmarks : go test -bench .
//
// A previous run show that generating a single noise value is relatively quick.
// Of course a single map may need multiple 256x256 noise tiles.
//   BenchmarkNoise2D-8      100000000           18.7 ns/op  using noise.floor()
//   BenchmarkNoise3D-8      50000000            32.8 ns/op  using noise.floor()
//
//   BenchmarkNoise2D-8      50000000            28.7 ns/op  using math.Floor()
//   BenchmarkNoise3D-8      30000000            44.3 ns/op  using math.Floor()

// How long does creating a single random noise value take.
func BenchmarkNoise2D(b *testing.B) {
	n := newNoise(123)
	for cnt := 0; cnt < b.N; cnt++ {
		n.generate2D(10, 10)
	}
}

func BenchmarkNoise3D(b *testing.B) {
	n := newNoise(123)
	for cnt := 0; cnt < b.N; cnt++ {
		n.generate3D(10, 10, 10)
	}
}

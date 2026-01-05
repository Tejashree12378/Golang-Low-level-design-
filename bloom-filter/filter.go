package filter

import (
	"sync/atomic"

	"github.com/spaolacci/murmur3"
)

type BloomFilter interface {
	Add(value string) bool
	MightContain(value string) bool
}

type filter struct {
	bitArray           []uint64
	numOfHashFunctions int
	size               int
}

func NewFilter(maxLen int, k int) BloomFilter {
	return &filter{
		bitArray:           make([]uint64, (maxLen+63)/64),
		numOfHashFunctions: k,
		size:               maxLen,
	}
}

func (f *filter) Add(value string) bool {
	h1, h2 := doubleHashes(value)
	bitSize := uint64(f.size)

	for i := 0; i < f.numOfHashFunctions; i++ {
		combined := h1 + uint64(i)*h2
		bit := combined % bitSize

		wordIndex := bit >> 6
		bitIndex := bit & 63
		mask := uint64(1) << bitIndex

		atomic.OrUint64(&f.bitArray[wordIndex], mask)
	}
	return true
}

func doubleHashes(data string) (uint64, uint64) {
	h1 := murmur3.Sum64([]byte(data))
	h2 := murmur3.Sum64WithSeed([]byte(data), 0x5bd1e995)
	return h1, h2
}

func (f *filter) MightContain(value string) bool {
	h1, h2 := doubleHashes(value)
	bitSize := uint64(f.size)

	for i := 0; i < f.numOfHashFunctions; i++ {
		combined := h1 + uint64(i)*h2
		bit := combined % bitSize

		wordIndex := bit >> 6
		bitIndex := bit & 63
		mask := uint64(1) << bitIndex

		word := atomic.LoadUint64(&f.bitArray[wordIndex])
		if word&mask == 0 {
			return false
		}
	}
	return true
}

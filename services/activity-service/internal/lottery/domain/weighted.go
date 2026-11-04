package domain

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sort"
)

var ErrInvalidWeights = errors.New("invalid lottery weights")

// WeightedPool is an immutable distribution over the configured prize indexes.
type WeightedPool struct {
	prefix []int64
	total  int64
}

func NewWeightedPool(weights []int64) (*WeightedPool, error) {
	p := &WeightedPool{prefix: make([]int64, len(weights))}
	for i, weight := range weights {
		if weight < 0 || weight > math.MaxInt64-p.total {
			return nil, ErrInvalidWeights
		}
		p.total += weight
		p.prefix[i] = p.total
	}
	if p.total == 0 {
		return nil, ErrInvalidWeights
	}
	return p, nil
}

// Select maps a uniform sample in [0,total) to its prize index.
func (p *WeightedPool) Select(sample int64) (int, error) {
	if p == nil || sample < 0 || sample >= p.total {
		return 0, errors.New("lottery sample out of range")
	}
	return sort.Search(len(p.prefix), func(i int) bool { return p.prefix[i] > sample }), nil
}

// Draw samples using the operating system's cryptographic random source.
func (p *WeightedPool) Draw() (int, error) {
	if p == nil || p.total <= 0 {
		return 0, ErrInvalidWeights
	}
	r, err := rand.Int(rand.Reader, big.NewInt(p.total))
	if err != nil {
		return 0, err
	}
	return p.Select(r.Int64())
}

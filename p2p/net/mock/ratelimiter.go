package test_mocknet

import (
	"sync"
	"time"
)

type RateLimiter struct {
	lock         sync.Mutex
	bandwidth    float64
	allowance    float64
	maxAllowance float64
	lastUpdate   time.Time
	count        int
	duration     time.Duration
}

func NewRateLimiter(bandwidth float64) *RateLimiter {
	b := bandwidth / float64(time.Second)
	return &RateLimiter{
		bandwidth:    b,
		allowance:    0,
		maxAllowance: bandwidth,
		lastUpdate:   time.Now(),
	}
}

// bandwidth 是1秒钟的速率
func (r *RateLimiter) UpdateBandwidth(bandwidth float64) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// time.Second = 1e+09
	b := bandwidth / float64(time.Second)
	r.bandwidth = b

	r.allowance = 0
	r.maxAllowance = bandwidth
	r.lastUpdate = time.Now()
}

func (r *RateLimiter) Limit(dataSize int) time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	var duration time.Duration = time.Duration(0)
	if r.bandwidth == 0 {
		return duration
	}

	// 现在离上一次调整距离多长时间
	current := time.Now()
	elapsedTime := current.Sub(r.lastUpdate)
	r.lastUpdate = current

	// 允许通过多少字节的数据
	allowance := r.allowance + float64(elapsedTime)*r.bandwidth
	if allowance > r.maxAllowance {
		allowance = r.maxAllowance
	}

	allowance -= float64(dataSize)
	if allowance < 0 {
		duration = time.Duration(-allowance / r.bandwidth)
		r.count++
		r.duration += duration
	}

	r.allowance = allowance
	return duration
}

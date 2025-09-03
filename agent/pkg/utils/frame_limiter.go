package utils

import (
	"time"
)

// FrameLimiter 帧率限制器，用于控制视频流的帧率
type FrameLimiter struct {
	DesiredFps  int
	frameTimeNs int64

	LastFrameTime     time.Time
	LastSleepDuration time.Duration

	DidSleep bool
	DidSpin  bool
}

// NewFrameLimiter 创建新的帧率限制器
func NewFrameLimiter(desiredFps int) *FrameLimiter {
	return &FrameLimiter{
		DesiredFps:    desiredFps,
		frameTimeNs:   (time.Second / time.Duration(desiredFps)).Nanoseconds(),
		LastFrameTime: time.Now(),
	}
}

// Wait 等待到下一帧的时间
func (l *FrameLimiter) Wait() {
	l.DidSleep = false
	l.DidSpin = false

	now := time.Now()
	spinWaitUntil := now

	sleepTime := l.frameTimeNs - now.Sub(l.LastFrameTime).Nanoseconds()

	if sleepTime > int64(1*time.Millisecond) {
		if sleepTime < int64(30*time.Millisecond) {
			l.LastSleepDuration = time.Duration(sleepTime / 8)
		} else {
			l.LastSleepDuration = time.Duration(sleepTime / 4 * 3)
		}
		time.Sleep(time.Duration(l.LastSleepDuration))
		l.DidSleep = true

		newNow := time.Now()
		spinWaitUntil = newNow.Add(time.Duration(sleepTime) - newNow.Sub(now))
		now = newNow

		for spinWaitUntil.After(now) {
			now = time.Now()
			// SPIN WAIT
			l.DidSpin = true
		}
	} else {
		l.LastSleepDuration = 0
		spinWaitUntil = now.Add(time.Duration(sleepTime))
		for spinWaitUntil.After(now) {
			now = time.Now()
			// SPIN WAIT
			l.DidSpin = true
		}
	}

	l.LastFrameTime = time.Now()
}

// GetStats 获取帧率限制器统计信息
func (l *FrameLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"desired_fps":        l.DesiredFps,
		"frame_time_ns":      l.frameTimeNs,
		"last_sleep_duration": l.LastSleepDuration,
		"did_sleep":          l.DidSleep,
		"did_spin":           l.DidSpin,
	}
}

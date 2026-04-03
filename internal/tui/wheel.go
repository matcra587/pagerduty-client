package tui

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// wheelBatchDelay is the debounce window for coalescing mouse wheel events.
// Raw wheel bursts are accumulated over this interval and flushed as a single
// scroll, preventing per-event Update + Render cycles from pegging CPU.
const wheelBatchDelay = 12 * time.Millisecond

// batchedScrollMsg carries an accumulated scroll delta. Positive = down.
type batchedScrollMsg struct{ delta int }

// wheelCoalescer intercepts mouse wheel events via a Bubble Tea filter,
// accumulates deltas and flushes them as a single batchedScrollMsg after
// a short debounce window. The flush happens outside the Bubble Tea
// update loop via time.AfterFunc, so burst events never trigger renders.
type wheelCoalescer struct {
	send  func(tea.Msg)
	delay time.Duration

	mu     sync.Mutex
	active bool
	delta  int
	timer  *time.Timer
}

// NewWheelCoalescer returns a coalescer that flushes via send.
// Call Stop when the program exits.
func NewWheelCoalescer(send func(tea.Msg)) *wheelCoalescer {
	return &wheelCoalescer{send: send, delay: wheelBatchDelay}
}

// Filter is a tea.WithFilter-compatible function. It swallows mouse wheel
// events directed at a scrollable view, accumulates their delta and returns
// nil so Bubble Tea skips the normal Update + Render cycle. Non-wheel
// messages pass through unchanged.
func (c *wheelCoalescer) Filter(model tea.Model, msg tea.Msg) tea.Msg {
	wm, ok := msg.(tea.MouseWheelMsg)
	if !ok {
		return msg
	}

	app, ok := model.(App)
	if !ok || app.current != viewDetail {
		return msg
	}

	var d int
	switch wm.Button {
	case tea.MouseWheelDown:
		d = 3
	case tea.MouseWheelUp:
		d = -3
	default:
		return msg
	}

	c.enqueue(d)
	return nil
}

func (c *wheelCoalescer) enqueue(d int) {
	c.mu.Lock()
	c.delta += d
	if !c.active {
		c.active = true
		c.mu.Unlock()
		c.scheduleFlush()
		return
	}
	c.mu.Unlock()
}

func (c *wheelCoalescer) scheduleFlush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = time.AfterFunc(c.delay, c.flush)
}

func (c *wheelCoalescer) flush() {
	c.mu.Lock()
	d := c.delta
	c.delta = 0
	c.active = false
	c.timer = nil
	c.mu.Unlock()

	if d == 0 || c.send == nil {
		return
	}
	go c.send(batchedScrollMsg{delta: d})
}

// Stop cancels any pending flush.
func (c *wheelCoalescer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.active = false
	c.delta = 0
}

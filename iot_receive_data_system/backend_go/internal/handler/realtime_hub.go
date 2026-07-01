package handler

import (
	"sync"

	"backend_go/internal/model"
)

type wardRealtimeHub struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[int]map[uint64]chan model.WardSensorRealtimeEvent
}

func newWardRealtimeHub() *wardRealtimeHub {
	return &wardRealtimeHub{
		subscribers: make(map[int]map[uint64]chan model.WardSensorRealtimeEvent),
	}
}

func (h *wardRealtimeHub) subscribe(floor int) (uint64, <-chan model.WardSensorRealtimeEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID
	ch := make(chan model.WardSensorRealtimeEvent, 256)
	if h.subscribers[floor] == nil {
		h.subscribers[floor] = make(map[uint64]chan model.WardSensorRealtimeEvent)
	}
	h.subscribers[floor][id] = ch
	return id, ch
}

func (h *wardRealtimeHub) unsubscribe(floor int, id uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	floorSubs := h.subscribers[floor]
	if floorSubs == nil {
		return
	}
	if ch, ok := floorSubs[id]; ok {
		delete(floorSubs, id)
		close(ch)
	}
	if len(floorSubs) == 0 {
		delete(h.subscribers, floor)
	}
}

func (h *wardRealtimeHub) publish(event model.WardSensorRealtimeEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subscribers[event.Floor] {
		select {
		case ch <- event:
		default:
		}
	}
}

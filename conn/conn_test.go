package conn

import (
	"testing"
	"time"

	"github.com/user/minegate/packet"
)

func TestFlowControllerAcquireRelease(t *testing.T) {
	fc := NewFlowController(100, 1024)

	err := fc.Acquire(100)
	if err != nil {
		t.Fatal(err)
	}

	if fc.Used() != 100 {
		t.Errorf("Used = %d, want 100", fc.Used())
	}

	if fc.Available() != 924 {
		t.Errorf("Available = %d, want 924", fc.Available())
	}

	fc.Release(100)

	if fc.Used() != 0 {
		t.Errorf("Used after release = %d, want 0", fc.Used())
	}
}

func TestFlowControllerNonBlock(t *testing.T) {
	fc := NewFlowController(100, 100)

	err := fc.AcquireNonBlock(50)
	if err != nil {
		t.Fatal(err)
	}

	err = fc.AcquireNonBlock(60)
	if err == nil {
		t.Error("expected error for over-capacity acquire, got nil")
	}
}

func TestFlowControllerUtilization(t *testing.T) {
	fc := NewFlowController(10, 200)
	fc.Acquire(100)

	util := fc.Utilization()
	if util != 0.5 {
		t.Errorf("Utilization = %f, want 0.5", util)
	}

	fc.Release(100)
	if fc.Utilization() != 0 {
		t.Errorf("Utilization after release = %f, want 0", fc.Utilization())
	}
}

func TestDroppableQueue(t *testing.T) {
	dq := NewDroppableQueue(3)

	dq.Push(1)
	dq.Push(2)
	dq.Push(3)

	if dq.Len() != 3 {
		t.Errorf("Len = %d, want 3", dq.Len())
	}

	dq.Push(4) // should drop 1

	if dq.Len() != 3 {
		t.Errorf("Len after push = %d, want 3", dq.Len())
	}

	item, ok := dq.Pop()
	if !ok {
		t.Fatal("Pop should succeed")
	}
	if item != 2 {
		t.Errorf("Pop = %d, want 2", item)
	}

	dropped := dq.Dropped()
	if dropped != 1 {
		t.Errorf("Dropped = %d, want 1", dropped)
	}
}

func TestBoundedQueue(t *testing.T) {
	bq := NewBoundedQueue(2)

	ok := bq.TryPush(1)
	if !ok {
		t.Error("TryPush should succeed")
	}

	ok = bq.TryPush(2)
	if !ok {
		t.Error("TryPush should succeed")
	}

	ok = bq.TryPush(3)
	if ok {
		t.Error("TryPush should fail on full queue")
	}

	item, ok := bq.TryPop()
	if !ok {
		t.Fatal("TryPop should succeed")
	}
	if item != 1 {
		t.Errorf("Pop = %d, want 1", item)
	}
}

func TestBoundedQueueClose(t *testing.T) {
	bq := NewBoundedQueue(1)
	bq.Push(1)
	bq.Close()

	_, ok := bq.TryPop()
	if !ok {
		t.Error("should be able to pop after close")
	}
}

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue(3)

	pq.Push(&PriorityItem{Data: packet.Packet{ID: 1}, Priority: Low})
	pq.Push(&PriorityItem{Data: packet.Packet{ID: 2}, Priority: Urgent})
	pq.Push(&PriorityItem{Data: packet.Packet{ID: 3}, Priority: Normal})

	item, err := pq.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if item.Data.ID != 2 {
		t.Errorf("expected urgent (2), got %d", int32(item.Data.ID))
	}

	item, err = pq.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if item.Data.ID != 3 {
		t.Errorf("expected normal (3), got %d", int32(item.Data.ID))
	}

	item, err = pq.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if item.Data.ID != 1 {
		t.Errorf("expected low (1), got %d", int32(item.Data.ID))
	}
}

func TestPriorityQueueClose(t *testing.T) {
	pq := NewPriorityQueue(3)
	pq.Close()

	_, err := pq.Pop()
	if err == nil {
		t.Error("expected error on closed queue")
	}
}

func TestMetrics(t *testing.T) {
	m := &Metrics{}

	m.RecordRead(100)
	m.RecordWrite(200)

	if m.PacketsRead.Load() != 1 {
		t.Errorf("PacketsRead = %d, want 1", m.PacketsRead.Load())
	}
	if m.BytesRead.Load() != 100 {
		t.Errorf("BytesRead = %d, want 100", m.BytesRead.Load())
	}
	if m.PacketsWritten.Load() != 1 {
		t.Errorf("PacketsWritten = %d, want 1", m.PacketsWritten.Load())
	}
	if m.BytesWritten.Load() != 200 {
		t.Errorf("BytesWritten = %d, want 200", m.BytesWritten.Load())
	}
}

func TestMetricsLatency(t *testing.T) {
	m := &Metrics{}

	m.RecordLatency(10 * time.Millisecond)
	m.RecordLatency(20 * time.Millisecond)

	avg := m.AverageLatency()
	if avg == 0 {
		t.Error("average latency should not be zero")
	}

	snap := m.Snapshot()
	if snap.PacketsRead != 0 {
		t.Errorf("snapshot PacketsRead = %d, want 0", snap.PacketsRead)
	}
}

func TestMetricsReset(t *testing.T) {
	m := &Metrics{}
	m.RecordRead(100)
	m.RecordWrite(200)
	m.Reset()

	if m.PacketsRead.Load() != 0 {
		t.Error("metrics should be zero after reset")
	}
}

func TestConnMetrics(t *testing.T) {
	cm := NewConnMetrics()
	if cm.Inbound == nil || cm.Outbound == nil {
		t.Error("ConnMetrics should have Inbound and Outbound")
	}

	cm.Inbound.RecordRead(50)
	cm.Outbound.RecordWrite(100)

	if cm.Inbound.BytesRead.Load() != 50 {
		t.Errorf("Inbound bytes = %d, want 50", cm.Inbound.BytesRead.Load())
	}
}

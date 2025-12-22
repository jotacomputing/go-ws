package shm
import (
	"fmt"
	"os"
	"sync/atomic"
	"unsafe"
	"github.com/edsrzf/mmap-go"
)

type OrderEvent struct {
	UserId        uint64
	OrderId       uint64
	Symbol        uint32
	EventKind     uint32 // 0..4

	FilledQty     uint32
	RemainingQty  uint32
	OriginalQty   uint32

	ErrorCode     uint32
}


type OrderEventQueueHeader struct {
	ProducerHead uint64
	_pad1        [56]byte
	ConsumerTail uint64
	_pad2        [56]byte
	Magic        uint32
	Capacity     uint32
}


const (
	OrderEventQueueMagic    = 0xEAAAAAAC
	OrderEventQueueCapacity = 65536
	OrderEventSize       = unsafe.Sizeof(OrderEvent{})
	OrderEventHeaderSize = unsafe.Sizeof(OrderEventQueueHeader{})
	OrderEventTotalSize  = OrderEventHeaderSize +
		(OrderEventQueueCapacity * OrderEventSize)
)


type OrderEventQueue struct {
	file   *os.File
	mmap   mmap.MMap
	header *OrderEventQueueHeader
	events []OrderEvent
}


func CreateOrderEventQueue(path string) (*OrderEventQueue, error) {
	_ = os.Remove(path)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return nil, err
	}

	if err := file.Truncate(int64(OrderEventTotalSize)); err != nil {
		file.Close()
		return nil, err
	}

	if err := file.Sync(); err != nil {
		file.Close()
		return nil, err
	}

	m, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	_ = m.Lock()

	header := (*OrderEventQueueHeader)(unsafe.Pointer(&m[0]))
	atomic.StoreUint64(&header.ProducerHead, 0)
	atomic.StoreUint64(&header.ConsumerTail, 0)
	atomic.StoreUint32(&header.Magic, OrderEventQueueMagic)
	atomic.StoreUint32(&header.Capacity, OrderEventQueueCapacity)

	data := m[OrderEventHeaderSize:OrderEventTotalSize]
	events := unsafe.Slice(
		(*OrderEvent)(unsafe.Pointer(&data[0])),
		OrderEventQueueCapacity,
	)

	return &OrderEventQueue{
		file:   file,
		mmap:   m,
		header: header,
		events: events,
	}, nil
}

func OpenOrderEventQueue(path string) (*OrderEventQueue, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if stat.Size() != int64(OrderEventTotalSize) {
		file.Close()
		return nil, fmt.Errorf("order event queue size mismatch")
	}

	m, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	_ = m.Lock()

	header := (*OrderEventQueueHeader)(unsafe.Pointer(&m[0]))
	if atomic.LoadUint32(&header.Magic) != OrderEventQueueMagic {
		return nil, fmt.Errorf("invalid order event queue magic")
	}

	data := m[OrderEventHeaderSize:OrderEventTotalSize]
	events := unsafe.Slice(
		(*OrderEvent)(unsafe.Pointer(&data[0])),
		OrderEventQueueCapacity,
	)

	return &OrderEventQueue{
		file:   file,
		mmap:   m,
		header: header,
		events: events,
	}, nil
}

func (q *OrderEventQueue) Enqueue(ev OrderEvent) error {
	consumer := atomic.LoadUint64(&q.header.ConsumerTail)
	producer := atomic.LoadUint64(&q.header.ProducerHead)

	if producer-consumer >= OrderEventQueueCapacity {
		return fmt.Errorf("order event queue full")
	}

	pos := producer % OrderEventQueueCapacity
	q.events[pos] = ev

	atomic.StoreUint64(&q.header.ProducerHead, producer+1)
	return nil
}

func (q *OrderEventQueue) Dequeue() (*OrderEvent, error) {
	producer := atomic.LoadUint64(&q.header.ProducerHead)
	consumer := atomic.LoadUint64(&q.header.ConsumerTail)

	if consumer == producer {
		return nil, nil
	}

	pos := consumer % OrderEventQueueCapacity
	ev := q.events[pos]

	atomic.StoreUint64(&q.header.ConsumerTail, consumer+1)
	return &ev, nil
}

func (q *OrderEventQueue) Depth() uint64 {
	return atomic.LoadUint64(&q.header.ProducerHead) -
		atomic.LoadUint64(&q.header.ConsumerTail)
}

func (q *OrderEventQueue) Flush() error {
	return q.mmap.Flush()
}

func (q *OrderEventQueue) Close() error {
	_ = q.mmap.Flush()
	_ = q.mmap.Unlock()
	_ = q.mmap.Unmap()
	return q.file.Close()
}

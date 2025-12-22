package shm
import (
	"fmt"
	"os"
	"sync/atomic"
	"unsafe"
	"github.com/edsrzf/mmap-go"
)

type Query struct {
	// All uint64s first (8-byte aligned)
	QueryId uint64
	UserId  uint64
	QueryType QueryType // 4 
	_ [4]byte
}

type QueryQueueHeader struct {
	ProducerHead uint64   // Offset 0 4 byte interger 
	_        [56]byte // Padding to cache line
	ConsumerTail uint64   // Offset 64
	_      [56]byte // Padding
	Magic        uint32   // Offset 128
	Capacity     uint32   // Offset 132
}

const (
	QueryQueueMagic    = 0x51554552 
	QueryQueueCapacity = 65536
	QuerySize       = unsafe.Sizeof(Query{})
	QueryHeaderSize = unsafe.Sizeof(QueryQueueHeader{})
	QueryTotalSize  = QueryHeaderSize + (QueryQueueCapacity * QuerySize)
)


type QueryQueue struct {
	file   *os.File
	mmap   mmap.MMap   // this is the array of bytes wich we will use to read and write 
	header *QueryQueueHeader
	queries []Query
}

func CreateQueryQueue(path string) (*QueryQueue, error) {
	_ = os.Remove(path)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return nil, err
	}

	if err := file.Truncate(int64(QueryTotalSize)); err != nil {
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

	header := (*QueryQueueHeader)(unsafe.Pointer(&m[0]))
	atomic.StoreUint64(&header.ProducerHead, 0)
	atomic.StoreUint64(&header.ConsumerTail, 0)
	atomic.StoreUint32(&header.Magic, QueryQueueMagic)
	atomic.StoreUint32(&header.Capacity, QueryQueueCapacity)

	data := m[QueryHeaderSize:QueryTotalSize]
	queries := unsafe.Slice(
		(*Query)(unsafe.Pointer(&data[0])),
		QueryQueueCapacity,
	)

	return &QueryQueue{
		file:    file,
		mmap:    m,
		header:  header,
		queries: queries,
	}, nil
}

// open queue from file on disk and return *Queue mmap-ed
func OpenQueryQueue(path string) (*QueryQueue, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if stat.Size() != int64(QueryTotalSize) {
		file.Close()
		return nil, fmt.Errorf("query queue size mismatch")
	}

	m, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	_ = m.Lock()

	header := (*QueryQueueHeader)(unsafe.Pointer(&m[0]))
	if atomic.LoadUint32(&header.Magic) != QueryQueueMagic {
		return nil, fmt.Errorf("invalid query queue magic")
	}

	data := m[QueryHeaderSize:QueryTotalSize]
	queries := unsafe.Slice(
		(*Query)(unsafe.Pointer(&data[0])),
		QueryQueueCapacity,
	)

	return &QueryQueue{
		file:    file,
		mmap:    m,
		header:  header,
		queries: queries,
	}, nil
}


func (q *QueryQueue) Enqueue(query Query) error {
	consumer := atomic.LoadUint64(&q.header.ConsumerTail)
	producer := atomic.LoadUint64(&q.header.ProducerHead)

	if producer-consumer >= QueryQueueCapacity {
		return fmt.Errorf("query queue full")
	}

	pos := producer % QueryQueueCapacity
	q.queries[pos] = query

	atomic.StoreUint64(&q.header.ProducerHead, producer+1)
	return nil
}


func (q *QueryQueue) Dequeue() (*Query, error) {
	producer := atomic.LoadUint64(&q.header.ProducerHead)
	consumer := atomic.LoadUint64(&q.header.ConsumerTail)

	if consumer == producer {
		return nil, nil
	}

	pos := consumer % QueryQueueCapacity
	query := q.queries[pos]

	atomic.StoreUint64(&q.header.ConsumerTail, consumer+1)
	return &query, nil
}


func (q *QueryQueue) Depth() uint64 {
	return atomic.LoadUint64(&q.header.ProducerHead) -
		atomic.LoadUint64(&q.header.ConsumerTail)
}

func (q *QueryQueue) Capacity() uint64 {
	return QueryQueueCapacity
}

func (q *QueryQueue) Flush() error {
	return q.mmap.Flush()
}

func (q *QueryQueue) Close() error {
	_ = q.mmap.Flush()
	_ = q.mmap.Unlock()
	_ = q.mmap.Unmap()
	return q.file.Close()
}

package shm

type QueryType uint32
const (
	QueryGetBalance QueryType = iota
	QueryGetHoldings
	QueryAddUser
)


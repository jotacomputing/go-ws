package main

import (
	pubsubmanager "exchange/PubSubManager"
	symbolmanager "exchange/SymbolManager"
	ws "exchange/Ws"
	shm "exchange/Shm"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	balance_Response_queue , berr := shm.OpenBalanceResponseQueue("/tmp/trading/BalanceResponseQueue")
	if berr!=nil{

	}
	cancel_order_queue , cerr := shm.OpenCancelOrderQueue("/tmp/trading/CancelOrders")
	if cerr!=nil{
		
	}
	holdings_response_queue , herr := shm.OpenHoldingResponseQueue("/tmp/trading/HoldingResponse")
	if herr!=nil{
		
	}
	order_events_queue , oerr := shm.OpenOrderEventQueue("/tmp/trading/OrderEvents")
	if oerr!=nil{
		
	}
	post_order_queue , qerr := shm.OpenQueue("/tmp/trading/PostOrders")
	if qerr!=nil{
		
	}
	queries_queue , querr := shm.OpenQueryQueue("/tmp/trading/Queries")
	if querr!=nil{
		
	}

	
	sm := symbolmanager.CreateSymbolManagerSingleton()
	pubsubm := pubsubmanager.CreateSingletonInstance(sm)
	sm.Subscriber = pubsubm
	sm.Unsubscriber = pubsubm
	go sm.StartSymbolMnagaer()

	wsServer := ws.NewServer(sm)
	// do we need go rotine here ?? idts the server can run in the main routine
	go wsServer.CreateServer()

	shmmanager:= shm.ShmManager{
		Balance_Response_queue: balance_Response_queue,
		CancelOrderQueue: cancel_order_queue,
		Holding_Response_queue: holdings_response_queue,
		Order_Events_queue: order_events_queue,
		Post_Order_queue: post_order_queue,
		Query_queue: queries_queue,
	}
	go shmmanager.PollOrderEvents()
	go shmmanager.PollQueryResponse()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Shutting down gracefully...")
}

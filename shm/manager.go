package shm


type ShmManager struct{
	Balance_Response_queue 	*BalanceResponseQueue
	CancelOrderQueue 	   	*CancelOrderQueue
	Holding_Response_queue	*HoldingResponseQueue
	Order_Events_queue		*OrderEventQueue
	Post_Order_queue		*Queue
	Query_queue				*QueryQueue
}

func(m*ShmManager)init(Balance_Response_queue *BalanceResponseQueue , 
	CancelOrderQueue *CancelOrderQueue , 
	Holding_Response_queue	*HoldingResponseQueue,
	Order_Evenets_queue		*OrderEventQueue,
	Post_Order_queue		*Queue , 
	Query_queue				*QueryQueue,
	)*ShmManager{
		return &ShmManager{
			Balance_Response_queue: Balance_Response_queue,
			CancelOrderQueue: CancelOrderQueue,
			Holding_Response_queue: Holding_Response_queue,
			Order_Events_queue: Order_Evenets_queue,
			Post_Order_queue: Post_Order_queue,
			Query_queue: Query_queue,
		}
}

// function to launch go routines to poll the order events and the query response queue 

func(m*ShmManager)PollOrderEvents(){
	
}

func(m*ShmManager)PollQueryResponse(){

}
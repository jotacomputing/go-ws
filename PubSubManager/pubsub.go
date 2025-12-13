package pubsubmanager

import(
	 "sync"
	 "github.com/redis/go-redis/v9"
	  "exchange/Contracts"
	  "context"
	  "encoding/json"
	  "fmt"
	)

// the pubsusb manager exposes the subscribe , unsubscribe methods , initiates the redis pubsub clietn
var PubSubManagerInstance *PubSubManager
var once sync.Once

type  PubSubManager struct{
	rclient 		*redis.Client 
	BroadCaster 	contracts.BroadCaster
	subscriptions 	map[string]*redis.PubSub // keeps a track of what all streams are we subscribed to 
	mu 				sync.Mutex
}


func CreateSingletonInstance(broadcaster contracts.BroadCaster) *PubSubManager{
	once.Do(func(){
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		if err := client.Ping(context.Background()).Err(); err != nil {
			panic(err)
		}

		PubSubManagerInstance = &PubSubManager{
			rclient: client,
			BroadCaster: broadcaster,
			subscriptions:  make(map[string]*redis.PubSub),
		}
	})

	return PubSubManagerInstance
}

func getPubSubManagerInstance() *PubSubManager{
	return PubSubManagerInstance
}


func (ps *PubSubManager)SubscribeToSymbolMethod(StreamName string){
	ps.mu.Lock()
	if _, already := ps.subscriptions[StreamName]; already {
		ps.mu.Unlock()
		return
	}
	// if not subscibed
	pubsub := ps.rclient.Subscribe(context.Background(), StreamName)
	ps.subscriptions[StreamName] = pubsub
	ps.mu.Unlock()

	
	ch := pubsub.Channel()
	// this is the reciver go routine for every stream 
	go func() {
		for msg := range ch {
			var m contracts.MessageFromPubSubForUser
			if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
				fmt.Println("Redis message unmarshal error:", err)
				continue
			}
			// Notify RoomManager (via Broadcaster interface)
			ps.BroadCaster.BroadCasteFromRemote(m)
		}
	}()
}


func (ps *PubSubManager) UnSubscribeToSymbolMethod(StreamName string) {
    ps.mu.Lock()
	// take out the pubsusb obj
    pubsub, exists := ps.subscriptions[StreamName]
    if !exists {
        ps.mu.Unlock()
        return 
    }
    delete(ps.subscriptions, StreamName)
    ps.mu.Unlock()

    if err := pubsub.Unsubscribe(context.Background(), StreamName); err != nil {
        fmt.Println("Error unsubscribing:", err)
    }
	// close pubusb
    if err := pubsub.Close(); err != nil {
        fmt.Println("Error closing pubsub:", err)
    }
}

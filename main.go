package main

import (
	pubsubmanager "exchange/PubSubManager"
	symbolmanager "exchange/SymbolManager"
	"exchange/ws"
)
func main(){
	
	sm := symbolmanager.CreateSymbolManagerSingleton()
	pubsubm := pubsubmanager.CreateSingletonInstance(sm)
	sm.Subscriber = pubsubm
	sm.Unsubscriber = pubsubm
	go ws.CreateServer()
	go sm.StartSymbolMnagaer()

	select {} // block forever
}

package main

import (
	"fmt"
	"time"

	"github.com/kelindar/event"
)

// Various event types
const EventA = 0x01

// Event type for testing purposes
type Event struct {
	Data string
}

// Type returns the event type
func (ev Event) Type() uint32 {
	return EventA
}

// newEventA creates a new instance of an event
func newEventA(data string) Event {
	return Event{Data: data}
}

func main() {
	bus := event.NewDispatcher()
	// bus.Close()

	// Subcribe to event A, and automatically unsubscribe at the end
	defer event.SubscribeTo(bus, EventA, func(e Event) {
		println("(consumer 1)", e.Data)
	})()

	// Subcribe to event A, and automatically unsubscribe at the end
	unsub := event.SubscribeTo(bus, EventA, func(e Event) {
		println("(consumer 2)", e.Data)
	})

	// Publish few events

	time.AfterFunc(time.Second*5, func() {
		unsub()
	})

	go func() {
		// after := time.After(time.Second * 5)

		tk := time.NewTicker(time.Second)
		// defer tk.Stop()

		for range tk.C {
			fmt.Println("publishing event 4")
			event.Publish(bus, newEventA("event 4"))
		}
	}()

	event.Publish(bus, newEventA("event 1"))
	event.Publish(bus, newEventA("event 2"))
	event.Publish(bus, newEventA("event 3"))

	time.Sleep(50 * time.Second)
}

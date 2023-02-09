package main

import "fmt"

func main() {
	fmt.Println("this ran")
}

func processTransaction() {
	// Process incoming transaction from pubsub

	// 1. Read transaction from pubsub
	// 2. Read account balance from transaction
	// 3. Push account balance to firestore
}

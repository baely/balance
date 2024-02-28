package main

import "fmt"

func main() {
	s := newServer()
	fmt.Println("Starting server")
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}

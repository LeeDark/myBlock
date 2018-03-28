package main

import (
	"fmt"
	"sync"
	"testing"
)

var (
	input1, input2 chan string
	done1, done2   chan bool
	wg             sync.WaitGroup
)

func TestScenario1(t *testing.T) {
	input1 := make(chan string)
	defer close(input1)
	input2 := make(chan string)
	defer close(input2)
	done1 := make(chan bool)
	defer close(done1)
	done2 := make(chan bool)
	defer close(done2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			value := <-input1
			fmt.Println("node1:", value)

			done1 <- true
			if value == "exit" {
				break
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			value := <-input2
			fmt.Println("node2:", value)

			done2 <- true
			if value == "exit" {
				break
			}
		}
	}()

	// command 1: to 1
	go func() {
		input1 <- "hello 1"
	}()
	<-done1

	// command 2: to 2
	go func() {
		input2 <- "hello 2"
	}()
	<-done2

	// command 3: to 1
	go func() {
		input1 <- "hello 3"
	}()
	<-done1

	// stop1
	go func() {
		input1 <- "exit"
	}()
	<-done1

	// stop2
	go func() {
		input2 <- "exit"
	}()
	<-done2

	wg.Wait()
	t.Log("finish")
}

package main

import (
	"fmt"
	"sync"
	"testing"
)

var (
	input1, input2, input3 chan string
	done1, done2, done3    chan bool
)

func TestScenario1(t *testing.T) {
	input1 := make(chan string)
	defer close(input1)
	input2 := make(chan string)
	defer close(input2)
	input3 := make(chan string)
	defer close(input3)
	done1 := make(chan bool)
	defer close(done1)
	done2 := make(chan bool)
	defer close(done2)
	done3 := make(chan bool)
	defer close(done3)

	var wg sync.WaitGroup
	wg.Add(3)

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
	go func() {
		defer wg.Done()
		for {
			value := <-input3
			fmt.Println("node3:", value)

			done3 <- true
			if value == "exit" {
				break
			}
		}
	}()

	runCommand1 := func(command string) {
		input1 <- command
	}
	runCommand2 := func(command string) {
		input2 <- command
	}
	runCommand3 := func(command string) {
		input3 <- command
	}

	// command 1: to 1
	go runCommand1("hello 1")
	<-done1

	// command 2: to 2
	go runCommand2("hello 2")
	<-done2

	// command 3: to 1
	go runCommand1("hello 3")
	<-done1

	// command 4: to 3
	go runCommand3("hello 4")
	<-done3

	// stop1
	go runCommand1("exit")
	<-done1

	// stop2
	go runCommand2("exit")
	<-done2

	// stop3
	go runCommand3("exit")
	<-done3

	wg.Wait()
	t.Log("finish")
}

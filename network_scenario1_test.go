package main

import (
	"fmt"
	"sync"
	"testing"
)

var (
	input1, input2 chan string
	done1, done2   chan bool
	close1, close2 bool
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
	close1 = false
	close2 = false

	wg.Add(2)
	go func() {
		defer wg.Done()
		fmt.Println("node1")

		for {
			fmt.Println("1.0")
			value := <-input1
			fmt.Println("1.1")
			fmt.Println(value)

			fmt.Println("1.2")
			done1 <- true

			fmt.Println("1.3")

			if value == "exit" {
				break
			}
		}
	}()
	go func() {
		defer wg.Done()
		fmt.Println("node2")

		for {
			fmt.Println("2.0")
			value := <-input2
			fmt.Println("2.1")
			fmt.Println(value)

			fmt.Println("2.2")
			done2 <- true

			fmt.Println("2.3")

			if value == "exit" {
				break
			}
		}
	}()

	// command 1: to 1
	//wg.Add(1)
	go func() {
		//defer wg.Done()
		fmt.Println("command 1: a")

		input1 <- "hello 1"
		fmt.Println("command 1: b")
	}()

	<-done1
	fmt.Println("command 1: c")

	// command 2: to 2
	//wg.Add(1)
	go func() {
		//defer wg.Done()
		fmt.Println("command 2: a")

		input2 <- "hello 2"
		fmt.Println("command 2: b")
	}()

	<-done2
	fmt.Println("command 2: c")

	// command 3: to 1
	//wg.Add(1)
	go func() {
		//defer wg.Done()
		fmt.Println("command 3: a")

		input1 <- "hello 3"
		fmt.Println("command 3: b")
	}()

	<-done1
	fmt.Println("command 3: c")

	// stop1
	//wg.Add(1)
	go func() {
		//defer wg.Done()
		fmt.Println("stop1: a")

		input1 <- "exit"
		fmt.Println("stop1: b")
	}()

	<-done1
	fmt.Println("stop1: c")

	// stop2
	//wg.Add(1)
	go func() {
		//defer wg.Done()
		fmt.Println("stop2: a")

		input2 <- "exit"
		fmt.Println("stop2: b")
	}()

	<-done2
	fmt.Println("stop2: c")

	wg.Wait()

	t.Log("finish")
}

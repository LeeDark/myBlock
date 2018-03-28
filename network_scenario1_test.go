package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
)

var (
	input1, input2, input3 chan string
	done1, done2, done3    chan bool
)

func clearData(nodeID string) {
	err := os.Remove("blockchain_" + nodeID + ".db")
	if err != nil {
	}
	err = os.Remove("wallet_" + nodeID + ".dat")
	if err != nil {
	}
}

func createWallet(nodeID string) string {
	wallets, _ := NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)
	return address
}

func createBlockchain(t *testing.T, address, nodeID string) {
	if !ValidateAddress(address) {
		t.Fatal("ERROR: Address is not valid")
	}
	bc := CreateBlockchain(address, nodeID)
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	t.Log("Blockchain creating: Done!")
}

func printChain(t *testing.T, bc *Blockchain) {
	//bc := NewBlockchain(testNodeID)
	//defer bc.db.Close()

	bci := bc.Iterator()

	for {
		block := bci.Next()

		t.Logf("============ Block %x ============\n", block.Hash)
		t.Logf("Height: %d\n", block.Height)
		t.Logf("Prev. block: %x\n", block.PrevBlockHash)
		pow := NewProofOfWork(block)
		t.Logf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			t.Log(tx)
		}
		t.Logf("\n\n")

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

func TestScenario1(t *testing.T) {
	clearData("3000")

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
		var nodeID = "3000"
		var walletAddress string
		for {
			value := <-input1
			fmt.Println("node1:", value)

			done1 <- true
			if value == "exit" {
				break
			}
			switch value {
			case "step1a":
				// create a wallet and a new blockchain
				walletAddress = createWallet(nodeID)
				t.Logf("Wallet address: %s\n", walletAddress)
				createBlockchain(t, walletAddress, nodeID)

				bc := NewBlockchain(nodeID)
				defer bc.db.Close()
				t.Logf("Blockchain TIP: %x\n", bc.tip)
				printChain(t, bc)
				t.Log("Passed")
			default:
				fmt.Println("wrong command")
			}
		}
	}()
	go func() {
		defer wg.Done()
		//var nodeID = "3001"
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

	go runCommand1("step1a") // create a wallet and a new blockchain
	<-done1

	//go runCommand1("step1b") // copy blockchain as genesis blockchain
	//<-done1

	//go runCommand2("step2a") // create three wallets
	//<-done1

	//go runCommand1("step1c") // send some coins from CENTRAL to WALLETS with immediately mining
	//<-done1

	go runCommand1("step1d") // start NODE_ID=3000 - THE NODE MUST BE RUNNING UNTIL THE END OF THE SCENARIO
	<-done1

	go runCommand1("step1z") // stop NODE_ID=3000 - THE NODE MUST BE RUNNING UNTIL THE END OF THE SCENARIO
	<-done1

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

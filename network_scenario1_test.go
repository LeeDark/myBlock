package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	input1, input2, input3 chan string
	done1, done2, done3    chan bool
)

func nclearData() {
	err := os.Remove("blockchain_genesis.db")
	if err != nil {
	}
	clearNodeData("3000")
	clearNodeData("3001")
}

func clearNodeData(nodeID string) {
	err := os.Remove("blockchain_" + nodeID + ".db")
	if err != nil {
	}
	err = os.Remove("wallet_" + nodeID + ".dat")
	if err != nil {
	}
}

func ncreateWallet(nodeID string) string {
	wallets, _ := NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)
	return address
}

func ncreateBlockchain(t *testing.T, address, nodeID string) {
	if !ValidateAddress(address) {
		t.Fatal("ERROR: Address is not valid")
	}
	bc := CreateBlockchain(address, nodeID)
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	t.Log("Blockchain creating: Done!")
}

func copyFile(src string, dst string) {
	data, err := ioutil.ReadFile(src)
	if err != nil {
	}
	err = ioutil.WriteFile(dst, data, 0644)
	if err != nil {
	}
}

func nsendTransaction(t *testing.T, bc *Blockchain, from, to string, amount int, nodeID string, mineNow bool) {
	if !ValidateAddress(from) {
		t.Fatal("ERROR: Sender address is not valid")
	}
	if !ValidateAddress(to) {
		t.Fatal("ERROR: Recipient address is not valid")
	}

	UTXOSet := UTXOSet{bc}

	wallets, err := NewWallets(nodeID)
	if err != nil {
		t.Fatal(err)
	}
	wallet := wallets.GetWallet(from)

	tx := NewUTXOTransaction(&wallet, to, amount, &UTXOSet)

	t.Logf("knownNodes: %v\n", knownNodes)
	if mineNow {
		cbTx := NewCoinbaseTX(from, "")
		txs := []*Transaction{cbTx, tx}

		newBlock := bc.MineBlock(txs)
		UTXOSet.Update(newBlock)
	} else {
		sendTx(knownNodes[0], nodeID, tx)
	}

	t.Log("Success!")
}

func nprintChain(t *testing.T, bc *Blockchain) {
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
	nclearData()

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

	var centralAddress string
	var walletAddress1, walletAddress2, walletAddress3 string

	go func() {
		defer wg.Done()
		var nodeID = "3000"
		//var bc *Blockchain
		for {
			value := <-input1
			fmt.Println("node1:", value)

			done1 <- true
			if value == "exit" {
				//bc.db.Close()
				break
			}
			switch value {
			case "step1a":
				// create a wallet and a new blockchain
				centralAddress = ncreateWallet(nodeID)
				t.Logf("centralAddress: %s\n", centralAddress)
				ncreateBlockchain(t, centralAddress, nodeID)

				bc := NewBlockchain(nodeID)
				t.Logf("Blockchain TIP: %x\n", bc.tip)
				//nprintChain(t, bc)
				bc.db.Close()
			case "step1b":
				copyFile("blockchain_3000.db", "blockchain_genesis.db")
			case "step1c": //send some coins from CENTRAL to WALLETS with immediately mining
				bc := NewBlockchain(nodeID)
				nsendTransaction(t, bc, centralAddress, walletAddress1, 10, nodeID, true)
				nsendTransaction(t, bc, centralAddress, walletAddress2, 10, nodeID, true)
				//nprintChain(t, bc)
				bc.db.Close()
			case "step1d":
				// start NODE_ID=3000
			case "step1z":
				// stop NODE_ID=3000
			default:
				t.Log("wrong command")
			}
		}
	}()
	go func() {
		defer wg.Done()
		var nodeID = "3001"
		for {
			value := <-input2
			fmt.Println("node2:", value)

			done2 <- true
			if value == "exit" {
				break
			}
			switch value {
			case "step2a":
				// create three wallets
				walletAddress1 = ncreateWallet(nodeID)
				t.Logf("walletAddress1: %s\n", walletAddress1)
				walletAddress2 = ncreateWallet(nodeID)
				t.Logf("walletAddress2: %s\n", walletAddress2)
				walletAddress3 = ncreateWallet(nodeID)
				t.Logf("walletAddress3: %s\n", walletAddress3)
			case "step2b":
				copyFile("blockchain_genesis.db", "blockchain_3001.db")
			case "step2c":
				// start NODE_ID=3001
			case "step2d":
				// stop NODE_ID=3001
			default:
				t.Log("wrong command")
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

	go runCommand1("step1b") // copy blockchain as genesis blockchain
	<-done1

	go runCommand2("step2a") // create three wallets
	<-done2
	time.Sleep(2000 * time.Millisecond)

	go runCommand1("step1c") // send some coins from CENTRAL to WALLETS with immediately mining
	<-done1
	time.Sleep(5000 * time.Millisecond)

	//go runCommand1("step1d") // start NODE_ID=3000 - THE NODE MUST BE RUNNING UNTIL THE END OF THE SCENARIO
	//<-done1
	wg.Add(1)
	server0 := NewServer("3000", "")
	go func() {
		defer wg.Done()
		t.Log("Starting node 3000")
		server0.Start()
	}()

	go runCommand2("step2b") // copy genesis blockchain as blockchain for NODE_ID=3001
	<-done2

	//go runCommand2("step2c") // start NODE_ID=3001 - it will download all the blocks from CENTRAL
	//<-done2
	wg.Add(1)
	server1 := NewServer("3001", "")
	go func() {
		defer wg.Done()
		t.Log("Starting node 3001")
		server1.Start()
	}()

	//go runCommand2("step2d") // stop NODE_ID=3001
	//<-done2
	time.Sleep(5000 * time.Millisecond)
	server1.Stop()

	//go runCommand1("step1z") // stop NODE_ID=3000 - THE NODE MUST BE RUNNING UNTIL THE END OF THE SCENARIO
	//<-done1
	time.Sleep(1000 * time.Millisecond)
	server0.Stop()

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

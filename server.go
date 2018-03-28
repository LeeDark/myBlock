package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

//var nodeAddress string
//var miningAddress string
var knownNodes = []string{"localhost:3000"}

//var blocksInTransit = [][]byte{}
//var mempool = make(map[string]Transaction)

type TCPServer struct {
	nodeID        string
	nodeAddress   string
	miningAddress string
	//knownNodes []string
	blocksInTransit [][]byte
	mempool         map[string]Transaction

	ln   net.Listener
	conn net.Conn
	bc   *Blockchain
}

func NewServer(nodeID, minerAddress string) *TCPServer {
	return &TCPServer{
		nodeID:        nodeID,
		nodeAddress:   fmt.Sprintf("localhost:%s", nodeID),
		miningAddress: minerAddress,
		//knownNodes: []string{"localhost:3000"},
		blocksInTransit: [][]byte{},
		mempool:         make(map[string]Transaction),
	}
}

// StartServer starts a node
func (s *TCPServer) Start() {
	var err error
	s.ln, err = net.Listen(protocol, s.nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	//defer server.ln.Close()

	s.bc = NewBlockchain(s.nodeID)

	if s.nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], s.nodeAddress, s.bc)
	}

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println(err)
			break
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	//fmt.Println("handleConnection")
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := bytesToCommand(request[:commandLength])
	nanonow := time.Now().Format(time.RFC3339Nano)
	fmt.Printf("nodeID: %s, %s: Received %s command\n", s.nodeAddress, nanonow, command)

	switch command {
	case "addr":
		s.handleAddr(request)
	case "block":
		s.handleBlock(request)
	case "inv":
		s.handleInv(request)
	case "getblocks":
		s.handleGetBlocks(request)
	case "getdata":
		s.handleGetData(request)
	case "tx":
		s.handleTx(request)
	case "version":
		s.handleVersion(request)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

func (s *TCPServer) Stop() {
	if s.ln != nil {
		s.ln.Close()
	}
	if s.bc != nil && s.bc.db != nil {
		s.bc.db.Close()
	}
}

type addr struct {
	AddrList []string
}

type block struct {
	AddrFrom string
	Block    []byte
}

type getblocks struct {
	AddrFrom string
}

type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type tx struct {
	AddFrom     string
	Transaction []byte
}

type verzion struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func extractCommand(request []byte) []byte {
	return request[:commandLength]
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func sendData(address string, data []byte) {
	conn, err := net.Dial(protocol, address)
	if err != nil {
		fmt.Printf("%s is not available\n", address)
		var updatedNodes []string

		for _, node := range knownNodes {
			if node != address {
				updatedNodes = append(updatedNodes, node)
			}
		}

		knownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func requestBlocks(nodeAddress string) {
	for _, node := range knownNodes {
		sendGetBlocks(node, nodeAddress)
	}
}

func sendAddr(address, nodeAddress string) {
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := gobEncode(nodes)
	request := append(commandToBytes("addr"), payload...)

	sendData(address, request)
}

func sendBlock(address, nodeAddress string, b *Block) {
	data := block{nodeAddress, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)

	sendData(address, request)
}

func sendInv(address, nodeAddress, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)

	sendData(address, request)
}

func sendGetBlocks(address, nodeAddress string) {
	payload := gobEncode(getblocks{nodeAddress})
	request := append(commandToBytes("getblocks"), payload...)

	sendData(address, request)
}

func sendGetData(address, nodeAddress, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddress, kind, id})
	request := append(commandToBytes("getdata"), payload...)

	sendData(address, request)
}

func sendTx(address, nodeAddress string, tnx *Transaction) {
	data := tx{nodeAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendData(address, request)
}

func sendVersion(address, nodeAddress string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(verzion{nodeVersion, bestHeight, nodeAddress})

	request := append(commandToBytes("version"), payload...)

	sendData(address, request)
}

func (s *TCPServer) handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	requestBlocks(s.nodeAddress)
}

func (s *TCPServer) handleBlock(request []byte) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := DeserializeBlock(blockData)

	fmt.Printf("nodeID: %s, Recevied a new block!\n", s.nodeAddress)
	s.bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(s.blocksInTransit) > 0 {
		blockHash := s.blocksInTransit[0]
		sendGetData(payload.AddrFrom, s.nodeAddress, "block", blockHash)

		s.blocksInTransit = s.blocksInTransit[1:]
	} else {
		UTXOSet := UTXOSet{s.bc}
		UTXOSet.Reindex()
	}
}

func (s *TCPServer) handleInv(request []byte) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)
	fmt.Printf("len(mempool): %d\n", len(s.mempool))

	if payload.Type == "block" {
		s.blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, s.nodeAddress, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range s.blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		s.blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if s.mempool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, s.nodeAddress, "tx", txID)
		}
	}
}

func (s *TCPServer) handleGetBlocks(request []byte) {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := s.bc.GetBlockHashes()
	fmt.Printf("blocks: %s\n", s.bc)
	sendInv(payload.AddrFrom, s.nodeAddress, "block", blocks)
}

func (s *TCPServer) handleGetData(request []byte) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := s.bc.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		sendBlock(payload.AddrFrom, s.nodeAddress, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := s.mempool[txID]

		sendTx(payload.AddrFrom, s.nodeAddress, &tx)
		// delete(mempool, txID)
	}
}

func (s *TCPServer) handleTx(request []byte) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	s.mempool[hex.EncodeToString(tx.ID)] = tx

	if s.nodeAddress == knownNodes[0] {
		fmt.Printf("nodeAddress: %s, knownNodes: %v\n", s.nodeAddress, knownNodes)
		for _, node := range knownNodes {
			if node != s.nodeAddress && node != payload.AddFrom {
				sendInv(node, s.nodeAddress, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		fmt.Printf("miningAddress: %s, len(mempool): %d\n", s.miningAddress, len(s.mempool))
		if len(s.mempool) >= 1 && len(s.miningAddress) > 0 {
		MineTransactions:
			fmt.Println("MineTransactions...")
			var txs []*Transaction

			for id := range s.mempool {
				tx := s.mempool[id]
				if s.bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}

			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			cbTx := NewCoinbaseTX(s.miningAddress, "")
			txs = append(txs, cbTx)

			newBlock := s.bc.MineBlock(txs)
			UTXOSet := UTXOSet{s.bc}
			UTXOSet.Reindex()

			fmt.Println("New block is mined!")

			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(s.mempool, txID)
			}

			for _, node := range knownNodes {
				if node != s.nodeAddress {
					sendInv(node, s.nodeAddress, "block", [][]byte{newBlock.Hash})
				}
			}

			if len(s.mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

func (s *TCPServer) handleVersion(request []byte) {
	var buff bytes.Buffer
	var payload verzion

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	myBestHeight := s.bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom, s.nodeAddress)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, s.nodeAddress, s.bc)
	}

	// sendAddr(payload.AddrFrom)
	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

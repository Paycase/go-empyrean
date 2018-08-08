package core

import (
	"math/big"
	"time"
	"strconv"
	"database/sql"
	"log"
	_ "github.com/lib/pq"
	"github.com/ShyftNetwork/go-empyrean/common"
	"github.com/ShyftNetwork/go-empyrean/core/types"
	Rewards "github.com/ShyftNetwork/go-empyrean/consensus/ethash"
	"github.com/ShyftNetwork/go-empyrean/shyfttracerinterface"
)

var IShyftTracer shyfttracerinterface.IShyftTracer

func SetIShyftTracer(st shyfttracerinterface.IShyftTracer) {
	IShyftTracer = st
}

//SBlock type
type SBlock struct {
	Hash     	string
	Coinbase 	string
	AgeGet      string
	Age 		time.Time
	ParentHash 	string
	UncleHash 	string
	Difficulty 	string
	Size 		string
	Rewards 	string
	Number   	string
	GasUsed	 	uint64
	GasLimit 	uint64
	Nonce 		uint64
	TxCount  	int
	UncleCount 	int
	Blocks 		[]SBlock
}

//blockRes struct
type blockRes struct {
	hash     string
	coinbase string
	number   string
	Blocks   []SBlock
}

type SAccounts struct {
	Addr    		string
	Balance 		string
	AccountNonce	string
}

type accountRes struct {
	addr        string
	balance     string
	AllAccounts []SAccounts
}

type txRes struct {
	TxEntry []ShyftTxEntryPretty
}

type ShyftTxEntryPretty struct {
	TxHash    		string
	To				*common.Address
	ToGet			string
	From      		string
	BlockHash 		string
	BlockNumber 	string
	Amount   		string
	GasPrice  		uint64
	Gas       		uint64
	GasLimit  		uint64
	Cost	  		uint64
	Nonce     		uint64
	Status	  		string
	IsContract 		bool
	Age        		time.Time
	Data      		[]byte
}

type SendAndReceive struct {
	To        	 string
	From      	 string
	Amount    	 string
	Address   	 string
	Balance   	 string
	AccountNonce uint64 `json:",string"`
}

//WriteBlock writes to block info to sql db
func SWriteBlock(block *types.Block, receipts []*types.Receipt) error {
	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	rewards := swriteMinerRewards(sqldb,block)

	blockData := SBlock{
		Hash: 			block.Header().Hash().Hex(),
		Coinbase: 		block.Header().Coinbase.String(),
		Number: 		block.Header().Number.String(),
		GasUsed: 		block.Header().GasUsed,
		GasLimit: 		block.Header().GasLimit,
		TxCount: 		block.Transactions().Len(),
		UncleCount: 	len(block.Uncles()),
		ParentHash: 	block.ParentHash().String(),
		UncleHash: 		block.UncleHash().String(),
		Difficulty: 	block.Difficulty().String(),
		Size: 			block.Size().String(),
		Nonce: 			block.Nonce(),
		Rewards:        rewards,
	}

	i, err := strconv.ParseInt(block.Time().String(), 10, 64)
	if err != nil {
		panic(err)
	}
	age := time.Unix(i, 0)

	blockAge := SBlock {
		Age:   age,
	}

	//Inserts block data into DB
	InsertBlock(sqldb, blockData, blockAge)

	if block.Transactions().Len() > 0 {
		for _, tx := range block.Transactions() {
			swriteTransactions(sqldb, tx, block.Header().Hash(), blockData.Number, receipts, age, blockData.GasLimit)
			if block.Transactions()[0].To() != nil {
				swriteFromBalance(sqldb, tx)
			}
			if block.Transactions()[0].To() == nil {
				swriteContractBalance(sqldb, tx)
			}
		}
	}
	return nil
}

//swriteTransactions writes to sqldb, a SHYFT postgres instance
func swriteTransactions(sqldb *sql.DB, tx *types.Transaction, blockHash common.Hash, blockNumber string,  receipts []*types.Receipt, age time.Time, gasLimit uint64) error {
	var isContract bool
	var statusFromReciept string
	var contractAddressFromReciept common.Address

	txData := ShyftTxEntryPretty{
		TxHash:    	 tx.Hash().Hex(),
		From:      	 tx.From().Hex(),
		To:        	 tx.To(),
		BlockHash: 	 blockHash.Hex(),
		BlockNumber: blockNumber,
		Amount:    	 tx.Value().String(),
		Cost:	   	 tx.Cost().Uint64(),
		GasPrice:  	 tx.GasPrice().Uint64(),
		GasLimit:  	 gasLimit,
		Gas:       	 tx.Gas(),
		Nonce:     	 tx.Nonce(),
		Age: 	   	 age,
		Data:      	 tx.Data(),
	}

	if tx.To() == nil {
		for _, receipt := range receipts {
			statusReciept := (*types.ReceiptForStorage)(receipt).Status
			contractAddressFromReciept = (*types.ReceiptForStorage)(receipt).ContractAddress
			switch {
			case statusReciept == 0:
				statusFromReciept = "FAIL"
			case statusReciept == 1:
				statusFromReciept = "SUCCESS"
			}
		}
		isContract = true
		contractData := ShyftTxEntryPretty{
			Status: 	statusFromReciept,
			IsContract: isContract,
			To:         &contractAddressFromReciept,
		}
		//Insert Tx into DB
		InsertTx(sqldb, txData, contractData)
	} else {
		isContract = false
		for _, receipt := range receipts {
			statusReciept := (*types.ReceiptForStorage)(receipt).Status
			switch {
			case statusReciept == 0:
				statusFromReciept = "FAIL"
			case statusReciept == 1:
				statusFromReciept = "SUCCESS"
			}
		}
		data := ShyftTxEntryPretty{
			Status: 	statusFromReciept,
			IsContract: isContract,
			To:			tx.To(),
		}
		//Insert Tx into DB
		InsertTx(sqldb, txData, data)
	}
	//Runs necessary functions for tracing internal transactions through tracers.go
	IShyftTracer.GetTracerToRun(tx.Hash())

	return nil
}

func swriteContractBalance(sqldb *sql.DB, tx *types.Transaction) error {
	sendAndReceiveData := SendAndReceive{
		From:   		tx.From().Hex(),
		Amount: 		tx.Value().String(),
		AccountNonce: 	tx.Nonce(),
	}

	fromAddressBalance, fromAccountNonce, err := AccountExists(sqldb, sendAndReceiveData.From)

	switch {
	case err == sql.ErrNoRows:
		accountNonce := strconv.FormatUint(tx.Nonce(), 10)
		CreateAccount(sqldb, sendAndReceiveData.From, sendAndReceiveData.Amount, accountNonce)
	default:
		var newBalanceSender,newAccountNonceSender  big.Int
		var nonceIncrement = big.NewInt(1)

		fromBalance := new(big.Int)
		fromBalance, _ = fromBalance.SetString(fromAddressBalance, 10)

		fromNonce := new(big.Int)
		fromNonce, _ = fromNonce.SetString(fromAccountNonce, 10)

		newBalanceSender.Sub(fromBalance, tx.Value())
		newAccountNonceSender.Add(fromNonce, nonceIncrement)

		UpdateAccount(sqldb, sendAndReceiveData.From, newBalanceSender.String(), newAccountNonceSender.String())
	}
	return nil
}

//writeFromBalance writes senders balance to accounts db
func swriteFromBalance(sqldb *sql.DB, tx *types.Transaction) error {
	sendAndReceiveData := SendAndReceive{
		To: tx.To().Hex(),
		From: tx.From().Hex(),
		Amount: tx.Value().String(),
	}

	toAddressBalance, toAccountNonce, err := AccountExists(sqldb, sendAndReceiveData.To)

	switch {
	case err == sql.ErrNoRows:
		accountNonce := strconv.FormatUint(tx.Nonce(), 10)
		CreateAccount(sqldb, sendAndReceiveData.To, sendAndReceiveData.Amount, accountNonce)
	case err != nil:
		log.Fatal(err)
	default:
		fromAddressBalance, fromAccountNonce, err := AccountExists(sqldb, sendAndReceiveData.From)
		if err != nil {
			log.Fatal(err)
		}
		var newBalanceReceiver, newBalanceSender, newAccountNonceReceiver, newAccountNonceSender  big.Int
		var nonceIncrement = big.NewInt(1)

		//STRING TO BIG INT
		//BALANCES TO AND FROM ADDR
		toBalance := new(big.Int)
		toBalance, _ = toBalance.SetString(toAddressBalance, 10)
		fromBalance := new(big.Int)
		fromBalance, _ = fromBalance.SetString(fromAddressBalance, 10)

		//ACCOUNT NONCES
		toNonce := new(big.Int)
		toNonce, _ = toNonce.SetString(toAccountNonce, 10)
		fromNonce := new(big.Int)
		fromNonce, _ = fromNonce.SetString(fromAccountNonce, 10)

		newBalanceReceiver.Add(toBalance, tx.Value())
		newBalanceSender.Sub(fromBalance, tx.Value())

		newAccountNonceReceiver.Add(toNonce, nonceIncrement)
		newAccountNonceSender.Add(fromNonce, nonceIncrement)

		//UPDATE ACCOUNTS BASED ON NEW BALANCES AND ACCOUNT NONCES
		UpdateAccount(sqldb, sendAndReceiveData.To,   newBalanceReceiver.String(), newAccountNonceReceiver.String())
		UpdateAccount(sqldb, sendAndReceiveData.From, newBalanceSender.String(),   newAccountNonceSender.String())
	}
	return nil
}

// @NOTE: This function is extremely complex and requires heavy testing and knowdlege of edge cases:
// uncle blocks, account balance updates based on reorgs, diverges that get dropped.
// Reason for this is because the accounts are not deterministic like the block and tx hashes.
// @TODO: Calculate reorg
func swriteMinerRewards(sqldb *sql.DB, block *types.Block) string {
	minerAddr := block.Coinbase().String()
	shyftConduitAddress := Rewards.ShyftNetworkConduitAddress.String()
	// Calculate the total gas used in the block
	totalGas := new(big.Int)
	for _, tx := range block.Transactions() {
		totalGas.Add(totalGas, new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas())))
	}

	totalMinerReward := totalGas.Add(totalGas, Rewards.ShyftMinerBlockReward)

	// References:
	// https://ethereum.stackexchange.com/questions/27172/different-uncles-reward
	// line 551 in consensus.go (shyft_go-ethereum/consensus/ethash/consensus.go)
	// Some weird constants to avoid constant memory allocs for them.
	var big8   = big.NewInt(8)
	var uncleRewards []*big.Int
	var uncleAddrs []string

	// uncleReward is overwritten after each iteration
	uncleReward := new(big.Int)
	for _, uncle := range block.Uncles() {
		uncleReward.Add(uncle.Number, big8)
		uncleReward.Sub(uncleReward, block.Number())
		uncleReward.Mul(uncleReward, Rewards.ShyftMinerBlockReward)
		uncleReward.Div(uncleReward, big8)
		uncleRewards = append(uncleRewards, uncleReward)
		uncleAddrs = append(uncleAddrs, uncle.Coinbase.String())
	}

	sstoreReward(sqldb, minerAddr, totalMinerReward)
	sstoreReward(sqldb, shyftConduitAddress, Rewards.ShyftNetworkBlockReward)
	var uncRewards = new(big.Int)
	for i := 0; i < len(uncleAddrs); i++ {
		_ = uncleRewards[i]
		sstoreReward(sqldb, uncleAddrs[i], uncleRewards[i])
	}

	fullRewardValue := new(big.Int)
	fullRewardValue.Add(totalMinerReward, Rewards.ShyftNetworkBlockReward)
	fullRewardValue.Add(fullRewardValue, uncRewards)

	return fullRewardValue.String()
}

func sstoreReward(sqldb *sql.DB, address string, reward *big.Int) {
	// Check if address exists
	addressBalance, accountNonce, err := AccountExists(sqldb, address)

	if err == sql.ErrNoRows {
		// Addr does not exist, thus create new entry
		// We convert totalReward into a string and postgres converts into number
		CreateAccount(sqldb, address, reward.String(), "1")
		return
	} else if err != nil {
		// Something went wrong panic
		panic(err)
	} else {
		// Addr exists, update existing balance
		bigBalance := new(big.Int)
		var nonceIncrement = big.NewInt(1)
		currentAccountNonce := new(big.Int)
		currentAccountNonce, errorr := currentAccountNonce.SetString(accountNonce, 10)
		if !errorr {
			panic(errorr)
		}
		bigBalance, err := bigBalance.SetString(addressBalance, 10)
		if !err {
			panic(err)
		}
		newBalance := new(big.Int)
		newAccountNonce := new(big.Int)
		newBalance.Add(newBalance, bigBalance)
		newBalance.Add(newBalance, reward)
		newAccountNonce.Add(currentAccountNonce, nonceIncrement)
		//Update the balance and nonce
		UpdateAccount(sqldb, address, newBalance.String(), newAccountNonce.String())
		return
	}
}

///////////////////////
//DB Utility functions
//////////////////////
func CreateAccount (sqldb *sql.DB, addr string, balance string, accountNonce string) {
	sqlStatement := `INSERT INTO accounts(addr, balance, accountNonce) VALUES(($1), ($2), ($3)) RETURNING addr`
	insertErr := sqldb.QueryRow(sqlStatement, addr, balance, accountNonce).Scan(&addr)
	if insertErr != nil {
		panic(insertErr)
	}
}

func AccountExists (sqldb *sql.DB, addr string) (string, string, error) {
	var addressBalance, accountNonce string
	sqlExistsStatement := `SELECT balance, accountNonce from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(sqlExistsStatement, addr).Scan(&addressBalance, &accountNonce)

	switch {
	case err == sql.ErrNoRows:
		return addressBalance, accountNonce, err
	case err != nil:
		panic(err)
	default:
		return addressBalance, accountNonce, err
	}
}

func UpdateAccount(sqldb *sql.DB, addr string, balance string, accountNonce string) {
	updateSQLStatement := `UPDATE accounts SET balance = ($2), accountNonce = ($3) WHERE addr = ($1)`
	_, updateErr := sqldb.Exec(updateSQLStatement, addr, balance, accountNonce)
	if updateErr != nil {
		panic(updateErr)
	}
}

func InsertBlock(sqldb *sql.DB, blockData SBlock, blockAge SBlock) {
	sqlStatement := `INSERT INTO blocks(hash, coinbase, number, gasUsed, gasLimit, txCount, uncleCount, age, parentHash, uncleHash, difficulty, size, rewards, nonce) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12),($13), ($14)) RETURNING number`
	qerr := sqldb.QueryRow(sqlStatement, blockData.Hash, blockData.Coinbase, blockData.Number, blockData.GasUsed, blockData.GasLimit, blockData.TxCount, blockData.UncleCount, blockAge.Age, blockData.ParentHash, blockData.UncleHash, blockData.Difficulty, blockData.Size, blockData.Rewards, blockData.Nonce).Scan(&blockData.Number)
	if qerr != nil {
		panic(qerr)
	}
}


func InsertTx (sqldb *sql.DB, txData ShyftTxEntryPretty, data ShyftTxEntryPretty) {
	var retNonce string
	sqlStatement := `INSERT INTO txs(txhash, from_addr, to_addr, blockhash, blockNumber, amount, gasprice, gas, gasLimit, txfee, nonce, isContract, txStatus, age, data) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12), ($13), ($14), ($15)) RETURNING nonce`
	err := sqldb.QueryRow(sqlStatement, txData.TxHash, txData.From, data.To.String(), txData.BlockHash, txData.BlockNumber, txData.Amount, txData.GasPrice, txData.Gas, txData.GasLimit, txData.Cost, txData.Nonce, data.IsContract, data.Status, txData.Age, txData.Data).Scan(&retNonce)
	if err != nil {
		panic(err)
	}
}


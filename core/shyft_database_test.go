package core

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ShyftNetwork/go-empyrean/core/sTypes"
	"github.com/ShyftNetwork/go-empyrean/ethdb"
	"github.com/jmoiron/sqlx"
)

var GenesisAcctAddresses = []string{"0x0000000000000000000000000000000000000000", "0x0000000000000000000000000000000000000001",
	"0x0000000000000000000000000000000000000002", "0x0000000000000000000000000000000000000003",
	"0x0000000000000000000000000000000000000004", "0x0000000000000000000000000000000000000005",
	"0x0000000000000000000000000000000000000006", "0x0000000000000000000000000000000000000007",
	"0x0000000000000000000000000000000000000008"}

func TestCreateAccount(t *testing.T) {
	t.Run("CreateAccount - creates an account in the PG db ", func(t *testing.T) {
		var accountSlice []ethdb.Account
		var newDbAccounts ethdb.Account

		db, err := ethdb.NewShyftDatabase()
		db.TruncateTables()
		addr := "0x7ef5a6135f1fd6a02593eedc869c6d41d934aef8"
		balance, _ := new(big.Int).SetString("3500000000", 10)
		accountNonce := strconv.Itoa(int(1))
		err = db.CreateAccount(addr, balance.String(), accountNonce)
		if err != nil {
			fmt.Println(err)
			return
		}

		accountJSON, err := ethdb.SGetAccount(addr)
		accountBYTE := []byte(accountJSON)
		err = json.Unmarshal(accountBYTE, &newDbAccounts)
		if err != nil {
			panic(err)
		}
		accounts := append(accountSlice, newDbAccounts)

		if len(accounts) > 1 {
			t.Errorf("Got %v Accounts Created: Expected 1", len(accounts))
		}
		stringBalance := accounts[0].Balance
		if accounts[0].Addr != addr || stringBalance != "3500000000" || accountNonce != "1" {
			t.Errorf("Account: Got %v Accounts Created: Expected addr: %s balance: %d nonce %s", newDbAccounts, addr, balance, accountNonce)
		}
	})
}

func TestInsertTx(t *testing.T) {
	// Set up a  test transaction
	tx := CreateTestTransactions()
	db, _ := ethdb.NewShyftDatabase()
	db.TruncateTables()
	testDbName := ethdb.DbName()

	blockHash := "0x656c34545f90a730a19008c0e7a7cd4fb3895064b48d6d69761bd5abad681056"
	txData := stypes.ShyftTxEntryPretty{
		TxHash:      tx[0].Hash().Hex(),
		From:        tx[0].From().Hex(),
		To:          tx[0].To().String(),
		BlockHash:   blockHash,
		BlockNumber: strconv.Itoa(21234),
		Amount:      tx[0].Value().String(),
		Cost:        tx[0].Cost().String(),
		GasPrice:    tx[0].GasPrice().Uint64(),
		GasLimit:    uint64(18000),
		Gas:         tx[0].Gas(),
		Nonce:       tx[0].Nonce(),
		Age:         time.Now(),
		Data:        tx[0].Data(),
		Status:      "SUCCESS",
		IsContract:  false,
	}
	t.Run("InsertTx - No Account exists inserts a transaction to the database and updates/creates accounts accordingly", func(t *testing.T) {
		db.TruncateTables()
		var dbSliceTransaction []ethdb.PgTransaction
		var accountSlice []ethdb.Account
		var accountBlockSlice []ethdb.AccountBlock

		db.InsertTx(txData)
		connectionStr := fmt.Sprintf("user=postgres password=docker database=%s sslmode=disable", testDbName)
		testdb, err := sqlx.Connect("postgres", connectionStr)
		if err != nil {
			log.Fatalln(err)
		}
		testdb.Select(&dbSliceTransaction, "SELECT * FROM txs")
		pgdb := dbSliceTransaction[0]
		txInput := txData
		if len(dbSliceTransaction) != 1 {
			t.Errorf("Got %v db transactions created \nExpected 1", len(dbSliceTransaction))
		}
		if pgdb.TxHash != txInput.TxHash && pgdb.Blockhash != txData.BlockHash && pgdb.To != txData.To && pgdb.From != txData.From && pgdb.Amount != txData.Amount {
			t.Errorf("Got %+v \nExpected %+v", dbSliceTransaction[0], txData)
		}
		testdb.Select(&accountSlice, "SELECT * FROM accounts;")
		if len(accountSlice) != 2 {
			t.Errorf("Got %v db accounts created -  Expected 2", len(accountSlice))
		}

		toAcct := accountSlice[0]
		fromAcct := accountSlice[1]
		if toAcct.Addr != txData.To && toAcct.Balance != tx[0].Value().String() && toAcct.Nonce != 1 {
			t.Errorf("Got %+v \nExpected %s %s %d", toAcct, txData.To, txData.Amount, 1)
		}
		fromAcctBal, _ := strconv.Atoi(txData.Amount)
		fromBalInt := -1 * fromAcctBal
		product := new(big.Int)
		product.Mul(new(big.Int).SetInt64(-1), tx[0].Value())
		if fromAcct.Addr != txData.To && fromAcct.Balance != product.String() &&
			fromAcct.Nonce != 1 {
			t.Errorf("Got %+v \nExpected %s %d %d", fromAcct, txData.From, fromBalInt, 1)
		}
		testdb.Select(&accountBlockSlice, "SELECT * FROM accountblocks;")

		if len(accountBlockSlice) != 2 {
			t.Errorf("Got %d db accountblocks created -  Expected 2", len(accountBlockSlice))
		}
		toAcctBl := accountBlockSlice[0]
		fromAcctBl := accountBlockSlice[1]
		if toAcctBl.Acct != txData.To && toAcctBl.Blockhash != txData.BlockHash &&
			strconv.Itoa(int(toAcctBl.Delta)) != txData.Amount && int(toAcctBl.TxCount) != 1 {
			t.Errorf("Got %+v \nExpected %s %s %s", toAcctBl, txData.To, txData.BlockHash, txData.Amount)
		}
		if fromAcctBl.Acct != txData.To && fromAcctBl.Blockhash != txData.BlockHash &&
			strconv.Itoa(int(fromAcctBl.Delta)*-1) != txData.Amount && int(toAcctBl.TxCount) != 1 {
			t.Errorf("Got %+v \nExpected %s %s %s", toAcctBl, txData.To, txData.BlockHash, txData.Amount)
		}
	})
}

func insertBlocksTransactions() (map[string][]ethdb.Account, []string, *ethdb.SPGDatabase, *sqlx.DB) {
	//ethdb.RemoveTestDbs("shyfttest")
	db, _ := ethdb.NewShyftDatabase()
	db.TruncateTables()

	blockHashes := []string{}
	blockAccounts := map[string][]ethdb.Account{}
	testDbName := ethdb.DbName()
	connectionStr := fmt.Sprintf("user=postgres password=docker database=%s sslmode=disable", testDbName)
	testdb, err := sqlx.Connect("postgres", connectionStr)
	if err != nil {
		panic(err)
	}
	for _, bl := range CreateTestBlocks() {
		var accountSlice []ethdb.Account
		// Write and verify the block in the database
		err := SWriteBlock(db, bl, CreateTestReceipts())
		if err != nil {
			panic(err)
		}
		testdb.Select(&accountSlice, "SELECT * FROM accounts;")
		blockHashes = append(blockHashes, bl.Hash().String())
		blockAccounts[bl.Hash().String()] = accountSlice
	}
	return blockAccounts, blockHashes, db, testdb
}

func TestRollbackReconcilesAccounts(t *testing.T) {

	t.Run("PgRollback - of all blocks reverses all account balances", func(t *testing.T) {
		_, blockHashes, db, testdb := insertBlocksTransactions()
		// Rollback 1 blocks
		db.RollbackPgDb(blockHashes[0:])

		var accountSlice []ethdb.Account
		testdb.Select(&accountSlice, "SELECT * FROM accounts;")

		if len(accountSlice) > 1 {
			t.Errorf("Rollback of the following blocks %+v expected %d accounts have %d\n", blockHashes[0:], 0, len(accountSlice))
		}

		blockAccounts := []ethdb.Account{}
		testdb.Select(&blockAccounts, "SELECT * FROM accountblocks;")
		if len(blockAccounts) != 0 {
			t.Errorf("Got %d db accountblocks on rollback -  Expected 2", len(blockAccounts))
		}

		var blockSlice []ethdb.Block
		testdb.Select(&blockSlice, "SELECT * FROM blocks;")
		if len(blockSlice) != 0 {
			t.Errorf("Got %d db blocks on rollback -  Expected 0", len(blockSlice))
		}

		var dbSliceTransaction []ethdb.PgTransaction
		testdb.Select(&dbSliceTransaction, "SELECT * FROM txs;")
		if len(dbSliceTransaction) != 0 {
			t.Errorf("Got %d db transactions on rollback -  Expected 0", len(dbSliceTransaction))
		}
	})
	t.Run("PgRollback - 2 Blocks- reverses all account balances accordingly", func(t *testing.T) {
		var dbSliceTransaction []ethdb.PgTransaction
		var accountSlice []ethdb.Account
		var accountBlockSlice []ethdb.AccountBlock
		var blockSlice []ethdb.Block
		//var newDbAccounts ethdb.Account

		blockDbAccts, blockHashes, db, testdb := insertBlocksTransactions()
		db.RollbackPgDb(blockHashes[1:])

		testdb.Select(&accountSlice, "SELECT * FROM accounts;")

		if len(accountSlice) != 6 {
			t.Errorf("Rollback of the following blocks %+v expected %d accounts have %d\n", blockHashes[1:], 5, len(accountSlice))
		}
		testdb.Select(&accountBlockSlice, "SELECT * FROM accountblocks;")
		if len(accountBlockSlice) != 6 {
			t.Errorf("Got %d db accountblocks on rollback -  Expected 5", len(accountBlockSlice))
		}

		testdb.Select(&blockSlice, "SELECT * FROM blocks;")
		if len(blockSlice) != 1 {
			t.Errorf("Got %d db blocks on rollback -  Expected 1", len(blockSlice))
		}

		testdb.Select(&dbSliceTransaction, "SELECT * FROM txs;")

		if len(dbSliceTransaction) != 3 {
			t.Errorf("Got %d db transactions on rollback -  Expected 2", len(dbSliceTransaction))
		}
		for _, acct := range blockDbAccts[blockHashes[0]] {
			log.Printf("Rollback 2 blocks acct -> %+v \n", acct)
			fetchDbBalanceStmnt := `SELECT * FROM accounts WHERE addr = $1`
			acctCheck := ethdb.Account{}
			err := testdb.Get(&acctCheck, fetchDbBalanceStmnt, acct.Addr)
			// log.Printf("ERR ACCOUNT %+v", acct.Addr)
			if err != nil {
				log.Printf("THIS IS THE ERROR")
				panic(err)
			}
			if acctCheck.Balance != acct.Balance || acctCheck.Nonce != acct.Nonce {
				t.Errorf("Got Balance: %s Nonce: %d Expected Balance: %s Nonce: %d - Addr: %s\n", acctCheck.Balance, acctCheck.Nonce, acct.Balance, acct.Nonce, acct.Addr)
			}
		}
	})
	t.Run("PgRollback - 1 Blocks- reverses all account balances accordingly", func(t *testing.T) {
		var dbSliceTransaction []ethdb.PgTransaction
		var accountSlice []ethdb.Account
		var accountBlockSlice []ethdb.AccountBlock
		var blockSlice []ethdb.Block
		//var newDbAccounts ethdb.Account

		blockDbAccts, blockHashes, db, testdb := insertBlocksTransactions()
		// Rollback 2 blocks
		db.RollbackPgDb(blockHashes[2:])
		testdb.Select(&accountSlice, "SELECT * FROM accounts;")
		if len(accountSlice) != 6 {
			t.Errorf("Rollback of the following blocks %+v expected %d accounts have %d\n", blockHashes[1:], 5, len(accountSlice))
		}

		testdb.Select(&accountBlockSlice, "SELECT * FROM accountblocks;")
		if len(accountBlockSlice) != 8 {
			t.Errorf("Got %d db accountblocks on rollback -  Expected 8", len(accountBlockSlice))
		}

		testdb.Select(&blockSlice, "SELECT * FROM blocks;")
		if len(blockSlice) != 2 {
			t.Errorf("Got %d db blocks on rollback -  Expected 2", len(blockSlice))
		}

		testdb.Select(&dbSliceTransaction, "SELECT * FROM txs;")

		if len(dbSliceTransaction) != 3 {
			t.Errorf("Got %d db transactions on rollback -  Expected 3", len(dbSliceTransaction))
		}
		for _, acct := range blockDbAccts[blockHashes[1]] {
			log.Printf("Rollback 1 blockacct -> %+v \n", acct)
			fetchDbBalanceStmnt := `SELECT * FROM accounts WHERE addr = $1`
			acctCheck := ethdb.Account{}
			err := testdb.Get(&acctCheck, fetchDbBalanceStmnt, acct.Addr)
			if err != nil {
				log.Printf("THIS IS THE ERROR")
				panic(err)
			}
			// log.Printf("ERR ACCOUNT %+v", acct.Addr)
			if acctCheck.Balance != acct.Balance || acctCheck.Nonce != acct.Nonce {
				t.Errorf("Got Balance: %s Nonce: %d Expected Balance: %s Nonce: %d - Addr: %s\n", acctCheck.Balance, acctCheck.Nonce, acct.Balance, acct.Nonce, acct.Addr)
			}
		}
	})
}

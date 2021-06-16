package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Define key names for options
const totalSupplyKey = "totalSupply"

// Define objectType names for prefix
const allowancePrefix = "allowance"

const holdPrefix = "hold"

const MintBurnKey = "MintBurn"
const BurnKey = "Burn"

const stateOrder = "Ordered"
const stateApproved = "Approved"
const stateRejected = "Rejected"

// SmartContract provides functions for transferring tokens between accounts
type SmartContract struct {
	contractapi.Contract
}

// event provides an organized struct for emitting events
type event struct {
	from  string
	to    string
	value int
}

type Account struct {
	ClientID string `json:"clientID"`
	Active   int    `json:"active"`
	OnHold   int    `json:"hold"`
}

type MintBurn struct {
	State map[string]St_am `json:"state"`
}

type St_am struct {
	MintBurn string `json:"mintburn"`
	Amount   int    `json:"amount"`
	State    string `json:"state"`
}

func (S *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface) error {
	// Get ID of client identity
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	balanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil || balanceBytes == nil {
		initBalance := 0

		err = ctx.GetStub().PutState(clientID, []byte(strconv.Itoa(initBalance)))
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("account %s already exist", clientID)
	}
	return nil
}

// Mint creates new tokens and adds them to minter's account balance
// This function triggers a Transfer event
func Mint(ctx contractapi.TransactionContextInterface, amount int) error {

	// Get ID of submitting client identity
	minter, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	if amount <= 0 {
		return fmt.Errorf("mint amount must be a positive integer")
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(minter)
	if err != nil {
		return fmt.Errorf("failed to read minter account %s from world state: %v", minter, err)
	}

	var currentBalance int

	// If minter current balance doesn't yet exist, we'll create it with a current balance of 0
	if currentBalanceBytes == nil {
		currentBalance = 0
	} else {
		currentBalance, _ = strconv.Atoi(string(currentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.
	}

	updatedBalance := currentBalance + amount

	err = ctx.GetStub().PutState(minter, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

	// Update the totalSupply
	totalSupplyBytes, err := ctx.GetStub().GetState(totalSupplyKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve total token supply: %v", err)
	}

	var totalSupply int

	// If no tokens have been minted, initialize the totalSupply
	if totalSupplyBytes == nil {
		totalSupply = 0
	} else {
		totalSupply, _ = strconv.Atoi(string(totalSupplyBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.
	}

	// Add the mint amount to the total supply and update the state
	totalSupply += amount
	err = ctx.GetStub().PutState(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

	// Emit the Transfer event
	transferEvent := event{"0x0", minter, amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.GetStub().SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("minter account %s balance updated from %d to %d", minter, currentBalance, updatedBalance)

	return nil
}

// Burn redeems tokens the minter's account balance
// This function triggers a Transfer event
func Burn(ctx contractapi.TransactionContextInterface, amount int) error {

	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to burn new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("client is not authorized to mint new tokens")
	}

	// Get ID of submitting client identity
	burner, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	if amount <= 0 {
		return errors.New("burn amount must be a positive integer")
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(burner)
	if err != nil {
		return fmt.Errorf("failed to read burner account %s from world state: %v", burner, err)
	}

	var currentBalance int

	// Check if burner current balance exists
	if currentBalanceBytes == nil {
		return errors.New("the balance does not exist")
	}

	currentBalance, _ = strconv.Atoi(string(currentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	updatedBalance := currentBalance - amount

	err = ctx.GetStub().PutState(burner, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

	// Update the totalSupply
	totalSupplyBytes, err := ctx.GetStub().GetState(totalSupplyKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve total token supply: %v", err)
	}

	// If no tokens have been burned, throw error
	if totalSupplyBytes == nil {
		return errors.New("totalSupply does not exist")
	}

	totalSupply, _ := strconv.Atoi(string(totalSupplyBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.

	// Subtract the burn amount to the total supply and update the state
	totalSupply -= amount
	err = ctx.GetStub().PutState(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

	// Emit the Transfer event
	transferEvent := event{burner, "0x0", amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.GetStub().SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("burner account %s balance updated from %d to %d", burner, currentBalance, updatedBalance)

	return nil
}

func (s *SmartContract) GetAccount(ctx contractapi.TransactionContextInterface) (*Account, error) {
	var currentBalance int
	var hold_amount int
	account := Account{
		ClientID: "",
		Active:   0,
		OnHold:   0,
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return &account, fmt.Errorf("failed to get client id: %v", err)
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil {
		return &account, fmt.Errorf("failed to read client's account %s from world state: %v", clientID, err)
	}

	// Check if minter current balance exists
	if currentBalanceBytes == nil {
		return &account, errors.New("the balance does not exist")
	}

	currentBalance, _ = strconv.Atoi(string(currentBalanceBytes))

	holdkey, err := ctx.GetStub().CreateCompositeKey(holdPrefix, []string{clientID})
	if err != nil {
		return &account, fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	// Read the allowance amount from the world state
	holdBytes, _ := ctx.GetStub().GetState(holdkey)

	if holdBytes == nil {
		hold_amount = 0
	} else {
		hold_amount, _ = strconv.Atoi(string(holdBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.
	}

	account = Account{
		ClientID: clientID,
		Active:   currentBalance,
		OnHold:   hold_amount,
	}

	return &account, nil
}

func (s *SmartContract) CreateHold(ctx contractapi.TransactionContextInterface, amount int) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	if amount <= 0 {
		return errors.New("hold amount must be a positive integer")
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil {
		return fmt.Errorf("failed to read client's account %s from world state: %v", clientID, err)
	}

	var currentBalance int

	// Check if minter current balance exists
	if currentBalanceBytes == nil {
		return errors.New("the balance does not exist")
	}

	currentBalance, _ = strconv.Atoi(string(currentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	updatedBalance := currentBalance - amount

	err = ctx.GetStub().PutState(clientID, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", clientID, err)
	}

	holdkey, err := ctx.GetStub().CreateCompositeKey(holdPrefix, []string{clientID})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", holdPrefix, err)
	}

	// Read the allowance amount from the world state
	holdBytes, _ := ctx.GetStub().GetState(holdkey)

	var hold_amount int

	// If no current allowance, set allowance to 0
	if holdBytes == nil {
		hold_amount = amount
	} else {
		hold_amount, _ = strconv.Atoi(string(holdBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.
		hold_amount = hold_amount + amount
	}

	// Update the state of the smart contract by adding the allowanceKey and value
	err = ctx.GetStub().PutState(holdkey, []byte(strconv.Itoa(hold_amount)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", holdkey, err)
	}
	return nil
}

func ExecuteHold(ctx contractapi.TransactionContextInterface, holder string, amount int) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	if amount <= 0 {
		return errors.New("hold amount must be a positive integer")
	}

	holdkey, err := ctx.GetStub().CreateCompositeKey(holdPrefix, []string{holder})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", holdPrefix, err)
	}

	// Read the allowance amount from the world state
	holdBytes, _ := ctx.GetStub().GetState(holdkey)

	var hold_amount int

	// If no current hold amount then error
	if holdBytes == nil {
		return fmt.Errorf("failed to get hold amount ")
	}
	hold_amount, _ = strconv.Atoi(string(holdBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.
	if hold_amount < amount {
		return fmt.Errorf("error with hold amount")
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil {
		return fmt.Errorf("failed to read client's account %s from world state: %v", clientID, err)
	}

	var currentBalance int

	// Check if minter current balance exists
	if currentBalanceBytes == nil {
		return errors.New("the balance does not exist")
	}

	currentBalance, _ = strconv.Atoi(string(currentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	updatedBalance := currentBalance + amount

	err = ctx.GetStub().PutState(clientID, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", clientID, err)
	}

	currentBalanceBytes_h, err := ctx.GetStub().GetState(holder)
	if err != nil {
		return fmt.Errorf("failed to read client's account %s from world state: %v", clientID, err)
	}

	var currentBalance_h int

	// Check if minter current balance exists
	if currentBalanceBytes_h == nil {
		return errors.New("the balance does not exist")
	}

	currentBalance_h, _ = strconv.Atoi(string(currentBalanceBytes_h)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	updatedBalance_h := currentBalance_h + hold_amount - amount

	err = ctx.GetStub().PutState(holder, []byte(strconv.Itoa(updatedBalance_h)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", clientID, err)
	}

	err = ctx.GetStub().PutState(holdkey, []byte(strconv.Itoa(hold_amount)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", holdkey, err)
	}

	return nil
}

func (s *SmartContract) ReturnHold(ctx contractapi.TransactionContextInterface, holder string) error {
	holdkey, err := ctx.GetStub().CreateCompositeKey(holdPrefix, []string{holder})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", holdPrefix, err)
	}

	// Read the allowance amount from the world state
	holdBytes, _ := ctx.GetStub().GetState(holdkey)

	var hold_amount int

	// If no current hold amount then error
	if holdBytes == nil {
		return fmt.Errorf("failed to get hold amount ")
	}
	hold_amount, _ = strconv.Atoi(string(holdBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.

	currentBalanceBytes, err := ctx.GetStub().GetState(holder)
	if err != nil {
		return fmt.Errorf("failed to read client's account %s from world state: %v", holder, err)
	}

	var currentBalance int

	// Check if minter current balance exists
	if currentBalanceBytes == nil {
		return errors.New("the balance does not exist")
	}

	currentBalance, _ = strconv.Atoi(string(currentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	updatedBalance := currentBalance + hold_amount

	err = ctx.GetStub().PutState(holder, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", holder, err)
	}

	hold_amount = 0
	err = ctx.GetStub().PutState(holdkey, []byte(strconv.Itoa(hold_amount)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", holdkey, err)
	}

	return nil
}

// Transfer transfers tokens from client account to recipient account
// recipient account must be a valid clientID as returned by the ClientID() function
// This function triggers a Transfer event
func (s *SmartContract) Transfer(ctx contractapi.TransactionContextInterface, recipient string, amount int) error {

	// Get ID of submitting client identity
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	err = transferHelper(ctx, clientID, recipient, amount)
	if err != nil {
		return fmt.Errorf("failed to transfer: %v", err)
	}

	// Emit the Transfer event
	transferEvent := event{clientID, recipient, amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.GetStub().SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

// BalanceOf returns the balance of the given account
func (s *SmartContract) BalanceOf(ctx contractapi.TransactionContextInterface, account string) (int, error) {
	balanceBytes, err := ctx.GetStub().GetState(account)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	if balanceBytes == nil {
		return 0, fmt.Errorf("the account %s does not exist", account)
	}

	balance, _ := strconv.Atoi(string(balanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	return balance, nil
}

// ClientAccountBalance returns the balance of the requesting client's account
func (s *SmartContract) ClientAccountBalance(ctx contractapi.TransactionContextInterface) (int, error) {

	// Get ID of submitting client identity
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return 0, fmt.Errorf("failed to get client id: %v", err)
	}

	balanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	if balanceBytes == nil {
		return 0, fmt.Errorf("the account %s does not exist", clientID)
	}

	balance, _ := strconv.Atoi(string(balanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	return balance, nil
}

// ClientAccountID returns the id of the requesting client's account
// In this implementation, the client account ID is the clientId itself
// Users can use this function to get their own account id, which they can then give to others as the payment address
func (s *SmartContract) ClientAccountID(ctx contractapi.TransactionContextInterface) (string, error) {

	// Get ID of submitting client identity
	clientAccountID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client id: %v", err)
	}

	return clientAccountID, nil
}

// TotalSupply returns the total token supply
func (s *SmartContract) TotalSupply(ctx contractapi.TransactionContextInterface) (int, error) {

	// Retrieve total supply of tokens from state of smart contract
	totalSupplyBytes, err := ctx.GetStub().GetState(totalSupplyKey)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve total token supply: %v", err)
	}

	var totalSupply int

	// If no tokens have been minted, return 0
	if totalSupplyBytes == nil {
		totalSupply = 0
	} else {
		totalSupply, _ = strconv.Atoi(string(totalSupplyBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.
	}

	log.Printf("TotalSupply: %d tokens", totalSupply)

	return totalSupply, nil
}

// Approve allows the spender to withdraw from the calling client's token account
// The spender can withdraw multiple times if necessary, up to the value amount
// This function triggers an Approval event
func (s *SmartContract) Approve(ctx contractapi.TransactionContextInterface, spender string, value int) error {

	// Get ID of submitting client identity
	owner, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	// Create allowanceKey
	allowanceKey, err := ctx.GetStub().CreateCompositeKey(allowancePrefix, []string{owner, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	// Update the state of the smart contract by adding the allowanceKey and value
	err = ctx.GetStub().PutState(allowanceKey, []byte(strconv.Itoa(value)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", allowanceKey, err)
	}

	// Emit the Approval event
	approvalEvent := event{owner, spender, value}
	approvalEventJSON, err := json.Marshal(approvalEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.GetStub().SetEvent("Approval", approvalEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("client %s approved a withdrawal allowance of %d for spender %s", owner, value, spender)

	return nil
}

// Allowance returns the amount still available for the spender to withdraw from the owner
func (s *SmartContract) Allowance(ctx contractapi.TransactionContextInterface, owner string, spender string) (int, error) {

	// Create allowanceKey
	allowanceKey, err := ctx.GetStub().CreateCompositeKey(allowancePrefix, []string{owner, spender})
	if err != nil {
		return 0, fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	// Read the allowance amount from the world state
	allowanceBytes, err := ctx.GetStub().GetState(allowanceKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read allowance for %s from world state: %v", allowanceKey, err)
	}

	var allowance int

	// If no current allowance, set allowance to 0
	if allowanceBytes == nil {
		allowance = 0
	} else {
		allowance, _ = strconv.Atoi(string(allowanceBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.
	}

	log.Printf("The allowance left for spender %s to withdraw from owner %s: %d", spender, owner, allowance)

	return allowance, nil
}

// TransferFrom transfers the value amount from the "from" address to the "to" address
// This function triggers a Transfer event
func (s *SmartContract) TransferFrom(ctx contractapi.TransactionContextInterface, from string, to string, value int) error {

	// Get ID of submitting client identity
	spender, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	// Create allowanceKey
	allowanceKey, err := ctx.GetStub().CreateCompositeKey(allowancePrefix, []string{from, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	// Retrieve the allowance of the spender
	currentAllowanceBytes, err := ctx.GetStub().GetState(allowanceKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve the allowance for %s from world state: %v", allowanceKey, err)
	}

	var currentAllowance int
	currentAllowance, _ = strconv.Atoi(string(currentAllowanceBytes)) // Error handling not needed since Itoa() was used when setting the totalSupply, guaranteeing it was an integer.

	// Check if transferred value is less than allowance
	if currentAllowance < value {
		return fmt.Errorf("spender does not have enough allowance for transfer")
	}

	// Initiate the transfer
	err = transferHelper(ctx, from, to, value)
	if err != nil {
		return fmt.Errorf("failed to transfer: %v", err)
	}

	// Decrease the allowance
	updatedAllowance := currentAllowance - value
	err = ctx.GetStub().PutState(allowanceKey, []byte(strconv.Itoa(updatedAllowance)))
	if err != nil {
		return err
	}

	// Emit the Transfer event
	transferEvent := event{from, to, value}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.GetStub().SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	log.Printf("spender %s allowance updated from %d to %d", spender, currentAllowance, updatedAllowance)

	return nil
}

func (s *SmartContract) OrderMint(ctx contractapi.TransactionContextInterface, amount int) error {
	_, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("account does not exist: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	} else if mintburnBytes == nil {
		state := make(map[string]St_am)
		var table St_am

		table.MintBurn = "Mint"
		table.Amount = amount
		table.State = stateOrder

		state[clientID] = table

		mintburn := MintBurn{
			State: state,
		}

		mintburnBytes, err := json.Marshal(mintburn)
		if err != nil {
			return fmt.Errorf("here lies the error: %v", err)
		}

		err = ctx.GetStub().PutState(MintBurnKey, mintburnBytes)
		if err != nil {
			return fmt.Errorf("failed to update MintBurn: %v", err)
		}

		return nil

	} else {

		var mintburn MintBurn
		err = json.Unmarshal(mintburnBytes, &mintburn)
		if err != nil {
			return fmt.Errorf("failed to get json")
		}

		var table St_am

		table.MintBurn = "Mint"
		table.Amount = amount
		table.State = stateOrder

		mintburn.State[clientID] = table

		upd_mintburnBytes, err := json.Marshal(mintburn)
		if err != nil {
			return fmt.Errorf("failed to get bytes")
		}

		err = ctx.GetStub().PutState(MintBurnKey, upd_mintburnBytes)
		if err != nil {
			return fmt.Errorf("failed to update state %v", err)
		}

		return nil
	}
}

func (s *SmartContract) ExecuteMint(ctx contractapi.TransactionContextInterface, amount int) error {
	_, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("account does not exist: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return fmt.Errorf("failed to get json")
	}

	table := mintburn.State[clientID]
	if (table.State != stateApproved) || (table.Amount != amount) || (table.MintBurn != "Mint") {
		return fmt.Errorf("mint is not approved or amount is different than amount ordered")
	}

	err = Mint(ctx, amount)
	if err != nil {
		return fmt.Errorf("error minting amount")
	}

	delete(mintburn.State, clientID)

	upd_mintburnBytes, err := json.Marshal(mintburn)
	if err != nil {
		return fmt.Errorf("failed to get bytes")
	}

	err = ctx.GetStub().PutState(MintBurnKey, upd_mintburnBytes)
	if err != nil {
		return fmt.Errorf("failed to update state %v", err)
	}

	return nil
}

func (s *SmartContract) GetMintOrder(ctx contractapi.TransactionContextInterface) (St_am, error) {
	var mo St_am
	_, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return mo, fmt.Errorf("account does not exist: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return mo, fmt.Errorf("failed to get client id: %v", err)
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return mo, fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return mo, fmt.Errorf("failed to get json")
	}

	mo = mintburn.State[clientID]

	if mo.MintBurn != "Mint" {
		return mo, fmt.Errorf("there is no mint order")
	}

	return mo, nil
}

func (s *SmartContract) OrderBurn(ctx contractapi.TransactionContextInterface, amount int) error {
	_, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("account does not exist: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	} else if mintburnBytes == nil {
		state := make(map[string]St_am)
		var table St_am

		table.MintBurn = "Burn"
		table.Amount = amount
		table.State = stateOrder

		state[clientID] = table

		mintburn := MintBurn{
			State: state,
		}

		mintburnBytes, err := json.Marshal(mintburn)
		if err != nil {
			return fmt.Errorf("here lies the error: %v", err)
		}

		err = ctx.GetStub().PutState(MintBurnKey, mintburnBytes)
		if err != nil {
			return fmt.Errorf("failed to update MintBurn: %v", err)
		}

		return nil

	} else {

		var mintburn MintBurn
		err = json.Unmarshal(mintburnBytes, &mintburn)
		if err != nil {
			return fmt.Errorf("failed to get json")
		}

		var table St_am

		table.MintBurn = "Mint"
		table.Amount = amount
		table.State = stateOrder

		mintburn.State[clientID] = table

		upd_mintburnBytes, err := json.Marshal(mintburn)
		if err != nil {
			return fmt.Errorf("failed to get bytes")
		}

		err = ctx.GetStub().PutState(MintBurnKey, upd_mintburnBytes)
		if err != nil {
			return fmt.Errorf("failed to update state %v", err)
		}

		return nil
	}
}

func (s *SmartContract) ExecuteBurn(ctx contractapi.TransactionContextInterface, amount int) error {
	_, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("account does not exist: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return fmt.Errorf("failed to get json")
	}

	table := mintburn.State[clientID]
	if (table.State != stateApproved) || (table.Amount != amount) || (table.MintBurn != "Burn") {
		return fmt.Errorf("burn is not approved or amount is different than amount ordered")
	}

	err = Burn(ctx, amount)
	if err != nil {
		return err
	}

	delete(mintburn.State, clientID)

	upd_mintburnBytes, err := json.Marshal(mintburn)
	if err != nil {
		return fmt.Errorf("failed to get bytes")
	}

	err = ctx.GetStub().PutState(MintBurnKey, upd_mintburnBytes)
	if err != nil {
		return fmt.Errorf("failed to update state %v", err)
	}

	return nil
}

func (s *SmartContract) GetBurnOrder(ctx contractapi.TransactionContextInterface) (St_am, error) {
	var mo St_am
	_, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return mo, fmt.Errorf("account does not exist: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return mo, fmt.Errorf("failed to get client id: %v", err)
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return mo, fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return mo, fmt.Errorf("failed to get json")
	}

	mo = mintburn.State[clientID]

	if mo.MintBurn != "Burn" {
		return mo, fmt.Errorf("there is no burn order")
	}

	return mo, nil
}

// Helper Functions

// transferHelper is a helper function that transfers tokens from the "from" address to the "to" address
// Dependant functions include Transfer and TransferFrom
func transferHelper(ctx contractapi.TransactionContextInterface, from string, to string, value int) error {

	if value < 0 { // transfer of 0 is allowed in ERC-20, so just validate against negative amounts
		return fmt.Errorf("transfer amount cannot be negative")
	}

	fromCurrentBalanceBytes, err := ctx.GetStub().GetState(from)
	if err != nil {
		return fmt.Errorf("failed to read client account %s from world state: %v", from, err)
	}

	if fromCurrentBalanceBytes == nil {
		return fmt.Errorf("client account %s has no balance", from)
	}

	fromCurrentBalance, _ := strconv.Atoi(string(fromCurrentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.

	if fromCurrentBalance < value {
		return fmt.Errorf("client account %s has insufficient funds", from)
	}

	toCurrentBalanceBytes, err := ctx.GetStub().GetState(to)
	if err != nil {
		return fmt.Errorf("failed to read recipient account %s from world state: %v", to, err)
	}

	var toCurrentBalance int
	// If recipient current balance doesn't yet exist, we'll create it with a current balance of 0
	if toCurrentBalanceBytes == nil {
		toCurrentBalance = 0
	} else {
		toCurrentBalance, _ = strconv.Atoi(string(toCurrentBalanceBytes)) // Error handling not needed since Itoa() was used when setting the account balance, guaranteeing it was an integer.
	}

	fromUpdatedBalance := fromCurrentBalance - value
	toUpdatedBalance := toCurrentBalance + value

	err = ctx.GetStub().PutState(from, []byte(strconv.Itoa(fromUpdatedBalance)))
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(to, []byte(strconv.Itoa(toUpdatedBalance)))
	if err != nil {
		return err
	}

	log.Printf("client %s balance updated from %d to %d", from, fromCurrentBalance, fromUpdatedBalance)
	log.Printf("recipient %s balance updated from %d to %d", to, toCurrentBalance, toUpdatedBalance)

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (s *SmartContract) GetMintOrders(ctx contractapi.TransactionContextInterface) (map[string]St_am, error) {
	var mo map[string]St_am
	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return mo, fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return mo, fmt.Errorf("client is not authorized to get mint orders")
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return mo, fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return mo, fmt.Errorf("there are no Mint Orders")
	}

	mo = mintburn.State

	for key, value := range mo {
		if value.MintBurn != "Mint" {
			delete(mo, key)
		} else if value.State != stateOrder {
			delete(mo, key)
		}
	}

	return mo, nil
}

func (s *SmartContract) GetBurnOrders(ctx contractapi.TransactionContextInterface) (map[string]St_am, error) {
	var mo map[string]St_am
	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return mo, fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return mo, fmt.Errorf("client is not authorized to get burn orders")
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return mo, fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return mo, fmt.Errorf("there are no Burn Orders")
	}

	mo = mintburn.State

	for key, value := range mo {
		if value.MintBurn != "Burn" {
			delete(mo, key)
		} else if value.State != stateOrder {
			delete(mo, key)
		}
	}

	return mo, nil
}

func (s *SmartContract) ApproveMint(ctx contractapi.TransactionContextInterface, mint_acc string) error {
	var mo St_am
	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("client is not authorized to get burn orders")
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return fmt.Errorf("there are no Burn Orders")
	}

	mo = mintburn.State[mint_acc]

	if mo.MintBurn != "Mint" {
		return fmt.Errorf("there are no Mint Orders")
	} else if mo.State != stateOrder {
		return fmt.Errorf("mint is not in order stage")
	}

	mo.State = stateApproved
	mintburn.State[mint_acc] = mo

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

func (s *SmartContract) ApproveBurn(ctx contractapi.TransactionContextInterface, burn_acc string) error {
	var mo St_am
	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("client is not authorized to get burn orders")
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return fmt.Errorf("there are no Burn Orders")
	}

	mo = mintburn.State[burn_acc]

	if mo.MintBurn != "Burn" {
		return fmt.Errorf("there are no Burn Orders")
	} else if mo.State != stateOrder {
		return fmt.Errorf("mint is not in order stage")
	}

	mo.State = stateApproved
	mintburn.State[burn_acc] = mo

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

func (s *SmartContract) RejectMint(ctx contractapi.TransactionContextInterface, mint_acc string) error {
	var mo St_am
	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("client is not authorized to get burn orders")
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return fmt.Errorf("there are no Burn Orders")
	}

	mo = mintburn.State[mint_acc]

	if mo.MintBurn != "Mint" {
		return fmt.Errorf("there are no Mint Orders")
	} else if mo.State != stateOrder {
		return fmt.Errorf("mint is not in order stage")
	}

	mo.State = stateRejected
	mintburn.State[mint_acc] = mo

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

func (s *SmartContract) RejectBurn(ctx contractapi.TransactionContextInterface, burn_acc string) error {
	var mo St_am
	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("client is not authorized to get burn orders")
	}

	mintburnBytes, err := ctx.GetStub().GetState(MintBurnKey)
	if err != nil {
		return fmt.Errorf("failed to read MintBurn from world state: %v", err)
	}

	var mintburn MintBurn
	err = json.Unmarshal(mintburnBytes, &mintburn)
	if err != nil {
		return fmt.Errorf("there are no Burn Orders")
	}

	mo = mintburn.State[burn_acc]

	if mo.MintBurn != "Burn" {
		return fmt.Errorf("there are no Burn Orders")
	} else if mo.State != stateOrder {
		return fmt.Errorf("mint is not in order stage")
	}

	mo.State = stateRejected
	mintburn.State[burn_acc] = mo

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

//check auction, if open and time is up then close it and end it
func (s *SmartContract) CheckAuction(ctx contractapi.TransactionContextInterface, auctionID string) (*Auction, error) {
	var auctionJSON Auction
	// Check authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return &auctionJSON, fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return &auctionJSON, fmt.Errorf("client is not authorized to check auctions")
	}

	// get the auction from state
	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return &auctionJSON, fmt.Errorf("couldn't get auction from global state")
	}

	if auctionBytes == nil {
		return &auctionJSON, fmt.Errorf("Auction not found: %v", auctionID)
	}
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return &auctionJSON, fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "open" {
		return &auctionJSON, fmt.Errorf("auction closed or ended auction")
	}

	t := int(time.Since(auctionJSON.Time_started).Minutes())
	if t >= auctionJSON.Time_remaining {
		_ = CloseAuction(ctx, auctionID)
		_ = EndAuction(ctx, auctionID)
		return &auctionJSON, fmt.Errorf("auction closed and ended")
	}

	return &auctionJSON, nil
}

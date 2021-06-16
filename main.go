package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	e_moneySmartContract, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating auction chaincode: %v", err)
	}

	if err := e_moneySmartContract.Start(); err != nil {
		log.Panicf("Error starting auction chaincode: %v", err)
	}

}

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Auction struct {
	Type           string             `json:"objectType"`
	ItemSold       string             `json:"item"`
	Amount         int                `json:"amount"`
	PricePerKWh    int                `json:"priceperkwh"`
	Time_started   time.Time          `json:"time_started"`
	Time_remaining int                `json:"time_remaining"`
	Seller         string             `json:"seller"`
	Orgs           []string           `json:"organizations"`
	PrivateBids    map[string]BidHash `json:"privateBids"`
	RevealedBids   map[string]FullBid `json:"revealedBids"`
	Winner         string             `json:"winner"`
	Price          int                `json:"price"`
	Status         string             `json:"status"`
}

// FullBid is the structure of a revealed bid
type FullBid struct {
	Type   string `json:"objectType"`
	Price  int    `json:"price"`
	Org    string `json:"org"`
	Bidder string `json:"bidder"`
}

// BidHash is the structure of a private bid
type BidHash struct {
	Org  string `json:"org"`
	Hash string `json:"hash"`
}

const bidKeyType = "bid"

// CreateAuction creates on auction on the public channel. The identity that
// submits the transacion becomes the seller of the auction
func (s *SmartContract) CreateAuction(ctx contractapi.TransactionContextInterface, auctionID string, priceperkwh int, amount int, time_rem int) error { //amount = how many kwh

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// get org of submitting client
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// Create auction
	bidders := make(map[string]BidHash)
	revealedBids := make(map[string]FullBid)
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp")
	}

	time := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)) //.String()

	auction := Auction{
		Type:           "auction",
		ItemSold:       "energy(KWh)",
		Amount:         amount,
		PricePerKWh:    priceperkwh,
		Time_started:   time,
		Time_remaining: time_rem,
		Price:          amount * priceperkwh,
		Seller:         clientID,
		Orgs:           []string{clientOrgID},
		PrivateBids:    bidders,
		RevealedBids:   revealedBids,
		Winner:         "",
		Status:         "open",
	}

	auctionBytes, err := json.Marshal(auction)
	if err != nil {
		return err
	}

	// put auction into state
	err = ctx.GetStub().PutState(auctionID, auctionBytes)
	if err != nil {
		return fmt.Errorf("failed to put auction in public data: %v", err)
	}

	// set the seller of the auction as an endorser
	err = setAssetStateBasedEndorsement(ctx, auctionID, clientOrgID)
	if err != nil {
		return fmt.Errorf("failed setting state based endorsement for new organization: %v", err)
	}

	return nil
}

// SubmitBid is used by the bidder to add the hash of that bid stored in private data to the
// auction. Note that this function alters the auction in private state, and needs
// to meet the auction endorsement policy. Transaction ID is used identify the bid
func (s *SmartContract) Bid_Rev(ctx contractapi.TransactionContextInterface, auctionID string, amount int) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	// get the MSP ID of the bidder's org
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	// get the auction from state
	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("couldn't get auction from global state")
	}
	var auctionJSON Auction

	if auctionBytes == nil {
		return fmt.Errorf("Auction not found: %v", auctionID)
	}
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// the auction needs to be open for users to add their bid
	Status := auctionJSON.Status
	if Status != "open" {
		return fmt.Errorf("cannot join closed or ended auction")
	}

	t := int(time.Since(auctionJSON.Time_started).Minutes())
	if t >= auctionJSON.Time_remaining {
		_ = CloseAuction(ctx, auctionID)
		_ = EndAuction(ctx, auctionID)
		return fmt.Errorf("time is up")
	}

	balance, err := s.ClientAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("cannot get balance")
	}
	if balance < amount {
		return fmt.Errorf("balance is less than amount")
	}

	// use the transaction ID passed as a parameter to create composite bid key
	bidKey, err := ctx.GetStub().CreateCompositeKey(bidKeyType, []string{auctionID})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	NewBid := FullBid{
		Type:   auctionJSON.ItemSold,
		Price:  amount,
		Org:    clientOrgID,
		Bidder: clientID,
	}

	bidders := make(map[string]FullBid)
	bidders = auctionJSON.RevealedBids
	bidders[bidKey] = NewBid
	auctionJSON.RevealedBids = bidders

	// Add the bidding organization to the list of participating organizations if it is not already
	Orgs := auctionJSON.Orgs
	if !(contains(Orgs, clientOrgID)) {
		newOrgs := append(Orgs, clientOrgID)
		auctionJSON.Orgs = newOrgs

		err = addAssetStateBasedEndorsement(ctx, auctionID, clientOrgID)
		if err != nil {
			return fmt.Errorf("failed setting state based endorsement for new organization: %v", err)
		}
	}

	newAuctionBytes, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, newAuctionBytes)
	if err != nil {
		return fmt.Errorf("failed to update auction: %v", err)
	}

	err = s.CreateHold(ctx, amount)
	if err != nil {
		return fmt.Errorf("cannot create hold: %v", err)
	}

	return nil
}

// CloseAuction can be used by the seller to close the auction. This prevents
// bids from being added to the auction, and allows users to reveal their bid
func (s *SmartContract) CloseAuction(ctx contractapi.TransactionContextInterface, auctionID string) error {

	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// the auction can only be closed by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be closed by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "open" {
		return fmt.Errorf("cannot close auction that is not open")
	}

	auctionJSON.Status = string("closed")

	closedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, closedAuction)
	if err != nil {
		return fmt.Errorf("failed to close auction: %v", err)
	}

	return nil
}

// CloseAuction can be used by the seller to close the auction. This prevents
// bids from being added to the auction, and allows users to reveal their bid
func CloseAuction(ctx contractapi.TransactionContextInterface, auctionID string) error {

	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// the auction can only be closed by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be closed by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "open" {
		return fmt.Errorf("cannot close auction that is not open")
	}

	auctionJSON.Status = string("closed")

	closedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, closedAuction)
	if err != nil {
		return fmt.Errorf("failed to close auction: %v", err)
	}

	return nil
}

// EndAuction both changes the auction status to closed and calculates the winners
// of the auction
func (s *SmartContract) EndAuction(ctx contractapi.TransactionContextInterface, auctionID string) error {

	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// Check that the auction is being ended by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be ended by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "closed" {
		return fmt.Errorf("can only end a closed auction")
	}

	// get the list of revealed bids
	revealedBidMap := auctionJSON.RevealedBids
	if len(auctionJSON.RevealedBids) == 0 {
		return fmt.Errorf("no bids have been revealed, cannot end auction: %v", err)
	}

	// determine the highest bid
	for _, bid := range revealedBidMap {
		if bid.Price > auctionJSON.Price {
			auctionJSON.Winner = bid.Bidder
			auctionJSON.Price = bid.Price
		}
	}

	// check if there is a winning bid that has yet to be revealed
	err = queryAllBids(ctx, auctionJSON.Price, auctionJSON.RevealedBids, auctionJSON.PrivateBids)
	if err != nil {
		return fmt.Errorf("cannot close auction: %v", err)
	}

	auctionJSON.Status = string("ended")

	closedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, closedAuction)
	if err != nil {
		return fmt.Errorf("failed to end auction: %v", err)
	}

	err = ctx.GetStub().DelState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to delete auction: %v", err)
	}

	return nil
}

// EndAuction both changes the auction status to closed and calculates the winners
// of the auction
func EndAuction(ctx contractapi.TransactionContextInterface, auctionID string) error {

	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// Check that the auction is being ended by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be ended by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "closed" {
		return fmt.Errorf("can only end a closed auction")
	}

	// get the list of revealed bids
	revealedBidMap := auctionJSON.RevealedBids
	if len(auctionJSON.RevealedBids) == 0 {
		return fmt.Errorf("no bids have been revealed, cannot end auction: %v", err)
	}

	// determine the highest bid
	for _, bid := range revealedBidMap {
		if bid.Price > auctionJSON.Price {
			auctionJSON.Winner = bid.Bidder
			auctionJSON.Price = bid.Price
		}
	}

	// check if there is a winning bid that has yet to be revealed
	err = queryAllBids(ctx, auctionJSON.Price, auctionJSON.RevealedBids, auctionJSON.PrivateBids)
	if err != nil {
		return fmt.Errorf("cannot close auction: %v", err)
	}

	auctionJSON.Status = string("ended")

	closedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, closedAuction)
	if err != nil {
		return fmt.Errorf("failed to close auction: %v", err)
	}

	err = ctx.GetStub().DelState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to delete auction: %v", err)
	}

	return nil
}

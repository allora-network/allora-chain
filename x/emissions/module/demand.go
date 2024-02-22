package module

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"

	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A structure to hold the original value and a random tiebreaker
type SortableItem[T any] struct {
	Value      T
	Weight     uint64
	Tiebreaker float64
}

type RequestId = string

type TopicId = uint64

type PriceAndReturn = struct {
	Price  cosmosMath.Uint
	Return cosmosMath.Uint
}

type Demand struct {
	Requests      []state.InferenceRequest
	FeesGenerated cosmosMath.Uint
}

// Sorts the given slice of topics in descending order according to their corresponding return, using randomness as tiebreaker
// e.g. ([]uint64{1, 2, 3}, map[uint64]uint64{1: 2, 2: 2, 3: 3}, 0) -> [3, 1, 2] or [3, 2, 1]
func SortTopicsByReturnDescWithRandomTiebreaker(valsToSort []state.Topic, weights map[TopicId]PriceAndReturn, randSeed uint64) []state.Topic {
	// Convert the slice of Ts to a slice of SortableItems, each with a random tiebreaker
	r := rand.New(rand.NewSource(int64(randSeed)))
	items := make([]SortableItem[state.Topic], len(valsToSort))
	for i, topic := range valsToSort {
		items[i] = SortableItem[state.Topic]{topic, weights[topic.Id].Price.Uint64(), r.Float64()}
	}

	// Sort the slice of SortableItems
	// If the values are equal, the tiebreaker will decide their order
	sort.Slice(items, func(i, j int) bool {
		if items[i].Value == items[j].Value {
			return items[i].Tiebreaker > items[j].Tiebreaker
		}
		return items[i].Weight > items[j].Weight
	})

	// Extract and print the sorted values to demonstrate the sorting
	sortedValues := make([]state.Topic, len(valsToSort))
	for i, item := range items {
		sortedValues[i] = item.Value
	}
	return sortedValues
}

// Check that a request:
//
//	should be checked in this timestep
//	AND did not expire
//	AND would have enough funds to cover the potential next price
//	AND admits and acceptable max price per inference
func IsValidAtPrice(
	ctx sdk.Context,
	k keeper.Keeper,
	req state.InferenceRequest,
	price cosmosMath.Uint,
	currentTime uint64) (bool, error) {
	reqId, err := req.GetRequestId()
	if err != nil {
		fmt.Println("Error getting request id: ", err)
		return false, err
	}
	reqUnmetDemand, err := k.GetRequestDemand(ctx, reqId)
	if err != nil {
		fmt.Println("Error getting request demand: ", err)
		return false, err
	}
	res :=
		req.LastChecked+req.Cadence <= currentTime &&
			req.TimestampValidUntil > currentTime &&
			reqUnmetDemand.GTE(price) &&
			req.MaxPricePerInference.GTE(price)
	return res, nil
}

// Inactivates topics with below keeper.MIN_TOPIC_DEMAND demand
// returns a list of topics that are still active after this operation
func InactivateLowDemandTopics(ctx context.Context, k keeper.Keeper) (remainingActiveTopics []*state.Topic, err error) {
	topicsActive, err := k.GetActiveTopics(ctx)
	remainingActiveTopics = make([]*state.Topic, 0)
	if err != nil {
		fmt.Println("Error getting active topics: ", err)
		return nil, err
	}
	minTopicDemand := cosmosMath.NewUint(keeper.MIN_TOPIC_DEMAND)
	for _, topic := range topicsActive {
		topicUnmetDemand, err := k.GetTopicUnmetDemand(ctx, topic.Id)
		if err != nil {
			fmt.Println("Error getting unmet demand: ", err)
			return nil, err
		}
		if topicUnmetDemand.LT(minTopicDemand) {
			fmt.Printf("Inactivating topic due to no demand: %v metadata: %s\n", topic.Id, topic.Metadata)
			k.InactivateTopic(ctx, topic.Id)
		} else {
			remainingActiveTopics = append(remainingActiveTopics, topic)
		}
	}
	return remainingActiveTopics, nil
}

// Generate a demand curve, which is a data structure that captures the price
// that maximizes the demand drawn from valid requests. Each price is mapped to the list of people
// willing to pay AT LEAST that price for inference. Then the maximum amount of fees is found
// by multiplying the price by the number of requests willing to pay at least that price.
// TODO: think if we can sort the data structure first, then process it in order to do O(2*n)
// probably we can use some kind of ordered tree to do this
// instead of what we are currently doing which is O(n^2)
func GetRequestsThatMaxFees(
	ctx sdk.Context,
	k keeper.Keeper,
	currentTime uint64,
	requestsForGivenTopic []state.InferenceRequest) (
	bestPrice cosmosMath.Uint,
	maxFees cosmosMath.Uint,
	requests []state.InferenceRequest,
	err error) {
	// Initialize a map of request price to map of valid requests
	// map must be of type string, because the complex type of a Uint
	// will not work for the map, equality test tests the pointer not the value
	demandCurve := make(map[string]Demand)
	requests = make([]state.InferenceRequest, 0)
	maxFees = cosmosMath.NewUint(0)
	bestPrice = cosmosMath.NewUint(0)
	// Loop through inference requests and then loop again (nested) checking validity of all other inferences at the first inference's max price
	for _, req := range requestsForGivenTopic {
		// Check validity of current request at its own price
		isValidAtPrice, err := IsValidAtPrice(ctx, k, req, req.MaxPricePerInference, currentTime)
		if err != nil {
			fmt.Println("Error checking if request is valid at price: ", err)
			return cosmosMath.Uint{}, cosmosMath.Uint{}, nil, err
		}
		if isValidAtPrice {
			price := req.MaxPricePerInference
			priceStr := price.String()
			_, exists := demandCurve[priceStr]
			if exists {
				// if the demand curve has already computed the list of buyers
				// at this price level before, we dont need to do it again
				continue
			}
			demandCurve[priceStr] = Demand{
				Requests:      make([]state.InferenceRequest, 0),
				FeesGenerated: cosmosMath.ZeroUint()}

			for _, req2 := range requestsForGivenTopic {
				isValidAtPrice, err := IsValidAtPrice(ctx, k, req2, price, currentTime)
				if err != nil {
					fmt.Println("Error checking if request is valid at price: ", err)
					return cosmosMath.Uint{}, cosmosMath.Uint{}, nil, err
				}
				if isValidAtPrice {
					newFeesGenerated := demandCurve[priceStr].FeesGenerated.Add(price)
					newRequests := append(demandCurve[priceStr].Requests, req2)
					demandCurve[priceStr] = Demand{
						Requests:      newRequests,
						FeesGenerated: newFeesGenerated}
					if newFeesGenerated.GT(maxFees) {
						maxFees = newFeesGenerated
						bestPrice = price
					}
				}
			}
		}
	}
	if !bestPrice.IsZero() {
		requests = demandCurve[bestPrice.String()].Requests
	}
	return bestPrice, maxFees, requests, nil
}

// The price of inference for a topic is determined by the price that maximizes the demand drawn from valid requests.
// Which topics get processed (inference solicitation and weight-adjustment) is based on ordering topics by their return
// at their optimal prices and then skimming the top.
func ChurnAndDrawFromRequestsToGetTopActiveTopicsAndMetDemand(ctx sdk.Context, k keeper.Keeper, currentTime uint64) (*[]state.Topic, *cosmosMath.Uint, error) {
	topicsActive, err := InactivateLowDemandTopics(ctx, k)
	if err != nil {
		fmt.Println("Error getting active topics: ", err)
		return nil, nil, err
	}
	fmt.Println("Active topics: ", len(topicsActive))

	topicsActiveWithDemand := make([]state.Topic, 0)
	topicBestPrices := make(map[TopicId]PriceAndReturn)
	requestsToDrawDemandFrom := make(map[TopicId][]state.InferenceRequest, 0)
	for _, topic := range topicsActive {
		inferenceRequests, err := k.GetMempoolInferenceRequestsForTopic(ctx, topic.Id)
		if err != nil {
			fmt.Println("Error getting mempool inference requests: ", err)
			return nil, nil, err
		}

		priceOfMaxReturn, maxReturn, requestsToUse, err := GetRequestsThatMaxFees(ctx, k, currentTime, inferenceRequests)
		if err != nil {
			fmt.Println("Error getting requests that maximize fees: ", err)
			return nil, nil, err
		}
		topicBestPrices[topic.Id] = PriceAndReturn{priceOfMaxReturn, maxReturn}
		requestsToDrawDemandFrom[topic.Id] = requestsToUse
	}

	// Sort topics by topicBestPrices
	sortedTopics := SortTopicsByReturnDescWithRandomTiebreaker(topicsActiveWithDemand, topicBestPrices, currentTime)
	// Take top keeper.MAX_TOPICS_PER_BLOCK number of topics with the highest demand
	topTopicsByReturn := sortedTopics[:uint(math.Min(float64(len(sortedTopics)), keeper.MAX_TOPICS_PER_BLOCK))]

	// Determine how many funds to draw from demand and Remove depleted/insufficiently funded requests
	totalFundsToDrawFromDemand := cosmosMath.NewUint(0)
	for _, topic := range topTopicsByReturn {
		// Log the accumulated met demand for each topic
		k.AddTopicAccumulateMetDemand(ctx, topic.Id, topicBestPrices[topic.Id].Return)

		// Draw demand from the valid requests
		bestPrice := topicBestPrices[topic.Id].Price
		numRequestsServed := 0
		for _, req := range requestsToDrawDemandFrom[topic.Id] {
			reqId, err := req.GetRequestId()
			if err != nil {
				fmt.Println("Error getting request demand: ", err)
				return nil, nil, err
			}
			reqDemand, err := k.GetRequestDemand(ctx, reqId)
			if err != nil {
				fmt.Println("Error getting request demand: ", err)
				return nil, nil, err
			}
			// all the previous conditionals were already applied to the requests in the previous loop
			// => should never be negative
			newReqDemand := reqDemand.Sub(bestPrice)
			k.SetRequestDemand(ctx, reqId, newReqDemand)
			if newReqDemand.LT(cosmosMath.NewUint(keeper.MIN_UNMET_DEMAND)) { // TylerTODO check cadence and remove if one-shot
				// Should convey to users to not surprise them. This helps prevent spamming the mempool with requests that are not worth serving
				// The effectively burned dust is 1-time "cost" the consumer incurs when they create "subscriptions" they don't ever refill nor fill enough
				// This encourages consumers to maximize how much they fund any single request, discouraging a pattern of many less-funded requests
				k.RemoveFromMempool(ctx, req)
			}
			numRequestsServed++
		}
		totalFundsToDrawFromDemand = totalFundsToDrawFromDemand.Add(bestPrice.Mul(cosmosMath.NewUint(uint64(numRequestsServed))))
	}

	return &topTopicsByReturn, &totalFundsToDrawFromDemand, nil
}

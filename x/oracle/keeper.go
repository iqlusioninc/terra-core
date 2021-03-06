package oracle

import (
	"strings"

	"github.com/terra-project/core/types"
	"github.com/terra-project/core/types/assets"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Keeper of the oracle store
type Keeper struct {
	cdc *codec.Codec
	key sdk.StoreKey

	mk  MintKeeper
	dk  DistributionKeeper
	fck FeeCollectionKeeper

	valset     sdk.ValidatorSet
	paramSpace params.Subspace
}

// NewKeeper constructs a new keeper for oracle
func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, mk MintKeeper, dk DistributionKeeper, fck FeeCollectionKeeper,
	valset sdk.ValidatorSet, paramspace params.Subspace) Keeper {
	return Keeper{
		cdc: cdc,
		key: key,

		mk:  mk,
		dk:  dk,
		fck: fck,

		valset:     valset,
		paramSpace: paramspace.WithKeyTable(paramKeyTable()),
	}
}

//-----------------------------------
// Prevote logic

// Iterate over prevotes in the store
func (k Keeper) iteratePrevotes(ctx sdk.Context, handler func(prevote PricePrevote) (stop bool)) {
	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefixPrevote)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var prevote PricePrevote
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &prevote)
		if handler(prevote) {
			break
		}
	}
}

// Iterate over votes in the store
func (k Keeper) iteratePrevotesWithPrefix(ctx sdk.Context, prefix []byte, handler func(vote PricePrevote) (stop bool)) {
	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var prevote PricePrevote
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &prevote)
		if handler(prevote) {
			break
		}
	}
}

//-----------------------------------
// Votes logic

// collectVotes collects all oracle votes for the period, categorized by the votes' denom parameter
func (k Keeper) collectVotes(ctx sdk.Context) (votes map[string]PriceBallot) {
	votes = map[string]PriceBallot{}
	handler := func(vote PriceVote) (stop bool) {
		votes[vote.Denom] = append(votes[vote.Denom], vote)
		return false
	}
	k.iterateVotes(ctx, handler)

	return
}

// Iterate over votes in the store
func (k Keeper) iterateVotes(ctx sdk.Context, handler func(vote PriceVote) (stop bool)) {
	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefixVote)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var vote PriceVote
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &vote)
		if handler(vote) {
			break
		}
	}
}

// Iterate over votes in the store
func (k Keeper) iterateVotesWithPrefix(ctx sdk.Context, prefix []byte, handler func(vote PriceVote) (stop bool)) {
	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var vote PriceVote
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &vote)
		if handler(vote) {
			break
		}
	}
}

// Retrieves a prevote from the store
func (k Keeper) getPrevote(ctx sdk.Context, denom string, voter sdk.ValAddress) (prevote PricePrevote, err sdk.Error) {
	store := ctx.KVStore(k.key)
	b := store.Get(keyPrevote(denom, voter))
	if b == nil {
		err = ErrNoPrevote(DefaultCodespace, voter, denom)
		return
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &prevote)
	return
}

// Add a prevote to the store
func (k Keeper) addPrevote(ctx sdk.Context, prevote PricePrevote) {
	store := ctx.KVStore(k.key)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(prevote)
	store.Set(keyPrevote(prevote.Denom, prevote.Voter), bz)
}

// Delete a prevote from the store
func (k Keeper) deletePrevote(ctx sdk.Context, prevote PricePrevote) {
	store := ctx.KVStore(k.key)
	store.Delete(keyPrevote(prevote.Denom, prevote.Voter))
}

// Retrieves a vote from the store
func (k Keeper) getVote(ctx sdk.Context, denom string, voter sdk.ValAddress) (vote PriceVote, err sdk.Error) {
	store := ctx.KVStore(k.key)
	b := store.Get(keyVote(denom, voter))
	if b == nil {
		err = ErrNoVote(DefaultCodespace, voter, denom)
		return
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &vote)
	return
}

// Add a vote to the store
func (k Keeper) addVote(ctx sdk.Context, vote PriceVote) {
	store := ctx.KVStore(k.key)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(vote)
	store.Set(keyVote(vote.Denom, vote.Voter), bz)
}

// Delete a vote from the store
func (k Keeper) deleteVote(ctx sdk.Context, vote PriceVote) {
	store := ctx.KVStore(k.key)
	store.Delete(keyVote(vote.Denom, vote.Voter))
}

//-----------------------------------
// Price logic

// GetLunaSwapRate gets the consensus exchange rate of Luna denominated in the denom asset from the store.
func (k Keeper) GetLunaSwapRate(ctx sdk.Context, denom string) (price sdk.Dec, err sdk.Error) {
	if denom == assets.MicroLunaDenom {
		return sdk.OneDec(), nil
	}

	store := ctx.KVStore(k.key)
	b := store.Get(keyPrice(denom))
	if b == nil {
		return sdk.ZeroDec(), ErrUnknownDenomination(DefaultCodespace, denom)
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &price)
	return
}

// SetLunaSwapRate sets the consensus exchange rate of Luna denominated in the denom asset to the store.
func (k Keeper) SetLunaSwapRate(ctx sdk.Context, denom string, price sdk.Dec) {
	store := ctx.KVStore(k.key)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(price)
	store.Set(keyPrice(denom), bz)
}

// deletePrice deletes the consensus exchange rate of Luna denominated in the denom asset from the store.
func (k Keeper) deletePrice(ctx sdk.Context, denom string) {
	store := ctx.KVStore(k.key)
	store.Delete(keyPrice(denom))
}

// Get all active oracle asset denoms from the store
func (k Keeper) getActiveDenoms(ctx sdk.Context) (denoms DenomList) {
	denoms = DenomList{}

	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefixPrice)
	for ; iter.Valid(); iter.Next() {
		n := len(prefixPrice) + 1
		denom := string(iter.Key()[n:])
		denoms = append(denoms, denom)
	}
	iter.Close()

	return
}

//-----------------------------------
// Params logic

// GetParams get oracle params from the global param store
func (k Keeper) GetParams(ctx sdk.Context) Params {
	var params Params
	k.paramSpace.Get(ctx, paramStoreKeyParams, &params)
	return params
}

// SetParams set oracle params from the global param store
func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramSpace.Set(ctx, paramStoreKeyParams, &params)
}

//-----------------------------------
// Feeder delegation logic

// GetFeedDelegate gets the account address that the feeder right was delegated to by the validator operator.
func (k Keeper) GetFeedDelegate(ctx sdk.Context, operator sdk.ValAddress) (delegate sdk.AccAddress) {
	store := ctx.KVStore(k.key)
	b := store.Get(keyFeederDelegation(operator))
	if b == nil {
		// By default the right is delegated to the validator itself
		return sdk.AccAddress(operator)
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &delegate)
	return
}

// SetFeedDelegate sets the account address that the feeder right was delegated to by the validator operator.
func (k Keeper) SetFeedDelegate(ctx sdk.Context, operator sdk.ValAddress, delegatedFeeder sdk.AccAddress) {
	store := ctx.KVStore(k.key)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(delegatedFeeder)
	store.Set(keyFeederDelegation(operator), bz)
}

// Iterate over feeder delegations in the store
func (k Keeper) iterateFeederDelegations(ctx sdk.Context, handler func(delegate sdk.AccAddress, operator sdk.ValAddress) (stop bool)) {
	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefixFeederDelegation)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		operatorAddress := strings.Split(string(iter.Key()), ":")[1]
		operator, _ := sdk.ValAddressFromBech32(operatorAddress)

		var delegate sdk.AccAddress
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &delegate)
		if handler(delegate, operator) {
			break
		}
	}
}

//-----------------------------------
// Swap fee pool logic

// GetSwapFeePool retrieves the swap fee pool from the store
func (k Keeper) GetSwapFeePool(ctx sdk.Context) (pool sdk.Coins) {
	store := ctx.KVStore(k.key)
	b := store.Get(keySwapFeePool)
	if b == nil {
		return sdk.Coins{}
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &pool)
	return
}

// setSwapFeePool sets the swap fee pool to the store
func (k Keeper) AddSwapFeePool(ctx sdk.Context, fees sdk.Coins) {
	pool := k.GetSwapFeePool(ctx)
	pool = pool.Add(fees)

	store := ctx.KVStore(k.key)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(pool)
	store.Set(keySwapFeePool, bz)
}

// clearSwapFeePool clears the swap fee pool from the store
func (k Keeper) clearSwapFeePool(ctx sdk.Context) {
	store := ctx.KVStore(k.key)
	store.Delete(keySwapFeePool)
}

//-----------------------------------
// Claim pool logic

// Iterate over oracle reward claims in the store
func (k Keeper) iterateClaimPool(ctx sdk.Context, handler func(recipient sdk.AccAddress, weight sdk.Int) (stop bool)) {
	store := ctx.KVStore(k.key)
	iter := sdk.KVStorePrefixIterator(store, prefixClaim)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		recipientAddress := strings.Split(string(iter.Key()), ":")[1]
		recipient, _ := sdk.AccAddressFromBech32(recipientAddress)

		var weight sdk.Int
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &weight)
		if handler(recipient, weight) {
			break
		}
	}
}

// addClaimPool adds a claim to the the claim pool in the store
func (k Keeper) addClaimPool(ctx sdk.Context, pool types.ClaimPool) {
	store := ctx.KVStore(k.key)

	for _, claim := range pool {
		storeKeyClaim := keyClaim(claim.Recipient)
		b := store.Get(storeKeyClaim)
		weight := claim.Weight
		if b != nil {
			var prevWeight sdk.Int
			k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &prevWeight)

			weight = weight.Add(prevWeight)
		}
		b = k.cdc.MustMarshalBinaryLengthPrefixed(weight)
		store.Set(storeKeyClaim, b)
	}
}

// clearClaimPool clears the claim pool from the store
func (k Keeper) clearClaimPool(ctx sdk.Context) {
	store := ctx.KVStore(k.key)
	k.iterateClaimPool(ctx, func(recipient sdk.AccAddress, weight sdk.Int) (stop bool) {
		store.Delete(keyClaim(recipient))
		return false
	})
}

package oracle

import (
	"encoding/hex"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/terra-project/core/types/assets"
)

func TestOracleThreshold(t *testing.T) {
	input, h := setup(t)

	// Less than the threshold signs, msg fails
	// Prevote without price
	salt := "1"
	bz, err := VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg := NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	res := h(input.ctx.WithBlockHeight(0), prevoteMsg)
	require.True(t, res.IsOK())

	// Vote and new Prevote
	voteMsg := NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	res = h(input.ctx.WithBlockHeight(1), voteMsg)
	require.True(t, res.IsOK())

	EndBlocker(input.ctx.WithBlockHeight(1), input.oracleKeeper)

	_, err = input.oracleKeeper.GetLunaSwapRate(input.ctx.WithBlockHeight(1), assets.MicroSDRDenom)
	require.NotNil(t, err)

	// More than the threshold signs, msg succeeds
	salt = "1"
	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg = NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	salt = "2"
	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[1]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg = NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	salt = "3"
	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[2]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[2], sdk.ValAddress(addrs[2]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg = NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[2], sdk.ValAddress(addrs[2]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	EndBlocker(input.ctx.WithBlockHeight(1), input.oracleKeeper)

	price, err := input.oracleKeeper.GetLunaSwapRate(input.ctx.WithBlockHeight(1), assets.MicroSDRDenom)
	require.Nil(t, err)
	require.Equal(t, randomPrice, price)

	// Less than the threshold signs, msg fails
	val, _ := input.stakingKeeper.GetValidator(input.ctx, sdk.ValAddress(addrs[2]))
	input.stakingKeeper.Delegate(input.ctx.WithBlockHeight(0), addrs[2], uLunaAmt.MulRaw(2), val, false)

	salt = "1"
	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg = NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	salt = "2"
	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[1]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg = NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	EndBlocker(input.ctx.WithBlockHeight(1), input.oracleKeeper)

	price, err = input.oracleKeeper.GetLunaSwapRate(input.ctx.WithBlockHeight(1), assets.MicroSDRDenom)
	require.NotNil(t, err)
}

func TestOracleMultiVote(t *testing.T) {
	input, h := setup(t)

	// Less than the threshold signs, msg fails
	salt := "1"
	bz, err := VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg := NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	res := h(input.ctx, prevoteMsg)
	require.True(t, res.IsOK())

	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[1]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	res = h(input.ctx, prevoteMsg)
	require.True(t, res.IsOK())

	bz, err = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[2]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[2], sdk.ValAddress(addrs[2]))
	res = h(input.ctx, prevoteMsg)
	require.True(t, res.IsOK())

	bz, err = VoteHash(salt, anotherRandomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	res = h(input.ctx, prevoteMsg)
	require.True(t, res.IsOK())

	bz, err = VoteHash(salt, anotherRandomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[1]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	res = h(input.ctx, prevoteMsg)
	require.True(t, res.IsOK())

	bz, err = VoteHash(salt, anotherRandomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[2]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[2], sdk.ValAddress(addrs[2]))
	res = h(input.ctx, prevoteMsg)
	require.True(t, res.IsOK())

	// Reveal Price
	input.ctx = input.ctx.WithBlockHeight(1)
	voteMsg := NewMsgPriceVote(anotherRandomPrice, salt, assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	res = h(input.ctx, voteMsg)
	require.True(t, res.IsOK())

	voteMsg = NewMsgPriceVote(anotherRandomPrice, salt, assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	res = h(input.ctx, voteMsg)
	require.True(t, res.IsOK())

	voteMsg = NewMsgPriceVote(anotherRandomPrice, salt, assets.MicroSDRDenom, addrs[2], sdk.ValAddress(addrs[2]))
	res = h(input.ctx, voteMsg)
	require.True(t, res.IsOK())

	EndBlocker(input.ctx, input.oracleKeeper)

	price, err := input.oracleKeeper.GetLunaSwapRate(input.ctx, assets.MicroSDRDenom)
	require.Nil(t, err)
	require.Equal(t, price, anotherRandomPrice)
}

func TestOracleDrop(t *testing.T) {
	input, h := setup(t)

	input.oracleKeeper.SetLunaSwapRate(input.ctx, assets.MicroKRWDenom, randomPrice)

	salt := "1"
	bz, err := VoteHash(salt, randomPrice, assets.MicroKRWDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg := NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroKRWDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx, prevoteMsg)

	input.ctx = input.ctx.WithBlockHeight(1)
	voteMsg := NewMsgPriceVote(randomPrice, salt, assets.MicroKRWDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx, voteMsg)

	// Immediately swap halt after an illiquid oracle vote
	EndBlocker(input.ctx, input.oracleKeeper)

	_, err = input.oracleKeeper.GetLunaSwapRate(input.ctx, assets.MicroKRWDenom)
	require.NotNil(t, err)
}

func TestOracleTally(t *testing.T) {
	input, _ := setup(t)

	ballot := PriceBallot{}
	prices, valAccAddrs, mockValset := generateRandomTestCase()
	input.oracleKeeper.valset = mockValset
	h := NewHandler(input.oracleKeeper)
	for i, price := range prices {

		decPrice := sdk.NewDecWithPrec(int64(price*math.Pow10(oracleDecPrecision)), int64(oracleDecPrecision))

		salt := string(i)
		bz, err := VoteHash(salt, decPrice, assets.MicroSDRDenom, sdk.ValAddress(valAccAddrs[i]))
		require.Nil(t, err)

		prevoteMsg := NewMsgPricePrevote(
			hex.EncodeToString(bz),
			assets.MicroSDRDenom,
			valAccAddrs[i],
			sdk.ValAddress(valAccAddrs[i]),
		)

		res := h(input.ctx.WithBlockHeight(0), prevoteMsg)
		require.True(t, res.IsOK())

		voteMsg := NewMsgPriceVote(
			decPrice,
			salt,
			assets.MicroSDRDenom,
			valAccAddrs[i],
			sdk.ValAddress(valAccAddrs[i]),
		)

		res = h(input.ctx.WithBlockHeight(1), voteMsg)
		require.True(t, res.IsOK())

		vote := NewPriceVote(decPrice, assets.MicroSDRDenom, sdk.ValAddress(valAccAddrs[i]))
		ballot = append(ballot, vote)

		// change power of every three validator
		if i%3 == 0 {
			mockValset.Validators[i].Power = sdk.NewInt(int64(i + 1))
		}
	}

	rewardees := []sdk.AccAddress{}
	weightedMedian := ballot.weightedMedian(input.ctx, mockValset)
	maxSpread := input.oracleKeeper.GetParams(input.ctx).OracleRewardBand.QuoInt64(2)

	for _, vote := range ballot {
		if vote.Price.GTE(weightedMedian.Sub(maxSpread)) && vote.Price.LTE(weightedMedian.Add(maxSpread)) {
			rewardees = append(rewardees, sdk.AccAddress(vote.Voter))
		}
	}

	tallyMedian := tally(input.ctx, input.oracleKeeper, ballot)

	require.Equal(t, countClaimPool(input.ctx, input.oracleKeeper), len(rewardees))
	require.Equal(t, tallyMedian.MulInt64(100).TruncateInt(), weightedMedian.MulInt64(100).TruncateInt())
}

func TestOracleTallyTiming(t *testing.T) {
	input, h := setup(t)

	// all the addrs vote for the block ... not last period block yet, so tally fails
	for _, addr := range addrs {
		salt := "1"
		bz, err := VoteHash(salt, sdk.OneDec(), assets.MicroSDRDenom, sdk.ValAddress(addr))
		require.Nil(t, err)

		prevoteMsg := NewMsgPricePrevote(
			hex.EncodeToString(bz),
			assets.MicroSDRDenom,
			addr,
			sdk.ValAddress(addr),
		)

		res := h(input.ctx, prevoteMsg)
		require.True(t, res.IsOK())

		voteMsg := NewMsgPriceVote(
			sdk.OneDec(),
			salt,
			assets.MicroSDRDenom,
			addr,
			sdk.ValAddress(addr),
		)

		res = h(input.ctx.WithBlockHeight(1), voteMsg)
		require.True(t, res.IsOK())
	}

	params := input.oracleKeeper.GetParams(input.ctx)
	params.VotePeriod = 10 // set vote period to 10 for now, for convinience
	input.oracleKeeper.SetParams(input.ctx, params)
	require.Equal(t, 0, int(input.ctx.BlockHeight()))

	EndBlocker(input.ctx, input.oracleKeeper)
	require.Equal(t, 0, countClaimPool(input.ctx, input.oracleKeeper))

	input.ctx = input.ctx.WithBlockHeight(params.VotePeriod - 1)

	EndBlocker(input.ctx, input.oracleKeeper)
	require.Equal(t, len(addrs), countClaimPool(input.ctx, input.oracleKeeper))
}

func countClaimPool(ctx sdk.Context, oracleKeeper Keeper) (claimCount int) {
	oracleKeeper.iterateClaimPool(ctx, func(recipient sdk.AccAddress, weight sdk.Int) (stop bool) {
		claimCount++
		return false
	})

	return claimCount
}

func TestOracleRewardDistribution(t *testing.T) {
	input, h := setup(t)

	salt := "1"
	bz, _ := VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[0]))
	prevoteMsg := NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg := NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[0], sdk.ValAddress(addrs[0]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	salt = "2"
	bz, _ = VoteHash(salt, randomPrice, assets.MicroSDRDenom, sdk.ValAddress(addrs[1]))
	prevoteMsg = NewMsgPricePrevote(hex.EncodeToString(bz), assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	h(input.ctx.WithBlockHeight(0), prevoteMsg)

	voteMsg = NewMsgPriceVote(randomPrice, salt, assets.MicroSDRDenom, addrs[1], sdk.ValAddress(addrs[1]))
	h(input.ctx.WithBlockHeight(1), voteMsg)

	input.oracleKeeper.AddSwapFeePool(input.ctx.WithBlockHeight(1), sdk.NewCoins(sdk.NewCoin(assets.MicroSDRDenom, uLunaAmt.MulRaw(100))))

	EndBlocker(input.ctx.WithBlockHeight(1), input.oracleKeeper)
	EndBlocker(input.ctx.WithBlockHeight(2), input.oracleKeeper)

	rewards := input.distrKeeper.GetValidatorOutstandingRewards(input.ctx.WithBlockHeight(2), sdk.ValAddress(addrs[0]))
	require.Equal(t, uLunaAmt.MulRaw(50), rewards.AmountOf(assets.MicroSDRDenom).TruncateInt())
	rewards = input.distrKeeper.GetValidatorOutstandingRewards(input.ctx.WithBlockHeight(2), sdk.ValAddress(addrs[1]))
	require.Equal(t, uLunaAmt.MulRaw(50), rewards.AmountOf(assets.MicroSDRDenom).TruncateInt())
}

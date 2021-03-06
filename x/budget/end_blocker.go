package budget

import (
	"strconv"

	"github.com/terra-project/core/types"
	"github.com/terra-project/core/types/assets"
	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/budget/tags"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Tally returns votePower = yesVotes minus NoVotes for program, as well as the total votes.
// Power is denominated in validator bonded tokens (Luna stake size)
func tally(ctx sdk.Context, k Keeper, targetProgramID uint64) (votePower sdk.Int, totalPower sdk.Int) {
	votePower = sdk.ZeroInt()
	totalPower = k.valset.TotalBondedTokens(ctx)

	targetProgramIDPrefix := keyVote(targetProgramID, sdk.AccAddress{})
	k.IterateVotesWithPrefix(ctx, targetProgramIDPrefix, func(programID uint64, voter sdk.AccAddress, option bool) (stop bool) {
		valAddr := sdk.ValAddress(voter)

		if validator := k.valset.Validator(ctx, valAddr); validator != nil {
			bondSize := validator.GetBondedTokens()
			if option {
				votePower = votePower.Add(bondSize)
			} else {
				votePower = votePower.Sub(bondSize)
			}
		} else {
			k.DeleteVote(ctx, targetProgramID, voter)
		}

		return false
	})

	return
}

// clearsThreshold returns true if totalPower * threshold < votePower
func clearsThreshold(votePower, totalPower sdk.Int, threshold sdk.Dec) bool {
	return votePower.GTE(threshold.MulInt(totalPower).RoundInt())
}

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k Keeper) (resTags sdk.Tags) {
	params := k.GetParams(ctx)
	resTags = sdk.EmptyTags()

	k.CandQueueIterateExpired(ctx, ctx.BlockHeight(), func(programID uint64) (stop bool) {
		program, err := k.GetProgram(ctx, programID)
		if err != nil {
			return false
		}

		// Did not pass the tally, delete program
		votePower, totalPower := tally(ctx, k, programID)

		if !clearsThreshold(votePower, totalPower, params.ActiveThreshold) {
			k.DeleteVotesForProgram(ctx, programID)
			k.DeleteProgram(ctx, programID)
			resTags.AppendTag(tags.Action, tags.ActionProgramRejected)
		} else {
			resTags.AppendTag(tags.Action, tags.ActionProgramPassed)
		}

		resTags.AppendTags(
			sdk.NewTags(
				tags.ProgramID, strconv.FormatUint(programID, 10),
				tags.Weight, votePower.String(),
			),
		)

		k.CandQueueRemove(ctx, program.getVotingEndBlock(ctx, k), programID)
		return false
	})

	// Time to re-weight programs
	if util.IsPeriodLastBlock(ctx, params.VotePeriod) {
		claims := types.ClaimPool{}

		// iterate programs and weight them
		k.IteratePrograms(ctx, true, func(program Program) (stop bool) {
			votePower, totalPower := tally(ctx, k, program.ProgramID)

			// Need to check if the program should be legacied
			if !clearsThreshold(votePower, totalPower, params.LegacyThreshold) {
				// Delete all votes on target program
				k.DeleteVotesForProgram(ctx, program.ProgramID)
				k.DeleteProgram(ctx, program.ProgramID)
				resTags.AppendTag(tags.Action, tags.ActionProgramLegacied)
			} else {
				claims = append(claims, types.NewClaim(votePower, program.Executor))
				resTags.AppendTag(tags.Action, tags.ActionProgramGranted)
			}

			resTags.AppendTags(
				sdk.NewTags(
					tags.ProgramID, strconv.FormatUint(program.ProgramID, 10),
					tags.Weight, votePower.String(),
				),
			)

			return false
		})

		k.addClaimPool(ctx, claims)
	}

	// Time to distribute rewards to claims
	if util.IsPeriodLastBlock(ctx, util.BlocksPerEpoch) {
		epoch := util.GetEpoch(ctx)
		rewardWeight := k.tk.GetRewardWeight(ctx, epoch)
		seigniorage := k.mk.PeekEpochSeigniorage(ctx, epoch)
		rewardPool := sdk.OneDec().Sub(rewardWeight).MulInt(seigniorage)

		if rewardPool.GT(sdk.ZeroDec()) {
			rewardPoolCoin, err := k.mrk.GetSwapDecCoin(ctx, sdk.NewDecCoinFromDec(assets.MicroLunaDenom, rewardPool), assets.MicroSDRDenom)
			if err != nil {
				// No SDR swap rate exists
				rewardPoolCoin = sdk.NewDecCoinFromDec(assets.MicroLunaDenom, rewardPool)
			}

			weightSum := sdk.ZeroInt()
			k.iterateClaimPool(ctx, func(_ sdk.AccAddress, weight sdk.Int) (stop bool) {
				weightSum = weightSum.Add(weight)
				return false
			})

			k.iterateClaimPool(ctx, func(recipient sdk.AccAddress, weight sdk.Int) (stop bool) {
				rewardAmt := rewardPoolCoin.Amount.MulInt(weight).QuoInt(weightSum).TruncateInt()

				// never return err, but handle err for lint
				err := k.mk.Mint(ctx, recipient, sdk.NewCoin(rewardPoolCoin.Denom, rewardAmt))
				if err != nil {
					panic(err)
				}

				return false
			})
		}

		// Clear all claims
		k.clearClaimPool(ctx)
	}
	return
}

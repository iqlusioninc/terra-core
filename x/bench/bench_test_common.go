package bench

import (
	"github.com/terra-project/core/types/assets"
	"github.com/terra-project/core/x/budget"
	"github.com/terra-project/core/x/market"
	"github.com/terra-project/core/x/mint"
	"github.com/terra-project/core/x/oracle"
	"github.com/terra-project/core/x/pay"
	"github.com/terra-project/core/x/treasury"

	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

const numOfValidators = 100

var (
	pubKeys [numOfValidators]crypto.PubKey
	addrs   [numOfValidators]sdk.AccAddress

	valConsPubKeys [numOfValidators]crypto.PubKey
	valConsAddrs   [numOfValidators]sdk.ConsAddress

	uLunaAmt = sdk.NewInt(10000000000).MulRaw(assets.MicroUnit)
	uSDRAmt  = sdk.NewInt(10000000000).MulRaw(assets.MicroUnit)
)

type testInput struct {
	ctx            sdk.Context
	cdc            *codec.Codec
	bankKeeper     bank.Keeper
	budgetKeeper   budget.Keeper
	oracleKeeper   oracle.Keeper
	marketKeeper   market.Keeper
	mintKeeper     mint.Keeper
	treasuryKeeper treasury.Keeper
}

func newTestCodec() *codec.Codec {
	cdc := codec.New()

	pay.RegisterCodec(cdc)
	treasury.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)

	return cdc
}

// createTestInput common test code for bench test
func createTestInput() testInput {
	keyAcc := sdk.NewKVStoreKey(auth.StoreKey)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tKeyParams := sdk.NewTransientStoreKey(params.TStoreKey)
	keyMarket := sdk.NewKVStoreKey(market.StoreKey)
	keyMint := sdk.NewKVStoreKey(mint.StoreKey)
	keyFee := sdk.NewKVStoreKey(auth.FeeStoreKey)
	keyBudget := sdk.NewKVStoreKey(budget.StoreKey)
	keyOracle := sdk.NewKVStoreKey(oracle.StoreKey)
	keyTreasury := sdk.NewKVStoreKey(treasury.StoreKey)
	keyStaking := sdk.NewKVStoreKey(staking.StoreKey)
	tKeyStaking := sdk.NewTransientStoreKey(staking.TStoreKey)
	keyDistr := sdk.NewKVStoreKey(distr.StoreKey)
	tKeyDistr := sdk.NewTransientStoreKey(distr.TStoreKey)

	cdc := newTestCodec()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ctx := sdk.NewContext(ms, abci.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())

	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyMarket, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyMint, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBudget, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyOracle, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyTreasury, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyStaking, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyDistr, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyDistr, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyFee, sdk.StoreTypeIAVL, db)

	if err := ms.LoadLatestVersion(); err != nil {
		panic(err)
	}

	paramsKeeper := params.NewKeeper(cdc, keyParams, tKeyParams)
	accKeeper := auth.NewAccountKeeper(
		cdc,
		keyAcc,
		paramsKeeper.Subspace(auth.DefaultParamspace),
		auth.ProtoBaseAccount,
	)

	bankKeeper := bank.NewBaseKeeper(
		accKeeper,
		paramsKeeper.Subspace(bank.DefaultParamspace),
		bank.DefaultCodespace,
	)

	stakingKeeper := staking.NewKeeper(
		cdc,
		keyStaking, tKeyStaking,
		bankKeeper, paramsKeeper.Subspace(staking.DefaultParamspace),
		staking.DefaultCodespace,
	)

	stakingKeeper.SetPool(ctx, staking.InitialPool())
	stakingParams := staking.DefaultParams()
	stakingParams.BondDenom = assets.MicroLunaDenom
	stakingKeeper.SetParams(ctx, stakingParams)

	feeKeeper := auth.NewFeeCollectionKeeper(
		cdc, keyFee,
	)

	distrKeeper := distr.NewKeeper(
		cdc, keyDistr, paramsKeeper.Subspace(distr.DefaultParamspace),
		bankKeeper, stakingKeeper, feeKeeper, distr.DefaultCodespace,
	)

	mintKeeper := mint.NewKeeper(
		cdc,
		keyMint,
		stakingKeeper,
		bankKeeper,
		accKeeper,
	)

	sh := staking.NewHandler(stakingKeeper)
	for i := 0; i < 100; i++ {
		pubKeys[i] = secp256k1.GenPrivKey().PubKey()
		addrs[i] = sdk.AccAddress(pubKeys[i].Address())

		valConsPubKeys[i] = ed25519.GenPrivKey().PubKey()
		valConsAddrs[i] = sdk.ConsAddress(valConsPubKeys[i].Address())

		err2 := mintKeeper.Mint(ctx, addrs[i], sdk.NewCoin(assets.MicroLunaDenom, uLunaAmt.MulRaw(3)))
		if err2 != nil {
			panic(err2)
		}

		// Add validators
		commission := staking.NewCommissionMsg(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
		msg := staking.NewMsgCreateValidator(sdk.ValAddress(addrs[i]), valConsPubKeys[i],
			sdk.NewCoin(assets.MicroLunaDenom, uLunaAmt), staking.Description{}, commission, sdk.OneInt())
		res := sh(ctx, msg)
		if !res.IsOK() {
			panic(res.Log)
		}

		distrKeeper.Hooks().AfterValidatorCreated(ctx, sdk.ValAddress(addrs[i]))
		staking.EndBlocker(ctx, stakingKeeper)
	}

	oracleKeeper := oracle.NewKeeper(
		cdc,
		keyOracle,
		mintKeeper,
		distrKeeper,
		feeKeeper,
		stakingKeeper.GetValidatorSet(),
		paramsKeeper.Subspace(oracle.DefaultParamspace),
	)

	marketKeeper := market.NewKeeper(
		cdc,
		keyMarket,
		oracleKeeper,
		mintKeeper,
		paramsKeeper.Subspace(market.DefaultParamspace))

	treasuryKeeper := treasury.NewKeeper(
		cdc,
		keyTreasury,
		stakingKeeper.GetValidatorSet(),
		mintKeeper,
		marketKeeper,
		paramsKeeper.Subspace(treasury.DefaultParamspace),
	)

	budgetKeeper := budget.NewKeeper(
		cdc, keyBudget, marketKeeper, mintKeeper, treasuryKeeper, stakingKeeper.GetValidatorSet(),
		paramsKeeper.Subspace(budget.DefaultParamspace),
	)

	budget.InitGenesis(ctx, budgetKeeper, budget.DefaultGenesisState())
	oracle.InitGenesis(ctx, oracleKeeper, oracle.DefaultGenesisState())
	treasury.InitGenesis(ctx, treasuryKeeper, treasury.DefaultGenesisState())

	return testInput{ctx, cdc, bankKeeper, budgetKeeper, oracleKeeper, marketKeeper, mintKeeper, treasuryKeeper}
}

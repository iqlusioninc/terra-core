package client

import (
	"github.com/terra-project/core/x/oracle/client/cli"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	amino "github.com/tendermint/go-amino"
)

// ModuleClient exports all client functionality from this module
type ModuleClient struct {
	storeKey string
	cdc      *amino.Codec
}

func NewModuleClient(storeKey string, cdc *amino.Codec) ModuleClient {
	return ModuleClient{storeKey, cdc}
}

// GetQueryCmd returns the cli query commands for this module
func (mc ModuleClient) GetQueryCmd() *cobra.Command {
	oracleQueryCmd := &cobra.Command{
		Use:   "oracle",
		Short: "Querying commands for the oracle module",
	}
	oracleQueryCmd.AddCommand(client.GetCommands(
		cli.GetCmdQueryPrice(mc.storeKey, mc.cdc),
		cli.GetCmdQueryVotes(mc.storeKey, mc.cdc),
		cli.GetCmdQueryPrevotes(mc.storeKey, mc.cdc),
		cli.GetCmdQueryActive(mc.storeKey, mc.cdc),
		cli.GetCmdQueryParams(mc.storeKey, mc.cdc),
		cli.GetCmdQueryFeederDelegation(mc.storeKey, mc.cdc),
	)...)

	return oracleQueryCmd

}

// GetTxCmd returns the transaction commands for this module
func (mc ModuleClient) GetTxCmd() *cobra.Command {
	oracleTxCmd := &cobra.Command{
		Use:   "oracle",
		Short: "Oracle transaction subcommands",
	}

	oracleTxCmd.AddCommand(client.PostCommands(
		cli.GetCmdPricePrevote(mc.cdc),
		cli.GetCmdPriceVote(mc.cdc),
		cli.GetCmdDelegateFeederPermission(mc.cdc),
	)...)

	return oracleTxCmd
}

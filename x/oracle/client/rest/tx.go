package rest

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/terra-project/core/x/oracle"

	"github.com/cosmos/cosmos-sdk/client/context"
	clientrest "github.com/cosmos/cosmos-sdk/client/rest"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/gorilla/mux"
)

func resgisterTxRoute(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	r.HandleFunc(fmt.Sprintf("/oracle/denoms/{%s}/prevotes", RestDenom), submitPrevoteHandlerFunction(cdc, cliCtx)).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/oracle/denoms/{%s}/votes", RestDenom), submitVoteHandlerFunction(cdc, cliCtx)).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/oracle/voters/{%s}/feeder", RestVoter), submitDelegateHandlerFunction(cdc, cliCtx)).Methods("POST")
}

// PrevoteReq ...
type PrevoteReq struct {
	BaseReq rest.BaseReq `json:"base_req"`

	Hash  string  `json:"hash"`
	Price sdk.Dec `json:"price"`
	Salt  string  `json:"salt"`

	Validator string `json:"validator"`
}

func submitPrevoteHandlerFunction(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		denom := vars[RestDenom]

		var req PrevoteReq
		if !rest.ReadRESTReq(w, r, cdc, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()

		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddress, err := sdk.AccAddressFromBech32(req.BaseReq.From)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Default validator is self address
		var valAddress sdk.ValAddress
		if len(req.Validator) == 0 {
			valAddress = sdk.ValAddress(fromAddress)
		} else {
			valAddress, err = sdk.ValAddressFromBech32(req.Validator)
			if err != nil {
				rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		// If hash is not given, then retrieve hash from price and salt
		if len(req.Hash) == 0 && (!req.Price.Equal(sdk.ZeroDec()) && len(req.Salt) > 0) {
			hashBytes, err := oracle.VoteHash(req.Salt, req.Price, denom, valAddress)
			if err != nil {
				rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}

			req.Hash = hex.EncodeToString(hashBytes)
		}

		// create the message
		msg := oracle.NewMsgPricePrevote(req.Hash, denom, fromAddress, valAddress)
		err = msg.ValidateBasic()
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		clientrest.WriteGenerateStdTxResponse(w, cdc, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}

//VoteReq ...
type VoteReq struct {
	BaseReq rest.BaseReq `json:"base_req"`

	Price sdk.Dec `json:"price"`
	Salt  string  `json:"salt"`

	Validator string `json:"validator"`
}

func submitVoteHandlerFunction(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		denom := vars[RestDenom]

		var req VoteReq
		if !rest.ReadRESTReq(w, r, cdc, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()

		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddress, err := sdk.AccAddressFromBech32(req.BaseReq.From)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Default validator is self address
		var valAddress sdk.ValAddress
		if len(req.Validator) == 0 {
			valAddress = sdk.ValAddress(fromAddress)
		} else {
			valAddress, err = sdk.ValAddressFromBech32(req.Validator)
			if err != nil {
				rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		// create the message
		msg := oracle.NewMsgPriceVote(req.Price, req.Salt, denom, fromAddress, valAddress)
		err = msg.ValidateBasic()
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		clientrest.WriteGenerateStdTxResponse(w, cdc, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}

// DelegateReq is request body to set feeder of validator
type DelegateReq struct {
	BaseReq rest.BaseReq `json:"base_req"`
	Feeder  string       `json:"feeder"`
}

func submitDelegateHandlerFunction(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		voter := vars[RestVoter]

		// Get voter validator address
		valAddress, err := sdk.ValAddressFromBech32(voter)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		var req DelegateReq
		if !rest.ReadRESTReq(w, r, cdc, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()

		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddress, err := sdk.AccAddressFromBech32(req.BaseReq.From)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Bytes comparison, so do not require type conversion
		if !valAddress.Equals(fromAddress) {
			err := fmt.Errorf("[%v] can not change [%v] delegation", fromAddress, valAddress)
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		feeder, err := sdk.AccAddressFromBech32(req.Feeder)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// create the message
		msg := oracle.NewMsgDelegateFeederPermission(valAddress, feeder)
		err = msg.ValidateBasic()
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		clientrest.WriteGenerateStdTxResponse(w, cdc, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}

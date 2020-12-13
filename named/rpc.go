package main

import (
	"encoding/json"
	"net/http"

	"github.com/fiatjaf/namechain/common"
)

func listenRPC() {
	log.Info().Str("addr", config.RPCAddr).Msg("listening")
	http.HandleFunc("/rpc", handleRPC)
	err := http.ListenAndServe(config.RPCAddr, nil)
	if err != nil {
		log.Error().Err(err).Msg("error serving http")
	}
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req common.RPCRequest
	var resp common.RPCResponse
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		resp.Error.Code = -32700
		resp.Error.Message = "error decoding request JSON"
		json.NewEncoder(w).Encode(resp)
	}
	resp.ID = req.ID

	switch req.Method {
	case "getinfo":
		resp.Result = struct {
		}{}
	case "publishblock":
		resp.Result = struct {
		}{}
	default:
		resp.Error.Code = -32601
		resp.Error.Message = "method not found: '" + req.Method + "'"
	}

	json.NewEncoder(w).Encode(resp)
}

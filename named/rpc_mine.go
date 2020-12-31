package main

import (
	"encoding/hex"
	"errors"
)

func RPCMine(params map[string]interface{}) (result interface{}, err error) {
	nextBMMTransaction := 1 // TODO
	rawBlockParam, ok := params["block"]
	if !ok {
		return nil, errors.New("Missing 'block' param.")
	}

	rawBlock, err := hex.DecodeString(rawBlockParam.(string))
	if err != nil {
		return nil, errors.New("'block' param is invalid hex.")
	}

	return map[string]interface{}{
		"": "",
	}, nil
}

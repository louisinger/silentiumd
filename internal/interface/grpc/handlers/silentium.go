package handlers

import (
	"context"
	"encoding/hex"

	silentiumv1 "github.com/louisinger/silentiumd/api/protobuf/gen/silentium/v1"
	"github.com/louisinger/silentiumd/internal/application"
)

type handler struct {
	svc application.SilentiumService
}

func NewHandler(service application.SilentiumService) silentiumv1.SilentiumServiceServer {
	return &handler{service}
}

func (h *handler) GetBlockFilter(ctx context.Context, req *silentiumv1.GetBlockFilterRequest) (*silentiumv1.GetBlockFilterResponse, error) {
	filter, blockhash, err := h.svc.GetBlockFilter(req.GetBlockId())
	if err != nil {
		return nil, err
	}

	return &silentiumv1.GetBlockFilterResponse{
		Blockhash: blockhash,
		Filter:    filter,
	}, nil
}

func (h *handler) GetBlockScalars(ctx context.Context, req *silentiumv1.GetBlockScalarsRequest) (*silentiumv1.GetBlockScalarsResponse, error) {
	scalars, err := h.svc.GetScalarsByHeight(uint32(req.GetBlockId()))
	if err != nil {
		return nil, err
	}

	res := &silentiumv1.GetBlockScalarsResponse{
		Scalars: make([]string, len(scalars)),
	}

	for i, scalar := range scalars {
		res.Scalars[i] = hex.EncodeToString(scalar.Scalar)
	}

	return res, nil
}

func (h *handler) GetChainTipHeight(_ context.Context, req *silentiumv1.GetChainTipHeightRequest) (*silentiumv1.GetChainTipHeightResponse, error) {
	tip, err := h.svc.GetChainTip()
	if err != nil {
		return nil, err
	}

	return &silentiumv1.GetChainTipHeightResponse{
		Height: tip,
	}, nil
}

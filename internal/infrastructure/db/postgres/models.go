package postgres

import "github.com/uptrace/bun"

type ScalarModel struct {
	bun.BaseModel `bun:"table:scalars,alias:s"`

	TxHash         string                `bun:",pk,unique"`
	Scalar         string                `bun:",notnull"`
	BlockHeight    int32                 `bun:",notnull"`
	TaprootOutputs []*TaprootOutputModel `bun:"rel:has-many,join:tx_hash=tx_hash"`
}

type TaprootOutputModel struct {
	bun.BaseModel `bun:"table:taproot_outputs,alias:o"`

	ID     int64  `bun:",pk,autoincrement"`
	TxHash string `bun:",notnull"`
	Index  uint32 `bun:",notnull"`
}

package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/btcsuite/btcd/wire"
	"github.com/louisinger/silentiumd/internal/domain"
	"github.com/louisinger/silentiumd/internal/ports"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type repository struct {
	db *bun.DB
}

type PostreSQLConfig struct {
	Dsn string
}

func New(opts PostreSQLConfig) (ports.ScalarRepository, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(opts.Dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	ctx := context.Background()

	if _, err := db.NewCreateTable().Model((*TaprootOutputModel)(nil)).IfNotExists().Exec(ctx); err != nil {
		return nil, err
	}

	if _, err := db.NewCreateTable().Model((*ScalarModel)(nil)).IfNotExists().Exec(ctx); err != nil {
		return nil, err
	}

	return &repository{db}, nil
}

// GetScalars returns all the scalars for a given block height where at least 1 taproot output is present.
func (r *repository) GetScalars(height int32) ([]string, error) {
	dest := make([]struct{ Scalar string }, 0)

	if err := r.db.NewSelect().Model((*ScalarModel)(nil)).
		Column("scalar").
		Where("block_height = ?", height).
		Join("JOIN taproot_outputs AS o").
		JoinOn("o.tx_hash = s.tx_hash").
		DistinctOn("scalar").
		Scan(context.Background(), &dest); err != nil {
		return nil, err
	}

	scalars := make([]string, 0, len(dest))
	for _, d := range dest {
		scalars = append(scalars, d.Scalar)
	}

	return scalars, nil
}

func (r *repository) MarkSpent(outpoints []wire.OutPoint) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	for _, outpoint := range outpoints {
		if _, err := tx.NewDelete().Model((*TaprootOutputModel)(nil)).
			Where("tx_hash = ?", outpoint.Hash.String()).
			Where("index = ?", outpoint.Index).
			Exec(context.Background()); err != nil {
			tx.Rollback()
		}
	}

	return tx.Commit()
}

// GetLatestBlockHeight returns the maximum block height value in the scalars table.
func (r *repository) GetLatestBlockHeight() (int32, error) {
	var maxBlockHeight int32
	err := r.db.NewSelect().
		Model((*ScalarModel)(nil)).
		ColumnExpr("MAX(block_height)").
		Scan(context.Background(), &maxBlockHeight)
	return maxBlockHeight, err
}

func (r *repository) Write(scalars []*domain.SilentScalar, blockHeight int32) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	for _, scalar := range scalars {
		scalarModel := &ScalarModel{
			TxHash:      scalar.TxHash.String(),
			Scalar:      hex.EncodeToString(scalar.Scalar),
			BlockHeight: blockHeight,
		}

		if _, err := tx.NewInsert().Model(scalarModel).Exec(context.Background()); err != nil {
			tx.Rollback()
			return err
		}

		for _, out := range scalar.TaprootOutputs {
			taprootOutputModel := &TaprootOutputModel{
				TxHash: scalar.TxHash.String(),
				Index:  out.Index,
			}

			if _, err := tx.NewInsert().Model(taprootOutputModel).Exec(context.Background()); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

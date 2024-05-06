package domain

import "errors"

var (
	ErrNoTaprootOutputs              = errors.New("no taproot outputs")
	ErrNonStandardScript             = errors.New("non standard script")
	ErrInvalidTaprootWitness         = errors.New("invalid taproot witness")
	ErrInternalTaprootKeyIsBasePoint = errors.New("internal taproot key is unspendable")
	ErrUnableToComputeScalar         = errors.New("unable to compute scalar")
)

package charger

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateKSE(base *KSE, identifier func() (string, error)) api.Charger {
	switch {
	case identifier == nil:
		return base

	case identifier != nil:
		return &struct {
			*KSE
			api.Identifier
		}{
			KSE: base,
			Identifier: &decorateKSEIdentifierImpl{
				identifier: identifier,
			},
		}
	}

	return nil
}

type decorateKSEIdentifierImpl struct {
	identifier func() (string, error)
}

func (impl *decorateKSEIdentifierImpl) Identify() (string, error) {
	return impl.identifier()
}

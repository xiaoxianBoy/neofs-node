package reputation

import (
	"errors"
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/core/mempoolevent"
	reputationWrapper "github.com/nspcc-dev/neofs-node/pkg/morph/client/reputation/wrapper"
	"github.com/nspcc-dev/neofs-node/pkg/morph/event"
	reputationEvent "github.com/nspcc-dev/neofs-node/pkg/morph/event/reputation"
	"github.com/nspcc-dev/neofs-node/pkg/services/reputation/common"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

type (
	// EpochState is a callback interface for inner ring global state.
	EpochState interface {
		EpochCounter() uint64
	}

	// AlphabetState is a callback interface for inner ring global state.
	AlphabetState interface {
		IsAlphabet() bool
	}

	// Processor of events produced by reputation contract.
	Processor struct {
		log  *zap.Logger
		pool *ants.Pool

		epochState    EpochState
		alphabetState AlphabetState

		reputationWrp *reputationWrapper.ClientWrapper

		mngBuilder common.ManagerBuilder

		notaryDisabled bool
	}

	// Params of the processor constructor.
	Params struct {
		Log               *zap.Logger
		PoolSize          int
		EpochState        EpochState
		AlphabetState     AlphabetState
		ReputationWrapper *reputationWrapper.ClientWrapper
		ManagerBuilder    common.ManagerBuilder
		NotaryDisabled    bool
	}
)

const (
	putReputationNotification = "reputationPut"
)

// New creates reputation contract processor instance.
func New(p *Params) (*Processor, error) {
	switch {
	case p.Log == nil:
		return nil, errors.New("ir/reputation: logger is not set")
	case p.EpochState == nil:
		return nil, errors.New("ir/reputation: global state is not set")
	case p.AlphabetState == nil:
		return nil, errors.New("ir/reputation: global state is not set")
	case p.ReputationWrapper == nil:
		return nil, errors.New("ir/reputation: reputation contract wrapper is not set")
	case p.ManagerBuilder == nil:
		return nil, errors.New("ir/reputation: manager builder is not set")
	}

	p.Log.Debug("reputation worker pool", zap.Int("size", p.PoolSize))

	pool, err := ants.NewPool(p.PoolSize, ants.WithNonblocking(true))
	if err != nil {
		return nil, fmt.Errorf("ir/reputation: can't create worker pool: %w", err)
	}

	return &Processor{
		log:            p.Log,
		pool:           pool,
		epochState:     p.EpochState,
		alphabetState:  p.AlphabetState,
		reputationWrp:  p.ReputationWrapper,
		mngBuilder:     p.ManagerBuilder,
		notaryDisabled: p.NotaryDisabled,
	}, nil
}

// ListenerNotificationParsers for the 'event.Listener' event producer.
func (rp *Processor) ListenerNotificationParsers() []event.NotificationParserInfo {
	if !rp.notaryDisabled {
		return nil
	}

	var parsers []event.NotificationParserInfo

	// put reputation event
	put := event.NotificationParserInfo{}
	put.SetType(putReputationNotification)
	put.SetScriptHash(rp.reputationWrp.ContractAddress())
	put.SetParser(reputationEvent.ParsePut)
	parsers = append(parsers, put)

	return parsers
}

// ListenerNotificationHandlers for the 'event.Listener' event producer.
func (rp *Processor) ListenerNotificationHandlers() []event.NotificationHandlerInfo {
	if !rp.notaryDisabled {
		return nil
	}

	var handlers []event.NotificationHandlerInfo

	// put reputation handler
	put := event.NotificationHandlerInfo{}
	put.SetType(putReputationNotification)
	put.SetScriptHash(rp.reputationWrp.ContractAddress())
	put.SetHandler(rp.handlePutReputation)
	handlers = append(handlers, put)

	return handlers
}

// ListenerNotaryParsers for the 'event.Listener' notary event producer.
func (rp *Processor) ListenerNotaryParsers() []event.NotaryParserInfo {
	var p event.NotaryParserInfo

	p.SetMempoolType(mempoolevent.TransactionAdded)
	p.SetRequestType(reputationEvent.PutNotaryEvent)
	p.SetScriptHash(rp.reputationWrp.ContractAddress())
	p.SetParser(reputationEvent.ParsePutNotary)

	return []event.NotaryParserInfo{p}
}

// ListenerNotaryHandlers for the 'event.Listener' notary event producer.
func (rp *Processor) ListenerNotaryHandlers() []event.NotaryHandlerInfo {
	var h event.NotaryHandlerInfo

	h.SetMempoolType(mempoolevent.TransactionAdded)
	h.SetRequestType(reputationEvent.PutNotaryEvent)
	h.SetScriptHash(rp.reputationWrp.ContractAddress())
	h.SetHandler(rp.handlePutReputation)

	return []event.NotaryHandlerInfo{h}
}

// TimersHandlers for the 'Timers' event producer.
func (rp *Processor) TimersHandlers() []event.NotificationHandlerInfo {
	return nil
}

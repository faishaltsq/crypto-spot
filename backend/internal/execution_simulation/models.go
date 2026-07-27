package execution_simulation

import "time"

type Config struct {
	Enabled              bool
	Notionals            []float64
	FeeBPS               float64
	IncludeEntryFee      bool
	IncludeExitFee       bool
	IncludeEntrySlippage bool
	IncludeExitSlippage  bool
	AllowPartialFill     bool
}

type Status string

const (
	StatusComplete    Status = "COMPLETE"
	StatusPartialFill Status = "PARTIAL_FILL"
	StatusIncomplete  Status = "INCOMPLETE"
)

// Simulation stores USDT amounts, BPS slippage, and decimal returns.
type Simulation struct {
	ID                       string    `json:"id"`
	SignalID                 string    `json:"signalId"`
	Notional                 float64   `json:"notional"`
	ReferencePrice           *float64  `json:"referencePrice,omitempty"`
	EstimatedEntryPrice      *float64  `json:"estimatedEntryPrice,omitempty"`
	EstimatedExitPrice       *float64  `json:"estimatedExitPrice,omitempty"`
	EntryFee                 *float64  `json:"entryFee,omitempty"`
	ExitFee                  *float64  `json:"exitFee,omitempty"`
	EntrySlippage            *float64  `json:"entrySlippage,omitempty"`
	ExitSlippage             *float64  `json:"exitSlippage,omitempty"`
	EntrySlippageBPS         *float64  `json:"entrySlippageBps,omitempty"`
	ExitSlippageBPS          *float64  `json:"exitSlippageBps,omitempty"`
	GrossReturn              *float64  `json:"grossReturn,omitempty"`
	NetReturn                *float64  `json:"netReturn,omitempty"`
	MaximumSupportedNotional *float64  `json:"maximumSupportedNotional,omitempty"`
	DepthCoverage            *float64  `json:"depthCoverage,omitempty"`
	LiquidityConfidence      *float64  `json:"liquidityConfidence,omitempty"`
	FilledNotional           *float64  `json:"filledNotional,omitempty"`
	UnfilledNotional         *float64  `json:"unfilledNotional,omitempty"`
	PartialFill              bool      `json:"partialFill"`
	Status                   Status    `json:"simulationStatus"`
	SimulatedAt              time.Time `json:"simulatedAt"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
	BaseFilled               float64   `json:"-"`
}

func ptr(value float64) *float64 { return &value }

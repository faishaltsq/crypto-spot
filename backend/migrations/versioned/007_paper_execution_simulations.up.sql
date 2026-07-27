CREATE TABLE paper_execution_simulations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id UUID NOT NULL REFERENCES signals(id) ON DELETE CASCADE,
    notional DOUBLE PRECISION NOT NULL CHECK (notional > 0),
    reference_price DOUBLE PRECISION,
    estimated_entry_price DOUBLE PRECISION,
    estimated_exit_price DOUBLE PRECISION,
    entry_fee DOUBLE PRECISION,
    exit_fee DOUBLE PRECISION,
    entry_slippage DOUBLE PRECISION,
    exit_slippage DOUBLE PRECISION,
    entry_slippage_bps DOUBLE PRECISION,
    exit_slippage_bps DOUBLE PRECISION,
    gross_return DOUBLE PRECISION,
    net_return DOUBLE PRECISION,
    maximum_supported_notional DOUBLE PRECISION,
    depth_coverage DOUBLE PRECISION,
    liquidity_confidence DOUBLE PRECISION,
    filled_notional DOUBLE PRECISION,
    unfilled_notional DOUBLE PRECISION,
    partial_fill BOOLEAN NOT NULL DEFAULT FALSE,
    simulation_status VARCHAR(20) NOT NULL,
    simulated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (signal_id, notional)
);
CREATE INDEX idx_paper_execution_signal_id ON paper_execution_simulations (signal_id);
CREATE INDEX idx_paper_execution_notional ON paper_execution_simulations (notional);
CREATE INDEX idx_paper_execution_status ON paper_execution_simulations (simulation_status);
CREATE INDEX idx_paper_execution_simulated_at ON paper_execution_simulations (simulated_at DESC);

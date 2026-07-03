package risk

import "fmt"

type Config struct {
	RiskPerTrade       float64
	DailyLossLimit     float64
	MaxConsecutiveLoss int
}

type Manager struct {
	cfg             Config
	dailyLoss       float64
	consecutiveLoss int
	balance         float64
}

func NewManager(cfg Config, balance float64) *Manager {
	return &Manager{
		cfg:     cfg,
		balance: balance,
	}
}

func (m *Manager) PositionSize(entry float64, stopLoss float64) (float64, error) {

	if m.dailyLoss >= m.cfg.DailyLossLimit {
		return 0, fmt.Errorf("daily loss limit reached: %.2f%%", m.dailyLoss)
	}

	if m.consecutiveLoss >= m.cfg.MaxConsecutiveLoss {
		return 0, fmt.Errorf("max consecutive losses reached: %d", m.consecutiveLoss)
	}

	riskAmount := m.balance * (m.cfg.RiskPerTrade / 100)

	slDistance := entry - stopLoss
	if slDistance <= 0 {
		slDistance = stopLoss - entry
	}

	size := riskAmount / slDistance

	return size, nil
}

func (m *Manager) RecordLoss(amount float64) {
	m.dailyLoss += (amount / m.balance) * 100
	m.consecutiveLoss++
}

func (m *Manager) RecordWin() {
	m.consecutiveLoss = 0
}

func (m *Manager) ResetDaily() {
	m.dailyLoss = 0
}

func (m *Manager) CanTrade() (bool, string) {
	if m.dailyLoss >= m.cfg.DailyLossLimit {
		return false, fmt.Sprintf("Daily loss limit reached: %.2f%%", m.dailyLoss)
	}
	if m.consecutiveLoss >= m.cfg.MaxConsecutiveLoss {
		return false, fmt.Sprintf("Max consecutive losses: %d", m.consecutiveLoss)
	}
	return true, "OK"
}

func (m *Manager) AdjustBalance(delta float64) {
	m.balance += delta
}

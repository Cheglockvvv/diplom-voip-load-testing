package sip

type CallState string

const (
	StateIdle       CallState = "IDLE"
	StateCalling    CallState = "CALLING"
	StateProceeding CallState = "PROCEEDING"
	StateConfirmed  CallState = "CONFIRMED"
	StateTerminated CallState = "TERMINATED"
)

type FSM struct {
	State CallState
}

func NewFSM() *FSM {
	return &FSM{State: StateIdle}
}

func (f *FSM) StartCall() {
	f.State = StateCalling
}

func (f *FSM) OnProvisional() {
	if f.State == StateCalling {
		f.State = StateProceeding
	}
}

func (f *FSM) OnAccepted() {
	if f.State == StateCalling || f.State == StateProceeding {
		f.State = StateConfirmed
	}
}

func (f *FSM) OnTerminated() {
	f.State = StateTerminated
}

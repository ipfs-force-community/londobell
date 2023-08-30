package common

type SegmentType int

const (
	BlockStates       SegmentType = iota // block message
	BlockMethodStates                    // block message by methodname
	ActorStates                          // actor message
	ActorMethodStates                    // actor message by methodname
	ActorTransferStates
	ActorEventStates
	MinedStates
	LargeAmountTransferStates
)

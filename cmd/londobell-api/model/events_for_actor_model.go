package model

type EventForActor struct {
	ActorID   string
	Epoch     int64
	Cid       string
	SignedCid string
	Topics    []string
	Data      string
	LogIndex  uint64
	Removed   bool
}
type EventsForActorRes struct {
	TotalCount     int64           `json:"totalCount"`
	EventsForActor []EventForActor `json:"eventsForActor"`
}

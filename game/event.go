package game

import (
	"fmt"
	"time"

	"github.com/pkartner/event"
)

const (
	NextTurnEventType = "next_turn"
	SetPolicyEventType = "set_policy"
	SetBranchEventType = "set_branch"
	SetScreenEventType = "set_screen"
	SetSelectedValueType = "set_selected_value"
)

func SetBasicEventValues(e *event.BaseEvent) {
	lastID := Current.Dispatcher.Store.LastEvent.ID().IDPart()
	t := uint64(time.Now().Unix())
	e.EventID = event.GenerateTimeID(t, lastID+1)
	e.EventTime = t
}

type NextTurnEvent struct {
	event.BaseTimelineEvent
}

// Type TODO
func (e *NextTurnEvent) Type() event.EventType {
    return NextTurnEventType
}

func NextTurn() *NextTurnEvent{
	store := Current.GetCurrentBranchStore()
	gameStore := Current.GetGameStore()
	lastID := Current.Dispatcher.Store.LastEvent.ID().IDPart()
	e := NextTurnEvent{}
	e.EventID = event.GenerateTimeID(store.Turn, lastID+1)
	fmt.Println(fmt.Sprintf("Next event turn time is %d", store.Turn))
	e.EventTime = store.Turn+1
	
	fmt.Println(fmt.Sprintf("Current event turn time is %d", e.Time()))
	e.BranchID = gameStore.CurrentBranch
	return &e
}

type SetPolicyEvent struct {
	event.BaseTimelineEvent
	Policy string
	State bool
}

func (e *SetPolicyEvent) Type() event.EventType {
	return SetPolicyEventType
}

func SetPolicy(policy string) *SetPolicyEvent {
	store := Current.GetCurrentBranchStore()
	gameStore := Current.GetGameStore()
	state := false
	_, ok := store.ActivePolicies[policy]
	if !ok {
		state = true
	}
	lastID := Current.Dispatcher.Store.LastEvent.ID().IDPart()
	e := SetPolicyEvent{
		Policy: policy,
		State: state,
	}
	e.EventID = event.GenerateTimeID(store.Turn, lastID+1)
	e.EventTime = store.Turn
	e.BranchID = gameStore.CurrentBranch
	return &e
}

// Game Events

type SetBranchEvent struct {
	event.BaseEvent
	BranchID event.ID
}

func (e *SetBranchEvent) Type() event.EventType {
	return SetBranchEventType
}

func SetBranch(branchID event.ID) *SetBranchEvent{
	e := SetBranchEvent{
		BranchID: branchID,
	}
	SetBasicEventValues(&e.BaseEvent)
	return &e
}

type SetScreenEvent struct {
	event.BaseEvent
	Screen string
}

func (e *SetScreenEvent) Type() event.EventType {
	return SetScreenEventType
}

func SetScreen(screen string) *SetScreenEvent {
	e := SetScreenEvent{
		Screen: screen,
	}
	SetBasicEventValues(&e.BaseEvent)
	return &e
}

type SetSelectedValueEvent struct {
	event.BaseEvent
	Value string
}

func (e *SetSelectedValueEvent) Type() event.EventType {
	return SetSelectedValueType
}

func SetSelectedValue(value string) *SetSelectedValueEvent {
		e := SetSelectedValueEvent{
		Value: value,
	}
	SetBasicEventValues(&e.BaseEvent)
	return &e
}
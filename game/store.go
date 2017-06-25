package game

import (
	"fmt"

	"github.com/pkartner/event"
)

const GameWon uint8 = 1
const GameLost uint8 = 2

type GameStore struct {
	CurrentBranch event.ID
	LastEventID uint64
	CurrentScreen string
	SelectedValue string
	Rewind bool
}

type BranchStore struct {
	BranchID event.ID
	Values ValueMap
	Weights WeightMap
	GameOver uint8
	ActivePolicies map[string]struct{}
	Turn uint64
}

// GetBranchID TODO
func(s *BranchStore) GetBranchID() event.ID{
    return s.BranchID
}

// SetBranchID TODO
func(s *BranchStore) SetBranchID(id event.ID) {
    s.BranchID = id
}

func NewBranchStoreFunc(values ValueMap) func() event.BranchStore {
	return func () event.BranchStore {
		store := &BranchStore{}
		store.ActivePolicies = map[string]struct{}{}
		store.Values = values
		return store
	}
}

func GetBranchStore(s *event.Store) *BranchStore {
    store, ok := s.Attributes.(*BranchStore)
    if !ok {
        panic(fmt.Errorf("Store not of type BranchStore"))
    }
    return store
}

func GetGameStore(s *event.Store) *GameStore {
	timeStore, ok := s.Attributes.(*event.TimelineStore)
	if !ok {
		panic(fmt.Errorf("Store not of type TimelineStore"))
	}
	gameStore, ok := timeStore.Attributes.(*GameStore)
	return gameStore
}
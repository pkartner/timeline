package game

import (
	"fmt"

	"github.com/pkartner/event"
)

func EventCastFailError(expected, actual string) error {
	return fmt.Errorf("Event not right type expected: %s, actual: %s", expected, actual)
}

func (g *Instance) SetPolicyHandler(e event.Event, s *event.Store) {
	event, ok := e.(*SetPolicyEvent)
	if !ok {
		panic(EventCastFailError(SetPolicyEventType, e.Type().String()))
	}
	store := GetBranchStore(s)
	fmt.Println(fmt.Sprintf("Setting policy %s to %t", event.Policy, event.State))
	if !event.State {
		delete(store.ActivePolicies, event.Policy)
		return
	}

	for _, v := range Current.GameData.Policies.MutualExclusive {
		present := false
		for _, v2 :=  range v {
			if v2 == event.Policy {
				present = true
				break
			}
		}

		if !present {
			continue
		}

		for _, v2 :=  range v {
			if _, ok := store.ActivePolicies[v2]; ok {
				delete(store.ActivePolicies, v2)
			}
		}
	}
	
	store.ActivePolicies[event.Policy] = struct{}{}
}

func (g *Instance) NextTurnHandler(e event.Event, s*event.Store) {
	store := GetBranchStore(s)

	store.ActivePolicies = ReEvaluatePolicies(store.ActivePolicies, g.GameData.Policies, store.Values)
	store.Weights = CalculateWeightMap(g.GameData.Policies, g.GameData.Values, store.ActivePolicies)
	store.Values = RecountValues(store.Values, g.GameData.Values, store.Weights, g.GameData.Policies, store.ActivePolicies)
	store.Turn++
	if store.GameOver == 0 {
		store.GameOver = EvaluateGameEndConditions(store.Turn, &g.GameData.Scenario, store.Values)
	}
	fmt.Println(fmt.Sprintf("Beginning turn %d", store.Turn))
}

func (g *Instance) SetBranchHandler(e event.Event, s *event.Store) {
	event, ok := e.(*SetBranchEvent)
	if !ok {
		panic(EventCastFailError(SetBranchEventType, e.Type().String()))
	}
	store := GetGameStore(s)
	store.CurrentBranch = event.BranchID
}

func (g *Instance) SetScreenHandler(e event.Event, s *event.Store) {
	event, ok := e.(*SetScreenEvent)
	if !ok {
		panic(EventCastFailError(SetScreenEventType, e.Type().String()))
	}
	store := GetGameStore(s)
	store.CurrentScreen = event.Screen
}

func (g *Instance) SetSelectedValueHandler(e event.Event, s *event.Store) {
	event, ok := e.(*SetSelectedValueEvent)
	if !ok {
		panic(EventCastFailError(SetSelectedValueType, e.Type().String()))
	}
	store := GetGameStore(s)
	store.SelectedValue = event.Value
}

func (g *Instance) WindbackHandler(e event.Event, s *event.Store) {
	g.Dispatcher.WindbackHandler(e, s)
	timeStore, ok := s.Attributes.(*event.TimelineStore)
	if !ok {
		panic(fmt.Errorf("Store not of type TimelineStore"))
	}
	
	store := GetGameStore(s)
	store.Rewind = false
	if !SameTime(timeStore) {
		fmt.Println(fmt.Sprintf("Setting rewind to true"))
		store.Rewind = true
	}
}

func SameTime(store *event.TimelineStore) bool {
	gameStore, ok :=  store.Attributes.(*GameStore)
	if !ok {
		panic("Store is not of type GameStore")
	}
	
	branchIndex, ok := store.BranchDictionary[gameStore.CurrentBranch]
	if !ok {
		panic(fmt.Errorf("Unknown branch requested: %s", gameStore.CurrentBranch.ToString()))
	}
	branch := store.Branches[branchIndex]

	rewind := GetBranchStore(&store.RewindStores[branch.StoreID])
	normal := GetBranchStore(&store.Stores[branch.StoreID])
	if rewind.Turn != normal.Turn {
		return false
	}
	return true
}
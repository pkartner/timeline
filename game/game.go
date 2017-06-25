package game

import (
	"fmt"

	"github.com/pkartner/event"

	"github.com/boltdb/bolt"
)

type ValueMap map[string]float64
type WeightMap map[string]map[string]Weight

var Current *Instance

type MaxMin struct {
	Set bool `json:"set"`
	Value float64 `json:"value"`
}

type Weight struct {
	Multiplier float64
	Max MaxMin
	Min MaxMin
}

type Values struct {
	Values map[string]struct {
		Name string `json:"name"`
		NaturalChange float64 `json:"natural_change"`
		Min MaxMin `json:"min"`
		Max MaxMin `json:"max"`
		AffectedBy []struct{
			Name string `json:"name"`
			Weight float64 `json:"weight"`
			Min MaxMin `json:"min"`
			Max MaxMin `json:"max"`
		} `json:"affected_by"`
	} `json:"values"`

}

type Policies struct {
	Policies map[string]struct {
		Name string `json:"name"`
		FlatAmountPerTurn []struct {
			ValueName string `json:"value_name"`
			Amount float64 `json:"amount"`
		} `json:"flat"`
		WeightChange []struct {
			DestValueName string `json:"dest"`
			SourceValueName string `json:"source"`
			Weight float64 `json:"weight"`
		} `json:"weight_change"`
		Restrictions []struct {
			ValueName string `json:"value_name"`
			Amount float64 `json:"amount"`
		}
	} `json:"policies"`
	MutualExclusive [][]string `json:"mutual_exclusive"`
}

type GameEndCondition struct {
	Name string `json:"name"`
	Sign string `json:"sign"`
	Value float64 `json:"value"`
}

type Scenario struct {
	StartValues ValueMap `json:"start_values"`
	WinCondition *GameEndCondition `json:"win_condition"`
	LoseCondition *GameEndCondition `json:"lose_condition"`
}

type Instance struct {
	GameData *Data
	EventStore event.EventStore
	Dispatcher *event.TimelineDispatcher
}

type Data struct {
	Values Values
	Policies Policies
	Scenario Scenario
}

func (g *Instance) Restore() {
	event.RestoreEvents(g.EventStore, g.Dispatcher)
}

func CalculateWeightMap(policies Policies, values Values, activatedPolicies map[string]struct{}) WeightMap {
	weightMap := WeightMap{}
	for k, v := range values.Values {
		weightMap[k] = map[string]Weight{}
		for _, v2 := range v.AffectedBy {
			weight := Weight{
				Multiplier: v2.Weight,
			}
			weight.Max = v2.Max
			weight.Min = v2.Min
			weightMap[k][v2.Name] = weight
		}
	}
	for k := range activatedPolicies {
		for _, v := range policies.Policies[k].WeightChange {
			value := weightMap[v.DestValueName][v.SourceValueName]
			value.Multiplier = value.Multiplier + v.Weight
			weightMap[v.DestValueName][v.SourceValueName] = value
		}
	}

	return weightMap
}

func ReEvaluatePolicies(activePolicies map[string]struct{}, policies Policies, values ValueMap) map[string]struct{} {
	removePolicies := []string{}
	for k := range activePolicies {
		policy := policies.Policies[k]
		for _, v := range policy.Restrictions {
			if values[v.ValueName] < v.Amount {
				removePolicies = append(removePolicies, k)
			}
		}
	}
	newPolicies := map[string]struct{}{}
	for k, v := range activePolicies {
  		newPolicies[k] = v
	}
	for _, v := range removePolicies {
		delete(newPolicies, v)
	}
	
	return newPolicies
}

func EvaluateGameEndConditions(turn uint64, scenario *Scenario, values ValueMap) uint8 {
	if EvaluateGameEndCondition(turn, scenario.WinCondition, values) {
		return GameWon
	}
	if EvaluateGameEndCondition(turn, scenario.LoseCondition, values) {
		return GameLost
	}
	return 0
}

func EvaluateGameEndCondition(turn uint64, endCondition *GameEndCondition, values ValueMap) bool {
	var value float64
	if endCondition.Name == "turn" {
		value = float64(turn)
	} else {
		var ok bool
		value, ok = values[endCondition.Name]
		if !ok {
			panic("Value doesn't exist")
		}
	}
	if endCondition.Sign == "+" && value >= endCondition.Value{
		return true
	}
	if endCondition.Sign == "-" && value <= endCondition.Value{
		return true
	}
	return false
}

func RecountValues(values ValueMap, valueData Values, weights WeightMap, policies Policies, activatedPolicies map[string]struct{}) ValueMap {
	fmt.Println("Recounting values")
	newValues := ValueMap{}
	for k, v := range values {
		value := valueData.Values[k]
		newValue := v + CalculateAddedValue(k, values, weights, policies, activatedPolicies)
		newValue += value.NaturalChange
		if value.Min.Set && newValue < value.Min.Value {
			newValue = value.Min.Value
		}
		if value.Max.Set && newValue > value.Max.Value {
			newValue = value.Max.Value
		}

		newValues[k] = newValue
	}
	return newValues
}

func CalculateAddedValue(key string, values ValueMap, weights WeightMap, policies Policies, activatedPolicies map[string]struct{}) float64 {
	fmt.Println(fmt.Sprintf("Recounting value for %s", key))

	weightsForValue := weights[key]
	adjustment := 0.0
	for k, v := range weightsForValue {
		weightAdjustment := values[k] * v.Multiplier
		if v.Max.Set && weightAdjustment > v.Max.Value {
			weightAdjustment = v.Max.Value
		}
		if v.Min.Set && weightAdjustment < v.Min.Value {
			weightAdjustment = v.Min.Value
		}
		adjustment += weightAdjustment
	}
	for k := range activatedPolicies {
		for _, v := range policies.Policies[k].FlatAmountPerTurn {
			if v.ValueName != key {
				continue
			}
			fmt.Println("Adding value")
			adjustment += v.Amount
		}
	}
	return adjustment
}

func NewGame(fileName string, GameData *Data) {
	databaseFileName := fileName+".db"
    db, err := bolt.Open(databaseFileName, 0600, nil)
    if nil != err {
        panic(err)
    }
    eventStore := event.NewBoltEventStore(db)
    timeStore := event.NewTimelineStore(NewBranchStoreFunc(GameData.Scenario.StartValues), event.Reloader{
        EventStore: eventStore,
    }, nil)
	timeStore.Attributes = &GameStore{}
    dispatcher := event.NewTimelineDispatcher(timeStore)
    dispatcher.SetMiddleware(
        event.EventStoreMiddleware(eventStore),
    )

	Current = &Instance{
		GameData: GameData,
		Dispatcher: dispatcher,
		EventStore: eventStore,
	}

    dispatcher.Dispatcher.Register(&event.WindbackEvent{}, Current.WindbackHandler)
    dispatcher.Dispatcher.Register(&event.NewBranchEvent{}, dispatcher.NewBranchHandler)
    dispatcher.Dispatcher.Register(&SetBranchEvent{}, Current.SetBranchHandler)
	dispatcher.Dispatcher.Register(&SetScreenEvent{}, Current.SetScreenHandler)
	dispatcher.Dispatcher.Register(&SetSelectedValueEvent{}, Current.SetSelectedValueHandler)

	dispatcher.Register(&NextTurnEvent{}, Current.NextTurnHandler)
    dispatcher.Register(&SetPolicyEvent{}, Current.SetPolicyHandler)
	eventStore.PrintAllEvents()
	eventStore.Restore(^uint64(0),func(e event.Event) error {
		fmt.Println(fmt.Sprintf("Loading back event with time %d and type %s", e.Time(), e.Type()))
		if err := dispatcher.Handle(e); nil != err {
			panic(err)
		}
		return nil
	})
	
}

func (g *Instance) GetTimeLineStore() *event.TimelineStore{
	store, ok := g.Dispatcher.Store.Attributes.(*event.TimelineStore)
	if !ok {
		panic("Store is not of type TimelineStore")
	}
	return store
}

func (g *Instance) GetGameStore() *GameStore {
	timeStore := g.GetTimeLineStore()
	gameStore, ok := timeStore.Attributes.(*GameStore)
	if !ok {
		panic("Store is not of type GameStore")
	}
	return gameStore
}

func (g *Instance) GetCurrentBranchStore() *BranchStore{
	timeStore := g.GetTimeLineStore()
	gameStore := g.GetGameStore()
	branch, err := timeStore.GetBranch(gameStore.CurrentBranch)
	if nil != err {
		panic(err)
	}
	storeID := branch.StoreID
	branchStore := timeStore.Stores[storeID]
	returnStore, ok := branchStore.Attributes.(*BranchStore)
	if !ok {
		panic("Store is not of type BranchStore")
	}
	return returnStore
}

func (g *Instance) GetRewindedBranchStore() *BranchStore{
	branchStore := g.GetRewindedStore()
	returnStore, ok := branchStore.Attributes.(*BranchStore)
	if !ok {
		panic("Store is not of type BranchStore")
	}
	return returnStore
}

func (g *Instance) GetRewindedStore() *event.Store{
	timeStore := g.GetTimeLineStore()
	gameStore := g.GetGameStore()
	branch, err := timeStore.GetBranch(gameStore.CurrentBranch)
	if nil != err {
		panic(err)
	}
	storeID := branch.StoreID
	branchStore := &timeStore.Stores[storeID]
	if gameStore.Rewind {
		branchStore = &timeStore.RewindStores[storeID]
	}

	return branchStore
}
package game

import (
	"fmt"

	"github.com/pkartner/event"

	"github.com/boltdb/bolt"
)

type ValueMap map[string]float64
type WeightMap map[string]map[string]float64

var Current *Instance

type Values struct {
	Values map[string]struct {
		Name string `json:"name"`
		NaturalChange float64 `json:"natural_change"`
		Min struct {
			Set bool `json:"set"`
			Value float64 `json:"value"`
		} `json:"min"`
		Max struct {
			Set bool `json:"set"`
			Value float64 `json:"value"`
		} `json:"max"`
		AffectedBy []struct{
			Name string `json:"name"`
			Weight float64 `json:"weight"`
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
			DestValueName string `json:"dest_value_name"`
			SourceValueName string `json:"source_value_name"`
			Weight float64 `json:"weight"`
		} `json:"weight_change"`
	} `json:"policies"`
	MutualExclusive [][]string `json:"mutual_exclusive"`
}

type Instance struct {
	GameData *Data
	EventStore event.EventStore
	Dispatcher *event.TimelineDispatcher
}

type Data struct {
	Values Values
	Policies Policies
	Startvalues ValueMap
}

func (g *Instance) Restore() {
	event.RestoreEvents(g.EventStore, g.Dispatcher)
}

func CalculateWeightMap(policies Policies, values Values, activatedPolicies map[string]struct{}) WeightMap {
	weightMap := WeightMap{}
	for k, v := range values.Values {
		weightMap[k] = map[string]float64{}
		for _, v2 := range v.AffectedBy {
			weightMap[k][v2.Name] = v2.Weight
		}
	}
	for k := range activatedPolicies {
		for _, v := range policies.Policies[k].WeightChange {
			value := weightMap[v.DestValueName][v.SourceValueName]
			weightMap[v.DestValueName][v.SourceValueName] = value + v.Weight
		}
	}

	return weightMap
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
		adjustment += values[k] * v
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
    timeStore := event.NewTimelineStore(NewBranchStoreFunc(GameData.Startvalues), event.Reloader{
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
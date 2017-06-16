package main

import (
	"fmt"
	"io/ioutil"
	"encoding/json"
	"time"


	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"

	"github.com/pkartner/event"
	"github.com/pkartner/timeline/game"
	
	//"github.com/faiface/pixel/imdraw"
)

const(
	ResolutionX = 1600
	ResolutionY = 900
)

func PolicyClickHandler(policy string) game.GuiEventHandler {
	return func(interface{}) {
		fmt.Println("Policy set event fired policy: "+ policy)
		game.Current.Dispatcher.Dispatch(game.SetPolicy(policy))
	}
}

func MenuButtonHandler(screen string) game.GuiEventHandler {
	return func(interface{}) {
		fmt.Println(fmt.Sprintf("Button %s pressed", screen))
		game.Current.Dispatcher.Dispatch(game.SetScreen(screen))
	}
}

func ValueClickHandler(value string) game.GuiEventHandler {
	return func(interface{}) {
		game.Current.Dispatcher.Dispatch(game.SetSelectedValue(value))
	}
}

func ValueStringProvider(value string) game.GuiStringProviderFunc {
	return func() string{
		valueNumber := game.Current.GetRewindedBranchStore().Values[value]
		//valueNumber := game.Current.GetCurrentBranchStore().Values[value]
		return fmt.Sprintf(value + ": %.2f", valueNumber)
	}
}

func EndturnHandler() game.GuiEventHandler {
	return func(interface{}) {
		game.Current.Dispatcher.Dispatch(game.NextTurn())
	}
}

func GotoBranch() game.GuiEventHandler {
	return func(arguments interface{}) {
		a := arguments.(*game.TimelineClicked)
		now := uint64(time.Now().Unix())
		lastID := game.Current.Dispatcher.Store.LastEvent.ID().IDPart()
		game.Current.Dispatcher.Dispatch(game.SetBranch(a.Branch))
		game.Current.Dispatcher.Dispatch(event.Windback(a.Time, now, lastID+1))

	}
}

func NewBranchHandler() game.GuiEventHandler {
	return func(interface{}) {
		gameStore := game.Current.GetGameStore()
		currentBranchID := gameStore.CurrentBranch
		store := game.Current.GetRewindedStore()
		branchStore := game.Current.GetRewindedBranchStore()

		lastEventID := event.ID{}
		if store.LastEvent != nil {
			lastEventID = store.LastEvent.ID()
		}
		
		currentTime := time.Now().Unix()
		lastID := game.Current.Dispatcher.Store.LastEvent.ID().IDPart()
		fmt.Println(fmt.Sprintf("Starting new branch at turn %d", branchStore.Turn))
		newBranchEvent := event.NewBranch(branchStore.Turn, currentBranchID, lastEventID, uint64(currentTime), lastID+1)
		fmt.Println(fmt.Sprintf("Prev branch id is: %s", currentBranchID.ToString()))
		fmt.Println(fmt.Sprintf("New branch id is: %s", newBranchEvent.NewBranchID.ToString()))
		fmt.Println(fmt.Sprintf("Currently we are rewinded: %t", gameStore.Rewind))
		game.Current.Dispatcher.Dispatch(newBranchEvent)
	}
}

func LoadData() *game.Data{
	gameData := game.Data{}
	// policies := game.Policies{}
	// values := game.Values{}
	// startValues := game.ValueMap{}
	data, err := ioutil.ReadFile("data/policies.json")
	if err != nil {
		panic (err)
	}	
	if err := json.Unmarshal(data, &gameData.Policies); err != nil {
		panic(err)
	}
	data, err = ioutil.ReadFile("data/values.json")
	if err != nil {
		panic (err)
	}	
	if err := json.Unmarshal(data, &gameData.Values); err != nil {
		panic(err)
	}
	data, err = ioutil.ReadFile("data/startvalues.json")
	if err != nil {
		panic (err)
	}
	if err := json.Unmarshal(data, &gameData.Startvalues); err != nil {
		panic(err)
	}
	
	return &gameData
}

func run() {
	cfg := pixelgl.WindowConfig{
		Title: "Timeline",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync: true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// Load Data
	fmt.Println("Loading data")
	gameData := LoadData()

	// Create Game object
	game.NewGame("time", gameData)

	gameStore := game.Current.GetGameStore()

	// Create GUI
	menu := game.GuiMenu{
		Position: pixel.Vec{X: 10, Y: 768-60},
		Bound: 20.0,
	}

	mainMenuItem := game.NewGuiMenuItem("Main")
	dataMenuItem := game.NewGuiMenuItem("Detail")
	policiesMenuItem := game.NewGuiMenuItem("Policies")
	timelineMenuItem := game.NewGuiMenuItem("Timeline")
	endTurnItem := game.NewGuiMenuItem("End Turn")
	newBranchItem := game.NewGuiMenuItem("New Branch")
	mainMenuItem.OnMouseClick = MenuButtonHandler("main")
	dataMenuItem.OnMouseClick = MenuButtonHandler("detail")
	policiesMenuItem.OnMouseClick = MenuButtonHandler("policies")
	timelineMenuItem.OnMouseClick = MenuButtonHandler("timeline")
	endTurnItem.OnMouseClick = EndturnHandler()
	newBranchItem.OnMouseClick = NewBranchHandler()

	menu.AddItem(mainMenuItem)
	menu.AddItem(dataMenuItem)
	menu.AddItem(policiesMenuItem)
	menu.AddItem(timelineMenuItem)
	menu.AddItem(endTurnItem)
	menu.AddItem(newBranchItem)

	policyList := game.GuiPolicyList{
		Position: pixel.Vec{X: 32, Y: 768-128},
	}
	for _, v := range gameData.Policies.Policies {
		guiPolicy := game.NewGuiPolicy(v.Name)
		guiPolicy.Dimension = pixel.Vec{X: 600, Y: 32}
		guiPolicy.OnMouseClick = PolicyClickHandler(v.Name)
		policyList.AddPolicy(guiPolicy)
	}
	valueList := game.GuiPolicyList{
		Position: pixel.Vec{X: 32, Y: 768-128},
	}
	for _, v := range gameData.Values.Values {
		guiValue := game.NewGuiPolicy(v.Name)
		guiValue.Dimension = pixel.Vec{X: 600, Y: 32}
		guiValue.OnMouseClick = ValueClickHandler(v.Name)
		guiValue.StringProvider = ValueStringProvider(v.Name)
		valueList.AddPolicy(guiValue)
	}
	guiTimeline := game.NewGuiTimeline(pixel.V(80, 768-128))
	guiTimeline.OnMouseClick = GotoBranch()

	mainScreen := game.GuiScreen{}
	detailScreen := game.GuiScreen{}
	policyScreen := game.GuiScreen{}
	timelineScreen := game.GuiScreen{}

	policyScreen.AddDrawable(&policyList)
	policyScreen.AddClickable(&policyList)
	mainScreen.AddDrawable(&valueList)
	mainScreen.AddClickable(&valueList)
	timelineScreen.AddDrawable(guiTimeline)
	timelineScreen.AddClickable(guiTimeline)

	screens := map[string]*game.GuiScreen {
		"main": &mainScreen,
		"detail": &detailScreen,
		"policies": &policyScreen,
		"timeline": &timelineScreen,
	}
	game.Current.Dispatcher.Dispatch(event.NewBranch(^uint64(0), event.ZeroID(), event.ZeroID(), uint64(time.Now().Unix()), 0))
	branchID := game.Current.GetTimeLineStore().Branches[0].BranchID
	game.Current.Dispatcher.Dispatch(game.SetBranch(branchID))
	game.Current.Dispatcher.Dispatch(game.SetScreen("policies"))

	for !win.Closed() {
		screen := screens[gameStore.CurrentScreen]
		if win.JustPressed(pixelgl.MouseButtonLeft) {
			mousePosition := win.MousePosition()
			screen.CheckMouse(game.LeftClick, mousePosition)
			menu.CheckMouse(game.LeftClick, mousePosition)
		}
		win.Clear(colornames.White)
		screen.Draw(win, pixel.ZV)
		menu.Draw(win, pixel.ZV)
		win.Update()
	}
}

func main() {
	pixelgl.Run(run)
}
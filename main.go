package main

import (
	"fmt"
	"io/ioutil"
	"encoding/json"
	"os"
	"time"
	"sort"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"

	"github.com/pkartner/event"
	"github.com/pkartner/timeline/game"
)

const(
	ResolutionX = 1600
	ResolutionY = 900
)

func SaveGameFileNames () []string {
	return  []string{
		"Game1",
		"Game2",
		"Game3",
		"Game4",
	}
}

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
		return fmt.Sprintf(value + ": %.2f", valueNumber)
	}
}

func SaveGameListClickedHandler(gameStarted *bool, gameData *game.Data) game.GuiEventHandler {
	return func(value interface{}) {
		gameClicked, ok := value.(*game.SaveGameClicked)
		if !ok {
			panic("Interface not of type SaveGameClicked")
		}
		dir := "save"
		databaseFileName := dir+"/"+gameClicked.Filename+".db"
		if gameClicked.Remake {
			if _, err := os.Stat(databaseFileName); !os.IsNotExist(err) {
				if err := os.Remove(databaseFileName); nil != err {
					panic(err)
				}
			}
		}
		newGame := false
		if _, err := os.Stat(databaseFileName); os.IsNotExist(err) {
			newGame = true
		}
		os.Mkdir(dir, os.ModePerm)
		game.NewGame(dir+"/"+gameClicked.Filename, gameData)
		if newGame {
			game.Current.Dispatcher.Dispatch(event.NewBranch(0, event.ZeroID(), event.ZeroID(), uint64(time.Now().Unix()), 0))
			branchID := game.Current.GetTimeLineStore().Branches[0].BranchID
			game.Current.Dispatcher.Dispatch(game.SetBranch(branchID))
			game.Current.Dispatcher.Dispatch(game.SetScreen("main"))
		}
		*gameStarted = true
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
		game.Current.Dispatcher.Dispatch(game.SetBranch(a.Branch))
		lastID := game.Current.Dispatcher.Store.LastEvent.ID().IDPart()
		game.Current.Dispatcher.Dispatch(event.Windback(a.Time, uint64(time.Now().Unix()), lastID+1))

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
	data, err = ioutil.ReadFile("data/scenario.json")
	if err != nil {
		panic (err)
	}
	if err := json.Unmarshal(data, &gameData.Scenario); err != nil {
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
	gameStarted := false

	// Create GUI
	menu := game.GuiMenu{
		Position: pixel.Vec{X: 10, Y: 768-30},
		Bound: 20.0,
	}

	endTurnItem := game.NewGuiMenuItem("End Turn")
	newBranchItem := game.NewGuiMenuItem("New Branch")
	endTurnItem.OnMouseClick = EndturnHandler()
	newBranchItem.OnMouseClick = NewBranchHandler()

	menu.AddItem(endTurnItem)
	menu.AddItem(newBranchItem)

	policyList := game.GuiPolicyList{
		Position: pixel.Vec{X: 500, Y: 768-64},
	}
	policyNames := []string{}
	for k := range gameData.Policies.Policies {
		policyNames = append(policyNames, k)
	}
	sort.Strings(policyNames)
	for _, v := range policyNames {
		policy := gameData.Policies.Policies[v]
		guiPolicy := game.NewGuiPolicy(policy.Name)
		guiPolicy.OnMouseClick = PolicyClickHandler(policy.Name)
		policyList.AddPolicy(guiPolicy)
	}
	valueList := game.GuiPolicyList{
		Position: pixel.Vec{X: 32, Y: 768-64},
	}
	valueNames := []string{}
	for k := range gameData.Values.Values {
		valueNames = append(valueNames, k)
	}
	sort.Strings(valueNames)
	for _, v := range valueNames {
		value := gameData.Values.Values[v]
		guiValue := game.NewGuiPolicy(value.Name)
		guiValue.OnMouseClick = ValueClickHandler(value.Name)
		guiValue.StringProvider = ValueStringProvider(value.Name)
		valueList.AddPolicy(guiValue)
	}
	guiTimeline := game.NewGuiTimeline(pixel.V(80, 768-400))
	guiTimeline.OnMouseClick = GotoBranch()

	saveGameList := game.NewSaveGameList(pixel.Vec{X: 350, Y: 768-250})
	fileNameList := SaveGameFileNames()
	for _, v := range fileNameList {
		databaseFileName := "save/"+v+".db"
		fileExists := false
		if _, err := os.Stat(databaseFileName); !os.IsNotExist(err) {
			fileExists = true
		}
		saveGameList.AddFile(v, fileExists)
	}
	saveGameList.OnMouseClick = SaveGameListClickedHandler(&gameStarted, gameData)

	winText := game.NewGuiBigText("You Won :)", pixel.V(350, 600))
	loseText := game.NewGuiBigText("You Lost :(", pixel.V(350, 600))

	mainScreen := game.GuiScreen{}
	winScreen := game.GuiScreen{}
	loseScreen := game.GuiScreen{}

	mainScreen.AddDrawable(&policyList)
	mainScreen.AddClickable(&policyList)
	mainScreen.AddDrawable(&valueList)
	mainScreen.AddClickable(&valueList)
	mainScreen.AddDrawable(guiTimeline)
	mainScreen.AddClickable(guiTimeline)
	mainScreen.AddClickable(&menu)
	mainScreen.AddDrawable(&menu)

	winScreen.AddDrawable(guiTimeline)
	winScreen.AddClickable(guiTimeline)
	winScreen.AddDrawable(winText)

	loseScreen.AddDrawable(guiTimeline)
	loseScreen.AddClickable(guiTimeline)
	loseScreen.AddDrawable(loseText)

	screens := map[string]*game.GuiScreen {
		"main": &mainScreen,
	}

	goLeft := false
	goLeftTimerExp := false
	goRight := false
	goRightTimerExp := false
	for !win.Closed() {
		win.Clear(colornames.White)
		if gameStarted {
			gameStore := game.Current.GetGameStore()
			branchStore := game.Current.GetRewindedBranchStore()
			screen := screens[gameStore.CurrentScreen]
			if branchStore.GameOver == game.GameWon {
				screen = &winScreen
			}
			if branchStore.GameOver == game.GameLost {
				screen = &loseScreen
			}
			if win.JustPressed(pixelgl.MouseButtonLeft) {
				mousePosition := win.MousePosition()
				screen.CheckMouse(game.LeftClick, mousePosition)
			}
			timelineDeltaX := 0.0
			if win.JustPressed(pixelgl.KeyA){
				timer := time.NewTimer(time.Millisecond*400)
				goLeftTimerExp = false
				go func() {
					<- timer.C
					goLeftTimerExp = true
				}()
				goLeft = true
				timelineDeltaX = -25.0
			}
			if win.JustPressed(pixelgl.KeyD) {
				timer := time.NewTimer(time.Millisecond*400)
				goRightTimerExp = false
				go func() {
					<- timer.C
					goRightTimerExp = true
				}()
				goRight = true
				timelineDeltaX = 25.0
			}
			if win.JustReleased(pixelgl.KeyA) {
				goLeft = false
			}
			if win.JustReleased(pixelgl.KeyD) {
				goRight = false
			}

			if goLeft && goLeftTimerExp {
				timelineDeltaX = -8.0
			}
			if goRight && goRightTimerExp {
				timelineDeltaX = 8.0
			}

			guiTimeline.Position.X += timelineDeltaX
			
			screen.Draw(win, pixel.ZV)
		} else {
			if win.JustPressed(pixelgl.MouseButtonLeft) {
				mousePosition := win.MousePosition()
				saveGameList.CheckMouse(game.LeftClick, mousePosition)
			}
			saveGameList.Draw(win, pixel.ZV)
		}

		win.Update()
	}
}

func main() {
	pixelgl.Run(run)
}
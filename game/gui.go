package game

import (
	"fmt"
	"unicode"

	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gomono"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/text"
	"github.com/golang/freetype/truetype"

	"github.com/pkartner/event"
)

const (
	LeftClick = "LeftClick"
)

type GuiEventHandler func(arguments interface{})

type GuiEvent struct {
	Handler GuiEventHandler
}

type Clickable interface {
	CheckMouse(key string, MousePosition pixel.Vec) bool
}

type Drawable interface {
	Draw(t pixel.Target, v pixel.Vec)
}

func ttfFromBytesMust(b []byte, size float64) font.Face {
	ttf, err := truetype.Parse(b)
	if err != nil {
		panic(err)
	}
	return truetype.NewFace(ttf, &truetype.Options{
		Size:              size,
		GlyphCacheEntries: 1,
	})
}

type GuiElement struct {
	
}

type GuiStringProvider interface {
	Provide() string
}

type GuiStringProviderFunc func() string

func (f GuiStringProviderFunc) Provide() string {
	return f()
}

type GuiPolicy struct {
	Name string
	NormalText *text.Text
	SelectedText *text.Text
	Background imdraw.IMDraw
	Position pixel.Vec
	Dimension pixel.Vec
	StringProvider GuiStringProvider
	OnMouseClick GuiEventHandler
}

func NewGuiPolicy(name string) *GuiPolicy {
	policy := GuiPolicy{
		Name: name,
	}
	regular := text.NewAtlas(
		ttfFromBytesMust(goregular.TTF, 42),
		text.ASCII, text.RangeTable(unicode.Latin),
	)
	policy.NormalText = text.New(pixel.ZV, regular)
	policy.NormalText.Color = pixel.ToRGBA(colornames.Black)
	_, err := policy.NormalText.WriteString(name)
	if err != nil {
		panic(err)
	}
	policy.SelectedText = text.New(pixel.ZV, regular)
	policy.SelectedText.Color = pixel.ToRGBA(colornames.Green)
	_, err = policy.SelectedText.WriteString(name)
	if err != nil {
		panic(err)
	}

	return &policy
}

func (p *GuiPolicy) Draw(t pixel.Target, v pixel.Vec) {
	if nil != p.StringProvider {
		p.Name = p.StringProvider.Provide()
		p.NormalText.Clear()
		p.NormalText.Dot = pixel.V(0, 0)
		p.NormalText.WriteString(p.Name)
		//p.StringProvider = nil
	}
	//store := Current.GetCurrentBranchStore()
	store := Current.GetRewindedBranchStore()
	m := pixel.IM.Moved(v)
	_, ok := store.ActivePolicies[p.Name]
	if ok {
		p.SelectedText.Draw(t, m)	
		return
	}
	p.NormalText.Draw(t, m)
}

func (p *GuiPolicy) CheckMouse(key string, MousePosition pixel.Vec) bool {
	//fmt.Println("CheckMouse for policy: "+p.Name)
	if p.OnMouseClick == nil {
		return false
	}
	if !p.Inside(MousePosition) {
		return false
	}
	p.OnMouseClick(nil)
	return true
}

func (p *GuiPolicy) Inside(pos pixel.Vec) bool {
	//fmt.Println(fmt.Sprintf("MouseX: %.2f, MouseY: %.2f, Width: %.2f, Height: %.2f", pos.X, pos.Y, p.Dimension.X, p.Dimension.Y))
	relativePos := pos.Sub(p.Position)
	if 	relativePos.X >= 0 && 
		relativePos.X <= p.Dimension.X &&
		relativePos.Y >= 0 &&
		relativePos.Y <= p.Dimension.Y {
			return true
	}
	return false
}

type GuiPolicyList struct {
	Policies []*GuiPolicy
	Position pixel.Vec
}

func (g *GuiPolicyList) AddPolicy(p *GuiPolicy) {
	g.Policies = append(g.Policies, p)
}

func (g *GuiPolicyList) CheckMouse(key string, MousePosition pixel.Vec) bool {
	y := 0.0
	for _, v := range g.Policies {
		addedVector := pixel.Vec{X: 0, Y: y}
		position := MousePosition.Sub(g.Position)
		position = position.Sub(addedVector)
		if v.CheckMouse(key, position) {
			return true
		}
		y -= 32
	}
	return false
}

func (g *GuiPolicyList) Draw(t pixel.Target, relPos pixel.Vec) {
	y := 0.0
	for _, v := range g.Policies {
		addedVector := pixel.Vec{X: 0, Y: y}
		position := g.Position.Add(addedVector)
		position = position.Add(relPos)
		v.Draw(t, position)
		y -= 32
	}
}

type GuiMenu struct {
	Items []*GuiMenuItem
	Position pixel.Vec
	Bound float64
}

func (m *GuiMenu) AddItem(mi *GuiMenuItem) {
	m.Items = append(m.Items, mi)
}

func (m *GuiMenu) Draw(t pixel.Target, relPos pixel.Vec) {
	x := 0.0
	for _, v := range m.Items {
		addedVector := pixel.Vec{X: x, Y: 0}
		position := m.Position.Add(addedVector)
		position = position.Add(relPos)
		v.Draw(t, position)
		x += v.Text.Bounds().W() + m.Bound
	}
}

func (m *GuiMenu) CheckMouse(key string, MousePosition pixel.Vec) bool {
	x := 0.0
	for _, v := range m.Items {
		addedVector := pixel.Vec{X: x, Y: 0}
		position := MousePosition.Sub(m.Position)
		position = position.Sub(addedVector)
		if v.CheckMouse(key, position) {
			return true
		}
		x += v.Text.Bounds().W() + m.Bound
	}
	return false
}

type GuiMenuItem struct {
	GuiClickable
	Label string
	Text *text.Text
}

func (mi *GuiMenuItem) Draw(t pixel.Target, v pixel.Vec) {
	m := pixel.IM.Moved(v)
	mi.Text.Draw(t, m)
}

func NewGuiMenuItem(label string) *GuiMenuItem {
	item := GuiMenuItem{
		Label: label,
	}
	regular := text.NewAtlas(
		ttfFromBytesMust(goregular.TTF, 42),
		text.ASCII, text.RangeTable(unicode.Latin),
	)
	item.Text = text.New(pixel.ZV, regular)
	item.Text.Color = pixel.ToRGBA(colornames.Black)
	_, err := item.Text.WriteString(label)
	if err != nil {
		panic(err)
	}
	item.Dimension = pixel.V(item.Text.Bounds().W(), item.Text.Bounds().H())
	return &item
}

type GuiClickable struct {
	Position pixel.Vec
	Dimension pixel.Vec
	OnMouseClick GuiEventHandler
}

func (c *GuiClickable) CheckMouse(key string, MousePosition pixel.Vec) bool {
	if c.OnMouseClick == nil {
		return false
	}
	if !c.Inside(MousePosition) {
		return false
	}
	c.OnMouseClick(nil)
	return true
}

func (c *GuiClickable) Inside(pos pixel.Vec) bool {
	relativePos := pos.Sub(c.Position)
	if 	relativePos.X >= 0 && 
		relativePos.X <= c.Dimension.X &&
		relativePos.Y >= 0 &&
		relativePos.Y <= c.Dimension.Y {
			return true
	}
	return false
}

type GuiScreen struct {
	Clickables []Clickable
	Drawables []Drawable
}

func (s *GuiScreen) AddClickable(c Clickable) {
	s.Clickables = append(s.Clickables, c)
}

func (s *GuiScreen) AddDrawable(d Drawable) {
	s.Drawables = append(s.Drawables, d)
}

func (s *GuiScreen) Draw(tar pixel.Target, vec pixel.Vec) {
	for _, v := range s.Drawables {
		v.Draw(tar, vec)
	}
}

func (s *GuiScreen) CheckMouse(key string, mousePosition pixel.Vec) bool {
	for _, v := range s.Clickables {
		if v.CheckMouse(key, mousePosition) {
			return true
		}
	}
	return false
}


type GuiTimeLine struct {
	Position pixel.Vec
	Text *text.Text
	SideLabels []*text.Text
	MaxTime uint64
	OnMouseClick GuiEventHandler	
}

func NewGuiTimeline(position pixel.Vec) *GuiTimeLine{
	timeline := GuiTimeLine{
		Position: position,
	}
	regular := text.NewAtlas(
		ttfFromBytesMust(gomono.TTF, 16),
		text.ASCII, text.RangeTable(unicode.Latin),
	)
	timeline.Text = text.New(pixel.ZV, regular)
	timeline.Text.Color = pixel.ToRGBA(colornames.Black)

	return &timeline
}

func (g *GuiTimeLine) Draw(tar pixel.Target, vec pixel.Vec) {
	timelineStore := Current.GetTimeLineStore()
	maxTime := uint64(0)
	sliceLength := len(timelineStore.Branches)
	numberOfSubBranches := make([]int, sliceLength, sliceLength)
	numberBranch := make([]int, sliceLength, sliceLength)
	branchLabels := []string{}
	for _, v := range timelineStore.Branches {
		if v.PrevBranch == event.ZeroID() {
			continue
		}
		prevIndex := timelineStore.BranchDictionary[v.PrevBranch]
		branchIndex := timelineStore.BranchDictionary[v.BranchID]
		subBranchCount := numberOfSubBranches[prevIndex]
		numberOfSubBranches[prevIndex] = subBranchCount+1
		numberBranch[branchIndex] = subBranchCount
	}

	for _, v := range timelineStore.Branches {
		var recFunc func(event.ID) string 
		recFunc = func(id event.ID) string  {
			branch, err := timelineStore.GetBranch(id)
			if err != nil {
				panic(err)
			}
			if branch.PrevBranch == event.ZeroID() {
				return "R"
			}
			branchIndex := timelineStore.BranchDictionary[branch.BranchID]
			subBranchCount := numberBranch[branchIndex]
			//numberOfSubBranches[branchIndex] = subBranchCount+1
			return recFunc(branch.PrevBranch)+fmt.Sprintf(".%d", subBranchCount)
		}
		label := recFunc(v.BranchID)
		branchLabels = append(branchLabels, label)
	}

	for i := len(g.SideLabels); i < len(branchLabels); i++ {
		regular := text.NewAtlas(
			ttfFromBytesMust(gomono.TTF, 16),
			text.ASCII, text.RangeTable(unicode.Latin),
		)
		labelText := text.New(pixel.ZV, regular)
		labelText.Color = pixel.ToRGBA(colornames.Black)
		_, err := labelText.WriteString(branchLabels[i])
		if nil != err {
			panic(err)
		}
		g.SideLabels = append(g.SideLabels, labelText)
	}

	for _, v := range timelineStore.Branches {
		if v.LastEventTime > maxTime {
			maxTime = v.LastEventTime
		}		
	}

	labelY := -20.0
	for _, v := range g.SideLabels {
		pos := pixel.V(vec.X, labelY)
		pos = g.Position.Add(pos)
		pos = pos.Add(pixel.V(-v.Bounds().W(), 0))
		m := pixel.IM.Moved(pos)
		v.Draw(tar, m)
		labelY -= 20
	}
	
	for i := g.MaxTime; i <= maxTime + 1; i++ {
		fmt.Println(fmt.Sprintf("current i is %d", i))
		fmt.Println(fmt.Sprintf("current maxTime is %d", maxTime))
		g.Text.WriteString(fmt.Sprintf("%03d ", i))
		g.MaxTime++
	}
	pos := g.Position.Add(vec)
	pos = pos.Add(pixel.V(20, 0))
	m := pixel.IM.Moved(pos)
	g.Text.Draw(tar, m)

	imd := imdraw.New(nil)
	for k, v := range timelineStore.Branches {
		starttime := uint64(0)
		if v.PrevBranchLastEvent != event.ZeroID() {
			starttime = v.CreationTime
		}
		y := -20 * float64(k)
		nodeStartPos := pixel.V(18, -20)
		branchStore := GetBranchStore(&timelineStore.Stores[v.StoreID])
		for i := starttime; i <= branchStore.Turn; i++ {
			color := pixel.RGB(0,0,0)
			if v.BranchID == Current.GetGameStore().CurrentBranch && Current.GetRewindedBranchStore().Turn == i {
				color = pixel.RGB(0,1,0)
			}
			x := 38.5 * float64(i)
			nodePos := pixel.V(x, y)
			nodePos = nodePos.Add(nodeStartPos)
			nodePos = nodePos.Add(g.Position)
			drawRectangle(imd, color, nodePos, pixel.V(30, 15))
		}
	}
	imd.Draw(tar)
}

type TimelineClicked struct {
	Branch event.ID
	Time uint64
}

func (g *GuiTimeLine) CheckMouse(key string, mousePosition pixel.Vec) bool {
	timelineStore := Current.GetTimeLineStore()
	maxTime := uint64(0)
	for _, v := range timelineStore.Branches {
		if v.LastEventTime > maxTime {
			maxTime = v.LastEventTime
		}		
	}
	for k, v := range timelineStore.Branches {
		starttime := uint64(0)
		if v.PrevBranchLastEvent != event.ZeroID() {
			starttime = v.CreationTime
		}
		y := -20 * float64(k)
		nodeStartPos := pixel.V(18, -20)
		for i := starttime; i <= maxTime; i++ {
			x := 38.5 * float64(i)
			nodePos := pixel.V(x, y)
			nodePos = nodePos.Add(nodeStartPos)
			nodePos = nodePos.Add(g.Position)
			relMousePos := mousePosition.Sub(nodePos)
			if insideRect(relMousePos, pixel.V(30, 15)) {
				g.OnMouseClick(&TimelineClicked{v.BranchID,i})
				fmt.Println(fmt.Sprintf("branch: %d, time: %d", k, i))
				return true
			}
		}
	}
	return false
}

func insideRect(position, dimension pixel.Vec) bool {
	if 	position.X >= 0 && 
		position.X <= dimension.X &&
		position.Y >= 0 &&
		position.Y <= dimension.Y {
			return true
	}
	return false
}

func drawRectangle(imd *imdraw.IMDraw, color pixel.RGBA, v pixel.Vec, b pixel.Vec) {
	imd.Color = color
	pos1 := v
	pos2 := v.Add(pixel.V(b.X, 0))
	pos3 := v.Add(b)
	pos4 := v.Add(pixel.V(0, b.Y))
	imd.Push(pos1)
	imd.Push(pos2)
	imd.Push(pos3)
	imd.Push(pos4)
	imd.Polygon(0)
}
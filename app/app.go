package app

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

//go:embed assets/*
var folder embed.FS

type Model struct {
	list     list.Model // white noise options
	selected string     // which choice is selected
	stop     chan bool  // a way to stop the running track
	err      errMsg     // any errors
}

type statusMsg int

type errMsg struct{ err error }

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

var (
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("231"))
	docStyle          = lipgloss.NewStyle().Margin(1, 2)
	selectedStyle     = lipgloss.NewStyle().Background(lipgloss.Color("103")).Foreground(lipgloss.Color("16")).Padding(0, 1)
	selectedDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("103"))

	defaultTitle = "Choose a sound to play with space or enter"
)

func InitialModel() Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = selectedStyle.MarginLeft(2)
	d.Styles.SelectedDesc = selectedDescStyle.MarginLeft(2)

	m := Model{
		list: list.New(
			[]list.Item{
				item{title: "Rain", desc: "Rain_Storm.wav by rambler52, https://freesound.org/s/332116/, License: Attribution 4.0"},
				item{title: "Forest", desc: "forest summer Roond 005 200619_0186.wav by klankbeeld, https://freesound.org/s/524238/, License: Attribution 4.0"},
				item{title: "Train car", desc: "Empty train moving slowly (recorded inside passenger car) by avakas, https://freesound.org/s/197124/, License: Creative Commons 0"},
				item{title: "Horse carriage", desc: "Canadian Horse Carriage.wav by vero.marengere, https://freesound.org/s/450325/, License: Attribution NonCommercial 4.0"},
			},
			d,
			0,
			0,
		),
		selected: "",
		stop:     make(chan bool),
	}

	m.list.Title = (defaultTitle)
	m.list.Styles.Title = titleStyle

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// Play or pause the track the cursor is pointing at
		case "enter", " ":
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}

			if i.FilterValue() != m.selected {
				m.selected = i.FilterValue()
				m.list.Title = fmt.Sprintf("⏯ Now playing: %s", m.selected)
				return m, m.playTrack

			} else {
				m.selected = ""
				m.list.Title = (defaultTitle)
				return m, m.stopTrack
			}
		}

	case statusMsg:

		// There was a non-ok status. Update the Model and exit.
		if msg != 0 {
			m.err = errMsg{fmt.Errorf("received non-ok status %d", msg)}
			return m, tea.Quit
		}

	case errMsg:

		// There was an error. Update the Model.
		// Don't exit, otherwise can't see the message.
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:

		// Adjust the window size.
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	if nil != m.err.err {
		s := "\nThere was an error processing the request: \n\n"
		s += m.err.err.Error() + "\n\n"
		return s
	}
	return docStyle.Render(m.list.View())
}

func (m Model) playTrack() tea.Msg {
	track := m.selected

	track = snakeCase(track)

	fileName := fmt.Sprintf("assets/%s.wav", track)

	f, err := folder.Open(fileName)
	if err != nil {
		return errMsg{fmt.Errorf("failed to open file %s: %s", fileName, err)}
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return errMsg{fmt.Errorf("failed to decode file %s: %s", fileName, err)}
	}
	defer streamer.Close()

	if err = speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Second/10),
	); nil != err {
		return errMsg{fmt.Errorf("failed to initialize speaker: %s", err)}
	}

	// Loop until further notice.
	loop := beep.Loop(-1, streamer)

	done := make(chan bool)
	speaker.Play(beep.Seq(loop, beep.Callback(func() {
		done <- true
	})))
	defer speaker.Close()

	// Keep looping until we get a stop signal
	// or the audio ends for some reason.
	select {
	case <-done:
		return statusMsg(0)
	case <-m.stop:
		return statusMsg(0)
	}
}

func (m Model) stopTrack() tea.Msg {
	m.stop <- true
	return statusMsg(0)
}

func snakeCase(s string) string {
	return strings.ToLower(
		strings.ReplaceAll(
			s, " ", "-",
		),
	)
}

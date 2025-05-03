package app

import (
	"embed"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

//go:embed assets/*
var folder embed.FS

type Model struct {
	choices  []string  // white noise options
	cursor   int       // which choice our cursor is pointing at
	selected int       // which choice is selected
	stop     chan bool // a way to stop the running track
	err      errMsg    // any errors
}

func InitialModel() Model {
	return Model{
		choices: []string{

			/*
				forest summer Roond 005 200619_0186.wav by klankbeeld
				-- https://freesound.org/s/524238/ -- License: Attribution 4.0
			*/
			"Forest",

			/*
				Empty train moving slowly (recorded inside passenger car) by avakas
				-- https://freesound.org/s/197124/ -- License: Creative Commons 0
			*/
			"Train car",

			/*
				Canadian Horse Carriage.wav by vero.marengere
				-- https://freesound.org/s/450325/ -- License: Attribution NonCommercial 4.0
			*/
			"Horse carriage",
		},
		selected: -1,
		stop:     make(chan bool),
	}
}

func (m Model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// Play or pause the track the cursor is pointing at
		case "enter", " ":
			if m.selected != m.cursor {
				m.selected = m.cursor
				return m, m.playTrack
			} else {
				m.selected = -1
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

		// There was an error. Update the Model and exit.
		m.err = msg
		return m, tea.Quit
	}

	// Return the updated Model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m Model) View() string {

	if nil != m.err.err {
		s := "\nThere was an error processing the request: \n\n"
		s += m.err.err.Error() + "\n\n"
		return s
	}

	s := "What sound do you want to hear?\n\n"

	for i, choice := range m.choices {

		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		checked := " " // not selected
		if i == m.selected {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

func (m Model) playTrack() tea.Msg {
	track := m.choices[m.selected]

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

type statusMsg int

type errMsg struct{ err error }

func snakeCase(s string) string {
	return strings.ToLower(
		strings.ReplaceAll(
			s, " ", "-",
		),
	)
}

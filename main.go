package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

// https://freesound.org/
// https://github.com/charmbracelet/bubbletea/tree/main/tutorials/basics
func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type model struct {
	choices  []string  // white noise options
	cursor   int       // which choice our cursor is pointing at
	selected int       // which choice is selected
	stop     chan bool // a way to stop the running track
}

func initialModel() model {
	return model{
		choices:  []string{"Rain", "Rain", "Rain"},
		selected: -1,
		stop:     make(chan bool),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {

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

func (m model) playTrack() tea.Msg {
	track := m.choices[m.selected]
	track = strings.ToLower(track)

	fileName := fmt.Sprintf("%s.wav", track)

	f, err := os.Open(fileName)
	if err != nil {
		return errMsg{fmt.Errorf("failed to open file %s: %s", fileName, err)}
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return errMsg{fmt.Errorf("failed to decode file %s: %s", fileName, err)}
	}
	defer streamer.Close()

	speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Second/10),
	)

	loop := beep.Loop(-1, streamer)

	done := make(chan bool)
	speaker.Play(beep.Seq(loop, beep.Callback(func() {
		done <- true
	})))
	defer speaker.Close()

	select {
	case <-done:
		return status(0)
	case <-m.stop:
		return status(0)
	}
}

func (m model) stopTrack() tea.Msg {
	m.stop <- true

	return status(0)
}

type status int

type errMsg struct{ err error }

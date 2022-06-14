package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(lipgloss.Color("#89b4fa"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#fab387"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	config            = GlobalConfiguration{}
)

type GlobalConfiguration struct {
	Config Config `json:"config"`
}

type Config struct {
	DataLocation string `json:"dataLocation"`
}

type Data struct {
	List []item `json:"list"`
}

type item struct {
	Desc string `json:"Desc"`
	Done bool   `json:"Done"`
}

func (i item) FilterValue() string { return i.Desc }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	Done := "[ ]"

	if i.Done {
		Done = "[x]"
	}

	str := fmt.Sprintf("%s %v", Done, i.Desc)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprintf(w, fn(str))
}

type model struct {
	list        list.Model
	textInput   textinput.Model
	typing      bool
	editing     bool
	shouldClean bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Save() {
	data := Data{}

	for _, i := range m.list.Items() {
		obj := i.(item)
		data.List = append(data.List, item{Desc: obj.Desc, Done: obj.Done})
	}

	content, err := json.Marshal(&data)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(config.Config.DataLocation)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	_, err = f.WriteString(string(content))

	if err != nil {
		panic(err)
	}
}

func (m model) IsTyping() bool  { return m.typing }
func (m model) IsEditing() bool { return m.editing }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {

		case "enter":
			if !m.IsTyping() {
				i, ok := m.list.SelectedItem().(item)
				if ok {
					i.Done = !i.Done

					newLocation := len(m.list.Items())
					if !i.Done {
						newLocation = 0
					}

					m.list.RemoveItem(m.list.Index())
					m.list.InsertItem(newLocation, i)
					m.Save()

				}
			} else {
				if !m.IsEditing() {
					m.list.InsertItem(0, item{Desc: m.textInput.Value(), Done: false})
					m.Save()
				} else {
					i, ok := m.list.SelectedItem().(item)
					if ok {
						i.Desc = m.textInput.Value()
						m.list.SetItem(m.list.Index(), i)
						m.Save()
					}
				}

				m.textInput.SetValue("")
				m.typing = false
			}

		case "i":
			if !m.IsTyping() {
				m.typing = true
				m.shouldClean = true
			}

		case "d":
			if !m.IsTyping() {
				m.list.RemoveItem(m.list.Index())
				m.Save()
			}

		case "c":
			if !m.IsTyping() {
				m.textInput.SetValue(m.list.SelectedItem().(item).Desc)
				m.editing = true
				m.typing = true
			}
		}

		if m.IsTyping() {
			var cmd tea.Cmd
			m.textInput.Focus()
			m.textInput, cmd = m.textInput.Update(msg)

			if m.shouldClean {
				m.textInput.SetValue("")
				m.shouldClean = false
			}

			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.typing {
		textinput.Blink()
		return fmt.Sprintf("Enter new task:\n%s", m.textInput.View())
	}

	return "\n" + m.list.View()
}

func main() {

	const defaultWidth = 20

	LoadConfig()
	items := LoadData()

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "ToGo - Tasks"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	text := textinput.New()
	text.Placeholder = "buy some milk"

	m := model{list: l, textInput: text}

	if err := tea.NewProgram(m, tea.WithAltScreen()).Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func LoadData() []list.Item {
	content, err := ioutil.ReadFile(config.Config.DataLocation)

	if err != nil {
		return []list.Item{}
	}

	contentBytes := []byte(content)

	var data Data

	if err := json.Unmarshal(contentBytes, &data); err != nil {
		panic(err)
	}

	items := []list.Item{}
	for _, i := range data.List {
		items = append(items, i)

	}

	return items
}

func LoadConfig() {
	cfg := GlobalConfiguration{Config{DataLocation: "data.json"}}

	home := os.Getenv("HOME")
	content, _ := ioutil.ReadFile(home + "/.config/togo/config.json")
	contentBytes := []byte(content)

	json.Unmarshal(contentBytes, &cfg)

	config = cfg
}

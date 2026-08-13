package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/logs"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/version"
)

type view int

const (
	viewDashboard view = iota
	viewAccounts
	viewModels
	viewTest
	viewLogs
	viewCount
)

type formKind int

const (
	formNone formKind = iota
	formEditKey
	formAddAccount
)

type providerInfo struct {
	name     string
	enabled  bool
	accounts int
	models   int
}

type formField struct {
	label string
	input textinput.Model
}

type Model struct {
	runtime    *Runtime
	configPath string

	view view

	loaded  bool
	loadErr error

	addr      string
	providers []providerInfo
	models    []model.Model
	accounts  []accounts.Summary
	logs      []logs.Entry

	running bool
	authing string

	accountCursor int
	modelCursor   int

	statusMsg string

	width  int
	height int

	formActive  bool
	formKind    formKind
	formTitle   string
	formAccount string
	formFields  []formField
	formCursor  int

	testModelIdx int
	testPrompt   textinput.Model
	testResult   string
	testErr      error
	testSending  bool
}

type loadDoneMsg struct{ err error }

type toggleMsg struct {
	running bool
	err     error
}

type authDoneMsg struct {
	account string
	err     error
}

type editDoneMsg struct{ err error }

type formDoneMsg struct{ err error }

type testDoneMsg struct {
	resp *model.ChatResponse
	err  error
}

type tickMsg struct{}

// New creates the root TUI model bound to a config path.
func New(configPath string) *Model {
	return &Model{
		runtime:    NewRuntime(configPath),
		configPath: configPath,
		view:       viewDashboard,
		testPrompt: newTextInput("Enter a message...", textinput.EchoNormal),
	}
}

// Run starts the interactive TUI.
func Run(configPath string) error {
	p := tea.NewProgram(New(configPath), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newTextInput(placeholder string, echo textinput.EchoMode) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.EchoMode = echo
	ti.CharLimit = 400
	ti.Width = 60
	return ti
}

func (m Model) Init() tea.Cmd {
	return m.load()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.formActive {
			return m.updateForm(msg)
		}
		return m.handleKey(msg)

	case loadDoneMsg:
		return m.handleLoaded(msg)

	case toggleMsg:
		m.running = msg.running
		switch {
		case msg.err != nil:
			m.statusMsg = msg.err.Error()
		case msg.running:
			m.statusMsg = "proxy listening on http://" + m.addr
		default:
			m.statusMsg = "proxy stopped"
		}
		return m, nil

	case authDoneMsg:
		m.authing = ""
		if msg.err != nil {
			m.statusMsg = "auth failed: " + msg.err.Error()
		} else {
			m.statusMsg = "authorized " + msg.account
		}
		m.refresh()
		return m, nil

	case editDoneMsg:
		if msg.err != nil {
			m.statusMsg = "editor error: " + msg.err.Error()
			return m, nil
		}
		if m.running {
			m.statusMsg = "config edited; stop and reload to apply"
		} else {
			m.statusMsg = "config edited"
			return m, m.load()
		}
		return m, nil

	case formDoneMsg:
		m.closeForm()
		if msg.err != nil {
			m.statusMsg = "config error: " + msg.err.Error()
			return m, nil
		}
		if m.running {
			m.statusMsg = "config saved; stop and reload to apply"
		} else {
			m.statusMsg = "config saved"
			return m, m.load()
		}
		return m, nil

	case testDoneMsg:
		m.testSending = false
		if msg.err != nil {
			m.testErr = msg.err
			m.testResult = ""
		} else {
			m.testErr = nil
			m.testResult = msg.resp.Content
		}
		return m, nil

	case tickMsg:
		m.refresh()
		return m, m.tick()
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, m.quit()
	}

	if key == "tab" {
		m.switchView((m.view + 1) % viewCount)
		return m, nil
	}

	if m.view == viewTest {
		return m.handleTestKey(msg)
	}

	switch key {
	case "q":
		return m, m.quit()

	case "s":
		if m.running {
			return m, func() tea.Msg {
				err := m.runtime.Stop()
				return toggleMsg{running: false, err: err}
			}
		}
		if !m.loaded {
			m.statusMsg = "config not loaded"
			return m, nil
		}
		return m, func() tea.Msg {
			err := m.runtime.Start()
			return toggleMsg{running: true, err: err}
		}

	case "r":
		if m.running {
			m.statusMsg = "stop the proxy before reloading config"
			return m, nil
		}
		m.statusMsg = "reloading config..."
		return m, m.load()

	case "e":
		return m, m.editConfig()

	case "a":
		if m.view == viewAccounts && m.loaded {
			if m.accountCursor >= len(m.accounts) {
				return m, nil
			}
			a := m.accounts[m.accountCursor]
			if a.HasAPIKey {
				m.statusMsg = fmt.Sprintf("%s/%s uses an API key; no browser login needed", a.Provider, a.Account)
				return m, nil
			}
			m.authing = a.Provider + "/" + a.Account
			m.statusMsg = "Opening browser to authorize " + m.authing + "..."
			return m, func() tea.Msg {
				err := m.runtime.Auth(a.Provider, a.Account)
				return authDoneMsg{account: a.Provider + "/" + a.Account, err: err}
			}
		}

	case "k":
		if m.view == viewAccounts && m.loaded {
			if m.accountCursor >= len(m.accounts) {
				return m, nil
			}
			a := m.accounts[m.accountCursor]
			if !a.HasAPIKey {
				m.statusMsg = "selected account is OAuth; k only applies to API-key accounts"
				return m, nil
			}
			m.openKeyForm(a.Account)
			return m, nil
		}

	case "n":
		if m.view == viewAccounts && m.loaded {
			m.openAddAccountForm()
			return m, nil
		}

	case "up":
		m.moveCursor(-1)
		return m, nil
	case "down":
		m.moveCursor(1)
		return m, nil
	}
	return m, nil
}

func (m Model) handleTestKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveTestModel(-1)
		return m, nil
	case "down":
		m.moveTestModel(1)
		return m, nil
	case "enter":
		if m.testSending {
			return m, nil
		}
		prompt := strings.TrimSpace(m.testPrompt.Value())
		if prompt == "" {
			return m, nil
		}
		m.testSending = true
		m.testErr = nil
		return m, m.sendTest(prompt)
	default:
		var cmd tea.Cmd
		m.testPrompt, cmd = m.testPrompt.Update(msg)
		return m, cmd
	}
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.closeForm()
		return m, nil
	case "enter":
		if m.formCursor == len(m.formFields)-1 {
			return m, m.submitForm()
		}
		m.formCursor++
		m.focusFormField()
		return m, nil
	case "tab", "down":
		m.formCursor = (m.formCursor + 1) % len(m.formFields)
		m.focusFormField()
		return m, nil
	case "up":
		m.formCursor = (m.formCursor - 1 + len(m.formFields)) % len(m.formFields)
		m.focusFormField()
		return m, nil
	}

	var cmd tea.Cmd
	f := m.formFields[m.formCursor]
	f.input, cmd = f.input.Update(keyMsg)
	m.formFields[m.formCursor] = f
	return m, cmd
}

func (m Model) submitForm() tea.Cmd {
	kind := m.formKind
	account := m.formAccount
	var values []string
	for _, f := range m.formFields {
		values = append(values, strings.TrimSpace(f.input.Value()))
	}
	return func() tea.Msg {
		var err error
		switch kind {
		case formEditKey:
			if values[0] == "" {
				err = fmt.Errorf("api key is empty")
			} else {
				err = m.runtime.SetGeminiAPIKey(account, values[0])
			}
		case formAddAccount:
			if values[0] == "" {
				err = fmt.Errorf("account name is empty")
			} else if values[1] == "" {
				err = fmt.Errorf("api key is empty")
			} else {
				err = m.runtime.AddGeminiAccount(values[0], values[1])
			}
		}
		return formDoneMsg{err: err}
	}
}

func (m *Model) openKeyForm(account string) {
	ti := newTextInput("enter new API key", textinput.EchoPassword)
	ti.EchoCharacter = '•'
	ti.Focus()
	m.formKind = formEditKey
	m.formTitle = "Set Gemini API key — " + account
	m.formAccount = account
	m.formFields = []formField{{label: "API key", input: ti}}
	m.formCursor = 0
	m.formActive = true
}

func (m *Model) openAddAccountForm() {
	name := newTextInput("e.g. gemini-backup", textinput.EchoNormal)
	key := newTextInput("AIza...", textinput.EchoPassword)
	key.EchoCharacter = '•'
	name.Focus()
	m.formKind = formAddAccount
	m.formTitle = "Add Gemini account"
	m.formFields = []formField{
		{label: "Name", input: name},
		{label: "API key", input: key},
	}
	m.formCursor = 0
	m.formActive = true
}

func (m *Model) focusFormField() {
	for i := range m.formFields {
		if i == m.formCursor {
			m.formFields[i].input.Focus()
		} else {
			m.formFields[i].input.Blur()
		}
	}
}

func (m *Model) closeForm() {
	m.formActive = false
	m.formKind = formNone
	m.formTitle = ""
	m.formAccount = ""
	m.formFields = nil
	m.formCursor = 0
}

func (m *Model) switchView(v view) {
	m.view = v
	m.accountCursor = 0
	m.modelCursor = 0
	if v == viewTest {
		m.testPrompt.Focus()
	} else {
		m.testPrompt.Blur()
	}
}

func (m Model) quit() tea.Cmd {
	if m.running {
		m.statusMsg = "stopping..."
		return func() tea.Msg {
			_ = m.runtime.Stop()
			return tea.Quit()
		}
	}
	return tea.Quit
}

func (m *Model) moveCursor(delta int) {
	switch m.view {
	case viewAccounts:
		n := len(m.accounts)
		if n == 0 {
			m.accountCursor = 0
			return
		}
		m.accountCursor = (m.accountCursor + delta + n) % n
	case viewModels:
		n := len(m.models)
		if n == 0 {
			m.modelCursor = 0
			return
		}
		m.modelCursor = (m.modelCursor + delta + n) % n
	}
}

func (m *Model) moveTestModel(delta int) {
	n := len(m.models)
	if n == 0 {
		m.testModelIdx = 0
		return
	}
	m.testModelIdx = (m.testModelIdx + delta + n) % n
}

func (m Model) load() tea.Cmd {
	return func() tea.Msg {
		return loadDoneMsg{err: m.runtime.Load()}
	}
}

func (m Model) sendTest(prompt string) tea.Cmd {
	modelID := ""
	if m.testModelIdx < len(m.models) {
		modelID = m.models[m.testModelIdx].ID
	}
	maxTokens := m.runtime.DefaultMaxTokens()
	return func() tea.Msg {
		resp, err := m.runtime.Chat(context.Background(), &model.ChatRequest{
			Model:     modelID,
			Messages:  []model.Message{{Role: model.RoleUser, Content: prompt}},
			MaxTokens: maxTokens,
		})
		return testDoneMsg{resp: resp, err: err}
	}
}

func (m Model) editConfig() tea.Cmd {
	return tea.ExecProcess(editorCommand(m.configPath), func(err error) tea.Msg {
		return editDoneMsg{err: err}
	})
}

func editorCommand(path string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	return exec.Command("sh", "-c", editor+" "+shellQuote(path))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (m Model) handleLoaded(msg loadDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.loaded = false
		m.loadErr = msg.err
		m.statusMsg = "load failed: " + msg.err.Error()
		return m, nil
	}
	m.loadErr = nil
	m.loaded = true
	m.statusMsg = "config loaded"
	m.refresh()
	return m, m.tick()
}

func (m *Model) refresh() {
	cfg := m.runtime.Config()
	if cfg == nil {
		return
	}
	m.addr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	m.providers = buildProviders(cfg)
	m.models = m.runtime.Models()
	m.accounts = m.runtime.AccountSummaries()
	m.logs = m.runtime.LogEntries()
	m.running = m.runtime.Running()
	if m.accountCursor >= len(m.accounts) {
		m.accountCursor = 0
	}
	if m.modelCursor >= len(m.models) {
		m.modelCursor = 0
	}
	if m.testModelIdx >= len(m.models) {
		m.testModelIdx = 0
	}
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func buildProviders(cfg *config.Config) []providerInfo {
	var out []providerInfo
	for name, p := range cfg.Providers {
		out = append(out, providerInfo{
			name:     name,
			enabled:  p.Enabled,
			accounts: len(p.Accounts),
			models:   len(p.Models),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (m Model) View() string {
	if m.loadErr != nil {
		return m.viewError()
	}
	if !m.loaded {
		return m.viewLoading()
	}

	if m.formActive {
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Unified AI Proxy "+version.Version),
			m.viewForm(),
			m.renderStatus(),
			footerStyle.Render("enter submit · esc cancel · up/down switch field"),
		)
	}

	var body string
	switch m.view {
	case viewDashboard:
		body = m.viewDashboard()
	case viewAccounts:
		body = m.viewAccounts()
	case viewModels:
		body = m.viewModels()
	case viewTest:
		body = m.viewTest()
	case viewLogs:
		body = m.viewLogs()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Unified AI Proxy "+version.Version),
		body,
		m.renderStatus(),
		footerStyle.Render(m.renderFooter()),
	)
}

func (m Model) viewLoading() string {
	return titleStyle.Render("Unified AI Proxy") +
		"\n" + dimStyle.Render("Loading "+m.configPath+"...")
}

func (m Model) viewError() string {
	return titleStyle.Render("Unified AI Proxy") +
		"\n" + badStyle.Render("Failed to load "+m.configPath) +
		"\n" + m.loadErr.Error() +
		"\n\n" + dimStyle.Render("Press r to retry, ctrl+c to quit")
}

func (m Model) viewDashboard() string {
	status := okStyle.Render("running")
	if !m.running {
		status = dimStyle.Render("stopped")
	}

	lines := []string{
		sectionStyle.Render("Server"),
		labelStyle.Render("Config:  ") + valueStyle.Render(m.configPath),
		labelStyle.Render("Address: ") + valueStyle.Render(m.addr),
		labelStyle.Render("Status:  ") + status,
	}

	var prov []string
	prov = append(prov, sectionStyle.Render("Providers"))
	prov = append(prov, tableHeaderStyle.Render(fmt.Sprintf("%-20s %-9s %-10s %-8s", "name", "enabled", "accounts", "models")))
	for _, p := range m.providers {
		enabled := "no"
		if p.enabled {
			enabled = "yes"
		}
		prov = append(prov, fmt.Sprintf("%-20s %-9s %-10d %-8d", p.name, enabled, p.accounts, p.models))
	}

	lines = append(lines, strings.Join(prov, "\n"))
	return strings.Join(lines, "\n\n")
}

func (m Model) viewAccounts() string {
	if len(m.accounts) == 0 {
		return sectionStyle.Render("Accounts") + "\n" + dimStyle.Render("no accounts configured")
	}

	lines := []string{
		sectionStyle.Render("Accounts"),
		tableHeaderStyle.Render(fmt.Sprintf("%-16s %-14s %-34s %-26s", "provider", "account", "status", "expires")),
	}
	for i, a := range m.accounts {
		row := fmt.Sprintf("%-16s %-14s %-34s %-26s", a.Provider, a.Account, a.Status, a.Expiry)
		if i == m.accountCursor {
			row = selectedStyle.Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewModels() string {
	if len(m.models) == 0 {
		return sectionStyle.Render("Models") + "\n" + dimStyle.Render("no models configured")
	}

	lines := []string{
		sectionStyle.Render("Models"),
		tableHeaderStyle.Render(fmt.Sprintf("%-40s %-16s", "model", "provider")),
	}
	for i, md := range m.models {
		row := fmt.Sprintf("%-40s %-16s", md.ID, md.Provider)
		if i == m.modelCursor {
			row = selectedStyle.Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewTest() string {
	lines := []string{sectionStyle.Render("Test Request")}

	if len(m.models) == 0 {
		lines = append(lines, dimStyle.Render("no models configured"))
	} else {
		lines = append(lines, labelStyle.Render("Model (up/down to select):"))
		for i, md := range m.models {
			row := "  " + md.ID + " (" + md.Provider + ")"
			if i == m.testModelIdx {
				row = selectedStyle.Render(row)
			}
			lines = append(lines, row)
		}
	}

	lines = append(lines, "", labelStyle.Render("Prompt:"))
	lines = append(lines, m.testPrompt.View())

	switch {
	case m.testSending:
		lines = append(lines, "", warnStyle.Render("Waiting for response..."))
	case m.testErr != nil:
		lines = append(lines, "", badStyle.Render("Error: "+m.testErr.Error()))
	case m.testResult != "":
		lines = append(lines, "", labelStyle.Render("Response:"))
		lines = append(lines, valueStyle.Render(m.testResult))
	}

	return strings.Join(lines, "\n")
}

func (m Model) viewLogs() string {
	lines := []string{sectionStyle.Render("Request Logs")}

	if len(m.logs) == 0 {
		lines = append(lines, dimStyle.Render("no requests logged yet"))
		return strings.Join(lines, "\n")
	}

	entries := m.logs
	if max := m.height - 8; max > 0 && len(entries) > max {
		entries = entries[len(entries)-max:]
	}

	lines = append(lines, tableHeaderStyle.Render(fmt.Sprintf("%-8s %-6s %-32s %-6s %s", "time", "method", "path", "status", "latency")))

	for _, e := range entries {
		if e.Message != "" {
			lines = append(lines, dimStyle.Render(e.Time.Format("15:04:05"))+" "+e.Message)
			continue
		}
		status := fmt.Sprintf("%d", e.Status)
		switch {
		case e.Status >= 400:
			status = badStyle.Render(status)
		case e.Status >= 300:
			status = warnStyle.Render(status)
		}
		lines = append(lines, fmt.Sprintf("%-8s %-6s %-32s %-6s %s",
			e.Time.Format("15:04:05"), e.Method, e.Path, status, e.Latency.Round(time.Millisecond)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewForm() string {
	lines := []string{sectionStyle.Render(m.formTitle)}
	for i, f := range m.formFields {
		prefix := "  "
		if i == m.formCursor {
			prefix = "> "
		}
		lines = append(lines, prefix+f.label+": "+f.input.View())
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderStatus() string {
	if m.statusMsg == "" {
		return ""
	}
	if m.authing != "" {
		return statusStyle.Render(m.statusMsg + " (waiting for browser callback)")
	}
	return statusStyle.Render(m.statusMsg)
}

func (m Model) renderFooter() string {
	switch m.view {
	case viewTest:
		return "type prompt · enter send · up/down select model · tab switch · ctrl+c quit"
	case viewAccounts:
		return "up/down select · a auth OAuth · k set Gemini API key · n add Gemini account · e edit config · tab switch · q quit"
	case viewModels:
		return "up/down select model · s start/stop · r reload · e edit config · tab switch · q quit"
	default:
		return "s start/stop · r reload · e edit config · tab switch · q quit"
	}
}

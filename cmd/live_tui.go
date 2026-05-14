package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type liveSnapshotMsg struct {
	snapshot *liveSnapshot
	err      error
}

type refreshTickMsg struct{}

type livePanel int

const (
	livePanelSystem livePanel = iota
	livePanelBattery
	livePanelSolar
	livePanelInverter
	livePanelGrid
)

type liveTUIModel struct {
	token          string
	siteID         int
	interval       time.Duration
	loading        bool
	snapshot       *liveSnapshot
	batteryTrend   []float64
	solarTrend     []float64
	loadTrend      []float64
	err            error
	updatedAt      time.Time
	width          int
	height         int
	selectedPanel  livePanel
	compact        bool
	titleStyle     lipgloss.Style
	subtitleStyle  lipgloss.Style
	headerBoxStyle lipgloss.Style
	panelStyle     lipgloss.Style
	activePanel    lipgloss.Style
	panelTitle     lipgloss.Style
	labelStyle     lipgloss.Style
	valueStyle     lipgloss.Style
	okStyle        lipgloss.Style
	warnStyle      lipgloss.Style
	errorStyle     lipgloss.Style
	mutedStyle     lipgloss.Style
	footerStyle    lipgloss.Style
	badgeStyle     lipgloss.Style
	detailStyle    lipgloss.Style
}

func newLiveTUIModel(token string, siteID int, interval time.Duration) liveTUIModel {
	borderColor := lipgloss.Color("238")
	return liveTUIModel{
		token:         token,
		siteID:        siteID,
		interval:      interval,
		loading:       true,
		selectedPanel: livePanelSystem,
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117")),
		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")),
		headerBoxStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 2),
		panelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1),
		activePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("75")).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("111")),
		labelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		valueStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),
		okStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true),
		warnStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("204")).
			Bold(true),
		mutedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		footerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		badgeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1),
		detailStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
	}
}

func (m liveTUIModel) Init() tea.Cmd {
	return m.fetchSnapshotCmd()
}

func (m liveTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.selectedPanel = (m.selectedPanel + 1) % 5
			return m, nil
		case "shift+tab", "left", "h":
			m.selectedPanel = (m.selectedPanel + 4) % 5
			return m, nil
		case "up", "k":
			if m.selectedPanel == livePanelGrid {
				m.selectedPanel = livePanelSolar
			} else if m.selectedPanel > livePanelSystem {
				m.selectedPanel--
			}
			return m, nil
		case "down", "j":
			if m.selectedPanel < livePanelInverter {
				m.selectedPanel++
			} else if m.selectedPanel == livePanelInverter {
				m.selectedPanel = livePanelGrid
			}
			return m, nil
		case "c":
			m.compact = !m.compact
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case liveSnapshotMsg:
		m.loading = false
		m.err = msg.err
		m.updatedAt = time.Now().Local()
		if msg.err == nil {
			m.snapshot = msg.snapshot
			m.recordTrends(msg.snapshot)
		}
		return m, tea.Tick(m.interval, func(time.Time) tea.Msg { return refreshTickMsg{} })
	case refreshTickMsg:
		m.loading = true
		return m, m.fetchSnapshotCmd()
	}

	return m, nil
}

func (m liveTUIModel) View() string {
	header := m.renderHeader()

	if m.loading && m.snapshot == nil && m.err == nil {
		body := m.wrapCentered(m.panel("Loading", "Fetching live snapshot...", false, false))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", m.footer())
	}

	if m.err != nil && m.snapshot == nil {
		body := m.wrapCentered(m.panel("Snapshot Error", m.errorStyle.Render(m.err.Error()), false, false))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", m.footer())
	}

	content := ""
	if m.snapshot != nil {
		content = m.renderDashboard(m.snapshot)
	}

	parts := []string{header, "", content}
	if m.err != nil {
		parts = append(parts, "", m.warnStyle.Render("Last refresh failed: "+m.err.Error()))
	}
	parts = append(parts, "", m.footer())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m liveTUIModel) fetchSnapshotCmd() tea.Cmd {
	token := m.token
	siteID := m.siteID
	return func() tea.Msg {
		snapshot, err := fetchLiveSnapshot(token, siteID)
		return liveSnapshotMsg{snapshot: snapshot, err: err}
	}
}

func (m liveTUIModel) renderHeader() string {
	title := m.titleStyle.Render(fmt.Sprintf("victronctl live  site %d", m.siteID))
	status := m.subtitleStyle.Render(fmt.Sprintf("refresh %s", m.interval))
	if m.snapshot != nil {
		if m.width > 0 && m.width < 90 {
			status = m.subtitleStyle.Render(fmt.Sprintf("obs %s  |  %s", m.snapshot.ObservedAt.Format("15:04:05"), m.interval))
		} else {
			status = m.subtitleStyle.Render(fmt.Sprintf("observed %s  |  refresh %s", formatDisplayTime(m.snapshot.ObservedAt), m.interval))
		}
	}

	lines := []string{title, status}
	if m.snapshot != nil {
		lines = append(lines, m.renderHealthBar(m.snapshot))
	}
	content := lipgloss.JoinVertical(lipgloss.Center, lines...)
	style := m.headerBoxStyle
	if m.width >= 100 {
		style = style.Width(m.headerContentWidth())
	}
	return m.wrapCentered(style.Render(content))
}

func (m liveTUIModel) renderDashboard(snapshot *liveSnapshot) string {
	compact := m.effectiveCompact()
	system := m.renderSystemPanel(snapshot)
	battery := m.renderBatteryPanel(snapshot)
	solar := m.renderSolarPanel(snapshot)
	inverter := m.renderInverterPanel(snapshot)
	grid := m.renderGridPanel(snapshot)
	warnings := m.renderWarningsPanel(snapshot)

	var dashboard string
	if m.width >= 100 {
		outerWidth := m.dashboardContentWidth()
		gapWidth := 6
		panelWidth := maxInt(18, (outerWidth-gapWidth)/4)
		panels := equalizePanelSizes([]string{
			m.panelWidth(system, panelWidth),
			m.panelWidth(battery, panelWidth),
			m.panelWidth(solar, panelWidth),
			m.panelWidth(inverter, panelWidth),
		}, panelWidth)
		middleContent := lipgloss.JoinHorizontal(
			lipgloss.Top,
			panels[0],
			"  ",
			panels[1],
			"  ",
			panels[2],
			"  ",
			panels[3],
		)
		middle := lipgloss.PlaceHorizontal(outerWidth, lipgloss.Center, middleContent)
		bottom := m.renderBottomSection(grid, warnings)
		dashboard = m.wrapCentered(lipgloss.JoinVertical(lipgloss.Left, middle, "", bottom))
	} else if m.width >= 68 || m.width == 0 {
		panelWidth := maxInt(28, (maxInt(m.width, 80)-6)/2)
		row1 := lipgloss.JoinHorizontal(lipgloss.Top,
			m.panelWidth(system, panelWidth),
			"  ",
			m.panelWidth(battery, panelWidth),
		)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top,
			m.panelWidth(solar, panelWidth),
			"  ",
			m.panelWidth(inverter, panelWidth),
		)
		row3 := m.panelWidth(grid, maxInt(m.width, 80))
		dashboard = lipgloss.JoinVertical(lipgloss.Left, row1, "", row2, "", row3)
	} else {
		dashboard = lipgloss.JoinVertical(lipgloss.Left, system, "", battery, "", solar, "", inverter, "", grid)
	}

	if compact || !m.canShowDetail() {
		return dashboard
	}

	detail := m.renderDetailPanel(snapshot)
	return lipgloss.JoinVertical(lipgloss.Left, dashboard, "", detail)
}

func (m liveTUIModel) renderBatteryPanel(snapshot *liveSnapshot) string {
	lines := []string{m.metric("SOC", snapshot.BatterySOC, m.emphasizePercent(snapshot.BatterySOC))}
	if m.effectiveCompact() {
		lines = append(lines,
			m.metric("Volt", snapshot.BatteryVoltage, m.valueStyle),
			m.metric("Trend", compactTrend(m.batteryTrend), m.trendStyle(m.batteryTrend)),
		)
	} else {
		lines = append(lines,
			m.metric("Voltage", snapshot.BatteryVoltage, m.valueStyle),
			m.metric("Current", snapshot.BatteryCurrent, m.valueStyle),
			m.metric("Data age", snapshot.DataAge, m.dataAgeStyle(snapshot)),
			m.metric("Trend", m.trendText(m.batteryTrend, "V"), m.trendStyle(m.batteryTrend)),
			m.metric("Spark", sparkline(m.batteryTrend), m.trendStyle(m.batteryTrend)),
		)
	}
	return m.panel("Battery", strings.Join(lines, "\n"), m.selectedPanel == livePanelBattery, false)
}

func (m liveTUIModel) renderSystemPanel(snapshot *liveSnapshot) string {
	panelObserved := snapshot.ObservedAt.Format("15:04:05")
	if m.effectiveCompact() {
		panelObserved = snapshot.ObservedAt.Format("15:04:05")
	}
	lines := []string{m.metric("Health", m.overallHealth(snapshot), m.healthStyle(snapshot))}
	if m.effectiveCompact() {
		lines = append(lines,
			m.metric("Grid", m.gridStatus(snapshot), m.gridStateStyle(m.gridStatus(snapshot))),
			m.metric("Age", snapshot.DataAge, m.dataAgeStyle(snapshot)),
		)
	} else {
		lines = append(lines,
			m.metric("Seen", panelObserved, m.valueStyle),
			m.metric("Age", snapshot.DataAge, m.dataAgeStyle(snapshot)),
			m.metric("Grid", m.gridStatus(snapshot), m.gridStateStyle(m.gridStatus(snapshot))),
			m.metric("Warn", fmt.Sprintf("%d", len(snapshot.Warnings)), m.warnStyle),
			m.metric("Mode", m.viewMode(), m.mutedStyle),
		)
	}
	return m.panel("System", strings.Join(lines, "\n"), m.selectedPanel == livePanelSystem, false)
}

func (m liveTUIModel) renderSolarPanel(snapshot *liveSnapshot) string {
	lines := []string{m.metric("Power", snapshot.SolarPower, m.valueStyle)}
	if m.effectiveCompact() {
		lines = append(lines,
			m.metric("State", snapshot.SolarState, m.solarStateStyle(snapshot.SolarState)),
			m.metric("Trend", compactTrend(m.solarTrend), m.trendStyle(m.solarTrend)),
		)
	} else {
		lines = append(lines,
			m.metric("State", snapshot.SolarState, m.solarStateStyle(snapshot.SolarState)),
			m.metric("Yield today", snapshot.SolarYieldToday, m.valueStyle),
			m.metric("Freshness", snapshot.DataAge, m.dataAgeStyle(snapshot)),
			m.metric("Trend", m.trendText(m.solarTrend, "W"), m.trendStyle(m.solarTrend)),
			m.metric("Spark", sparkline(m.solarTrend), m.trendStyle(m.solarTrend)),
		)
	}
	return m.panel("Solar", strings.Join(lines, "\n"), m.selectedPanel == livePanelSolar, false)
}

func (m liveTUIModel) renderInverterPanel(snapshot *liveSnapshot) string {
	lines := []string{
		m.metric("State", snapshot.InverterState, m.inverterStateStyle(snapshot.InverterState)),
		m.metric("Load", snapshot.LoadPower, m.valueStyle),
	}
	if m.effectiveCompact() {
		lines = append(lines, m.metric("Trend", compactTrend(m.loadTrend), m.trendStyle(m.loadTrend)))
	} else {
		lines = append(lines,
			m.metric("Battery V", snapshot.BatteryVoltage, m.mutedStyle),
			m.metric("Battery I", snapshot.BatteryCurrent, m.mutedStyle),
			m.metric("Trend", m.trendText(m.loadTrend, "W"), m.trendStyle(m.loadTrend)),
			m.metric("Spark", sparkline(m.loadTrend), m.trendStyle(m.loadTrend)),
		)
	}
	return m.panel("Inverter / Load", strings.Join(lines, "\n"), m.selectedPanel == livePanelInverter, false)
}

func (m liveTUIModel) renderGridPanel(snapshot *liveSnapshot) string {
	status := m.gridStatus(snapshot)
	lines := []string{
		m.metric("Import", snapshot.GridImportPower, m.gridStateStyle(snapshot.GridImportPower)),
		m.metric("Status", status, m.gridStateStyle(status)),
	}
	if !m.effectiveCompact() {
		lines = append(lines, m.metric("Export", snapshot.GridExportPower, m.gridStateStyle(snapshot.GridExportPower)))
	}
	if m.width < 100 {
		warningBlock := m.renderWarnings(snapshot)
		if warningBlock != "" && !m.effectiveCompact() {
			lines = append(lines, "", warningBlock)
		}
	}
	if m.width >= 100 {
		return m.subSection("Grid", strings.Join(lines, "\n"))
	}
	return m.panel("Grid", strings.Join(lines, "\n"), m.selectedPanel == livePanelGrid, false)
}

func (m liveTUIModel) renderWarningsPanel(snapshot *liveSnapshot) string {
	warningBlock := m.renderWarnings(snapshot)
	if warningBlock == "" {
		warningBlock = m.okStyle.Render("No active warnings")
	}
	return m.subSection("Warnings", warningBlock)
}

func (m liveTUIModel) renderBottomSection(grid, warnings string) string {
	outerWidth := m.dashboardContentWidth()
	innerWidth := maxInt(30, (outerWidth-6)/2)
	sections := equalizePanelSizes([]string{
		lipgloss.NewStyle().Width(innerWidth).Render(grid),
		lipgloss.NewStyle().Width(innerWidth).Render(warnings),
	}, innerWidth)
	content := lipgloss.JoinHorizontal(lipgloss.Top, sections[0], "  ", sections[1])
	title := m.panelTitle.Render("Grid / Warnings")
	container := m.panelStyle
	if m.selectedPanel == livePanelGrid {
		container = m.activePanel
	}
	container = container.Width(outerWidth)
	return container.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}

func (m liveTUIModel) subSection(title, body string) string {
	style := lipgloss.NewStyle().Padding(0, 0)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, m.panelTitle.Render(title), "", body))
}

func (m liveTUIModel) renderWarnings(snapshot *liveSnapshot) string {
	if len(snapshot.Warnings) == 0 && !snapshot.Stale {
		return m.okStyle.Render("No active warnings")
	}
	var lines []string
	if snapshot.Stale && !hasDetailedStaleWarning(snapshot.Warnings) {
		lines = append(lines, m.warnStyle.Render("- Data is stale"))
	}
	for _, warning := range snapshot.Warnings {
		lines = append(lines, m.warnStyle.Render("- "+warning))
	}
	return strings.Join(lines, "\n")
}

func (m liveTUIModel) renderDetailPanel(snapshot *liveSnapshot) string {
	title := "Detail"
	var lines []string
	switch m.selectedPanel {
	case livePanelBattery:
		title = "Battery Detail"
		lines = []string{
			m.metric("SOC", snapshot.BatterySOC, m.emphasizePercent(snapshot.BatterySOC)),
			m.metric("Voltage", snapshot.BatteryVoltage, m.valueStyle),
			m.metric("Current", snapshot.BatteryCurrent, m.valueStyle),
			m.metric("Direction", m.trendDirection(m.batteryTrend), m.trendStyle(m.batteryTrend)),
			m.metric("Sparkline", sparkline(m.batteryTrend), m.trendStyle(m.batteryTrend)),
		}
	case livePanelSolar:
		title = "Solar Detail"
		lines = []string{
			m.metric("Power", snapshot.SolarPower, m.valueStyle),
			m.metric("State", snapshot.SolarState, m.solarStateStyle(snapshot.SolarState)),
			m.metric("Yield today", snapshot.SolarYieldToday, m.valueStyle),
			m.metric("Direction", m.trendDirection(m.solarTrend), m.trendStyle(m.solarTrend)),
			m.metric("Sparkline", sparkline(m.solarTrend), m.trendStyle(m.solarTrend)),
		}
	case livePanelInverter:
		title = "Inverter / Load Detail"
		lines = []string{
			m.metric("State", snapshot.InverterState, m.inverterStateStyle(snapshot.InverterState)),
			m.metric("Load", snapshot.LoadPower, m.valueStyle),
			m.metric("Direction", m.trendDirection(m.loadTrend), m.trendStyle(m.loadTrend)),
			m.metric("Sparkline", sparkline(m.loadTrend), m.trendStyle(m.loadTrend)),
			m.metric("Battery V", snapshot.BatteryVoltage, m.mutedStyle),
			m.metric("Battery I", snapshot.BatteryCurrent, m.mutedStyle),
		}
	case livePanelSystem:
		title = "System Detail"
		lines = []string{
			m.metric("Health", m.overallHealth(snapshot), m.healthStyle(snapshot)),
			m.metric("Observed", formatDisplayTime(snapshot.ObservedAt), m.valueStyle),
			m.metric("Data age", snapshot.DataAge, m.dataAgeStyle(snapshot)),
			m.metric("Grid", m.gridStatus(snapshot), m.gridStateStyle(m.gridStatus(snapshot))),
			m.metric("View", m.viewMode(), m.mutedStyle),
			m.metric("Warnings", fmt.Sprintf("%d", len(snapshot.Warnings)), m.warnStyle),
		}
	case livePanelGrid:
		title = "Grid / Warning Detail"
		lines = []string{
			m.metric("Import", snapshot.GridImportPower, m.gridStateStyle(snapshot.GridImportPower)),
			m.metric("Export", snapshot.GridExportPower, m.gridStateStyle(snapshot.GridExportPower)),
			m.metric("Status", m.gridStatus(snapshot), m.gridStateStyle(m.gridStatus(snapshot))),
			m.metric("Warnings", fmt.Sprintf("%d", len(snapshot.Warnings)), m.warnStyle),
		}
		if len(snapshot.Warnings) > 0 {
			lines = append(lines, "")
			for _, warning := range snapshot.Warnings {
				lines = append(lines, m.warnStyle.Render("- "+warning))
			}
		}
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.panelTitle.Render(title), "", strings.Join(lines, "\n"))
	return m.detailStyle.Render(content)
}

func (m liveTUIModel) renderHealthBar(snapshot *liveSnapshot) string {
	parts := []string{
		m.badge("HEALTH", m.overallHealth(snapshot), m.healthBadgeColor(snapshot)),
		m.badge("INVERTER", snapshot.InverterState, m.inverterBadgeColor(snapshot.InverterState)),
		m.badge("GRID", m.gridStatus(snapshot), m.gridBadgeColor(m.gridStatus(snapshot))),
		m.badge("AGE", snapshot.DataAge, m.ageBadgeColor(snapshot)),
		m.badge("VIEW", m.viewMode(), lipgloss.Color("60")),
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	if m.width > 0 {
		return lipgloss.PlaceHorizontal(m.width-6, lipgloss.Center, bar)
	}
	return bar
}

func (m liveTUIModel) dashboardContentWidth() int {
	if m.width <= 0 {
		return 100
	}
	return maxInt(96, m.width-4)
}

func (m liveTUIModel) headerContentWidth() int {
	width := m.dashboardContentWidth() - 6
	if width < 40 {
		return 40
	}
	return width
}

func (m liveTUIModel) metric(label, value string, style lipgloss.Style) string {
	return m.labelStyle.Render(fmt.Sprintf("%-11s", label)) + style.Render(value)
}

func (m liveTUIModel) panel(title, body string, active bool, centered bool) string {
	align := lipgloss.Left
	if centered {
		align = lipgloss.Center
	}
	content := lipgloss.JoinVertical(align, m.panelTitle.Render(title), "", body)
	if active {
		return m.activePanel.Render(content)
	}
	return m.panelStyle.Render(content)
}

func (m liveTUIModel) panelWidth(rendered string, width int) string {
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, rendered)
}

func equalizePanelSizes(panels []string, width int) []string {
	maxHeight := 0
	for _, panel := range panels {
		height := lipgloss.Height(panel)
		if height > maxHeight {
			maxHeight = height
		}
	}

	normalized := make([]string, len(panels))
	for i, panel := range panels {
		normalized[i] = lipgloss.NewStyle().Width(width).Height(maxHeight).Render(panel)
	}

	return normalized
}

func (m liveTUIModel) footer() string {
	parts := []string{"q quit", "tab move", "c compact"}
	if !m.effectiveCompact() {
		parts[1] = "tab/hjkl move"
	}
	parts = append(parts, fmt.Sprintf("refresh %s", m.interval))
	if !m.updatedAt.IsZero() {
		parts = append(parts, "last update "+m.updatedAt.Format("15:04:05"))
	}
	footer := m.footerStyle.Render(strings.Join(parts, "  |  "))
	return m.wrapCentered(footer)
}

func (m *liveTUIModel) recordTrends(snapshot *liveSnapshot) {
	if value, ok := parseMetricNumber(snapshot.BatteryVoltage); ok {
		m.batteryTrend = appendTrend(m.batteryTrend, value)
	}
	if value, ok := parseMetricNumber(snapshot.SolarPower); ok {
		m.solarTrend = appendTrend(m.solarTrend, value)
	}
	if value, ok := parseMetricNumber(snapshot.LoadPower); ok {
		m.loadTrend = appendTrend(m.loadTrend, value)
	}
}

func (m liveTUIModel) badge(label, value string, bg lipgloss.Color) string {
	return m.badgeStyle.Background(bg).Render(label+" "+value) + " "
}

func (m liveTUIModel) overallHealth(snapshot *liveSnapshot) string {
	if snapshot.Stale || len(snapshot.Warnings) > 0 {
		return "WARN"
	}
	return "OK"
}

func (m liveTUIModel) healthStyle(snapshot *liveSnapshot) lipgloss.Style {
	if snapshot.Stale || len(snapshot.Warnings) > 0 {
		return m.warnStyle
	}
	return m.okStyle
}

func (m liveTUIModel) healthBadgeColor(snapshot *liveSnapshot) lipgloss.Color {
	if snapshot.Stale || len(snapshot.Warnings) > 0 {
		return lipgloss.Color("130")
	}
	return lipgloss.Color("28")
}

func (m liveTUIModel) inverterBadgeColor(state string) lipgloss.Color {
	lower := strings.ToLower(state)
	switch {
	case strings.Contains(lower, "fault"), strings.Contains(lower, "alarm"), strings.Contains(lower, "off"):
		return lipgloss.Color("160")
	case strings.Contains(lower, "invert"), strings.Contains(lower, "bulk"), strings.Contains(lower, "absorption"), strings.Contains(lower, "float"), strings.Contains(lower, "passthru"):
		return lipgloss.Color("28")
	default:
		return lipgloss.Color("240")
	}
}

func (m liveTUIModel) gridBadgeColor(status string) lipgloss.Color {
	if status == "Absent" {
		return lipgloss.Color("130")
	}
	if status == "Importing" || status == "Exporting" || status == "Present" {
		return lipgloss.Color("28")
	}
	return lipgloss.Color("240")
}

func (m liveTUIModel) ageBadgeColor(snapshot *liveSnapshot) lipgloss.Color {
	if snapshot.Stale {
		return lipgloss.Color("130")
	}
	return lipgloss.Color("28")
}

func (m liveTUIModel) dataAgeStyle(snapshot *liveSnapshot) lipgloss.Style {
	if snapshot.Stale {
		return m.warnStyle
	}
	return m.okStyle
}

func (m liveTUIModel) inverterStateStyle(state string) lipgloss.Style {
	lower := strings.ToLower(state)
	switch {
	case strings.Contains(lower, "fault"), strings.Contains(lower, "alarm"), strings.Contains(lower, "off"):
		return m.errorStyle
	case strings.Contains(lower, "invert"), strings.Contains(lower, "bulk"), strings.Contains(lower, "absorption"), strings.Contains(lower, "float"), strings.Contains(lower, "passthru"):
		return m.okStyle
	default:
		return m.valueStyle
	}
}

func (m liveTUIModel) gridStateStyle(value string) lipgloss.Style {
	lower := strings.ToLower(value)
	if lower == "n/a" || strings.Contains(lower, "absent") || strings.Contains(lower, "not detected") || value == "0.0 V" {
		return m.warnStyle
	}
	return m.okStyle
}

func (m liveTUIModel) solarStateStyle(state string) lipgloss.Style {
	lower := strings.ToLower(state)
	if strings.Contains(lower, "fault") || strings.Contains(lower, "off") {
		return m.errorStyle
	}
	if strings.Contains(lower, "bulk") || strings.Contains(lower, "float") || strings.Contains(lower, "absorption") || strings.Contains(lower, "ext. control") {
		return m.okStyle
	}
	return m.valueStyle
}

func (m liveTUIModel) emphasizePercent(value string) lipgloss.Style {
	if strings.Contains(value, "n/a") {
		return m.warnStyle
	}
	return m.okStyle
}

func (m liveTUIModel) gridStatus(snapshot *liveSnapshot) string {
	if snapshot.GridImportPower == "n/a" && snapshot.GridExportPower == "n/a" {
		return "Absent"
	}
	if snapshot.GridExportPower != "0 W" && snapshot.GridExportPower != "n/a" {
		return "Exporting"
	}
	if snapshot.GridImportPower != "0 W" && snapshot.GridImportPower != "n/a" {
		return "Importing"
	}
	return "Present"
}

func (m liveTUIModel) trendStyle(values []float64) lipgloss.Style {
	if len(values) < 2 {
		return m.mutedStyle
	}
	delta := values[len(values)-1] - values[len(values)-2]
	if delta > 0.001 {
		return m.okStyle
	}
	if delta < -0.001 {
		return m.warnStyle
	}
	return m.mutedStyle
}

func (m liveTUIModel) trendDirection(values []float64) string {
	if len(values) < 2 {
		return "steady"
	}
	delta := values[len(values)-1] - values[len(values)-2]
	if delta > 0.001 {
		return fmt.Sprintf("up %+0.2f", delta)
	}
	if delta < -0.001 {
		return fmt.Sprintf("down %+0.2f", delta)
	}
	return "steady 0.00"
}

func (m liveTUIModel) trendText(values []float64, unit string) string {
	if len(values) == 0 {
		return "n/a"
	}
	return m.trendDirection(values) + " " + unit
}

func (m liveTUIModel) viewMode() string {
	if m.effectiveCompact() {
		return "COMPACT"
	}
	return "EXPANDED"
}

func (m liveTUIModel) effectiveCompact() bool {
	if m.compact {
		return true
	}
	if m.width > 0 && m.width < 104 {
		return true
	}
	if m.height > 0 && m.height < 32 {
		return true
	}
	return false
}

func (m liveTUIModel) canShowDetail() bool {
	if m.width > 0 && m.width < 118 {
		return false
	}
	if m.height > 0 && m.height < 38 {
		return false
	}
	return true
}

func (m liveTUIModel) wrapCentered(content string) string {
	if m.width <= 0 {
		return content
	}
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content)
}

func appendTrend(values []float64, value float64) []float64 {
	values = append(values, value)
	if len(values) > 24 {
		values = values[len(values)-24:]
	}
	return values
}

func parseMetricNumber(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "n/a" {
		return 0, false
	}
	var builder strings.Builder
	for i, r := range value {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			builder.WriteRune(r)
			continue
		}
		if i == 0 {
			return 0, false
		}
		break
	}
	if builder.Len() == 0 {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(builder.String(), 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func sparkline(values []float64) string {
	if len(values) == 0 {
		return "n/a"
	}
	const chars = "▁▂▃▄▅▆▇█"
	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	if maxValue == minValue {
		return strings.Repeat("▅", len(values))
	}
	var builder strings.Builder
	for _, value := range values {
		index := int((value - minValue) / (maxValue - minValue) * float64(len(chars)-1))
		if index < 0 {
			index = 0
		}
		if index >= len(chars) {
			index = len(chars) - 1
		}
		builder.WriteByte(chars[index])
	}
	return builder.String()
}

func compactTrend(values []float64) string {
	if len(values) == 0 {
		return "n/a"
	}
	return sparkline(values) + " " + shortTrendDirection(values)
}

func shortTrendDirection(values []float64) string {
	if len(values) < 2 {
		return "="
	}
	delta := values[len(values)-1] - values[len(values)-2]
	if delta > 0.001 {
		return "+"
	}
	if delta < -0.001 {
		return "-"
	}
	return "="
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

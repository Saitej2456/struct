package main

/*
#include <stdlib.h>
#include "backend.h"
*/
import "C"

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// --- GLOBALS ---
var rootPath string

// --- STRUCT PACKAGE ARCHITECTURE ---
type StructManifest struct {
	Name  string       `yaml:"name"`
	Files []FileRecord `yaml:"files"`
}

type FileRecord struct {
	Path  string `yaml:"path"`
	Hash  string `yaml:"hash"`
	IsDir bool   `yaml:"is_dir"`
}

// --- ENUMS ---
type sessionState int

const (
	stateMenu sessionState = iota
	stateSelectStruct
	stateDashboard
	statePopupInput
	statePopupConfirm
	statePopupUseChoice
)

type inputMode int

const (
	modeNone inputMode = iota
	modeNewStruct
	modeNewFile
	modeNewDir
	modeNewScript
	modeRename
)

// --- STYLES ---
var (
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	danger    = lipgloss.AdaptiveColor{Light: "#F00", Dark: "#F00"}
	text      = lipgloss.AdaptiveColor{Light: "#434343", Dark: "#DDDDDD"}

	boxStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(highlight).Padding(0, 1)
	titleStyle = lipgloss.NewStyle().MarginLeft(2).MarginTop(1).Foreground(highlight).Bold(true)
	selectedItemStyle = lipgloss.NewStyle().Foreground(highlight).Bold(true)
	itemStyle = lipgloss.NewStyle().PaddingLeft(2)
	popupBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(highlight).Padding(1, 2).Align(lipgloss.Center).Width(60)
	buttonStyle = lipgloss.NewStyle().Foreground(text).Padding(0, 2).Margin(0, 1)
	activeButtonStyle = buttonStyle.Copy().Foreground(lipgloss.Color("#FFF")).Background(highlight).Bold(true)
)

// --- DATA STRUCTURES ---
type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Mode     fs.FileMode
}

type model struct {
	state  sessionState
	width  int
	height int

	menuChoices []string
	menuCursor  int

	textInput    textinput.Model
	activeInMode inputMode

	structList   []string
	structCursor int
	selectMode   string

	// Workspace Management
	activeStructName  string
	activeTempDir     string
	currentStructPath string
	rootConstraint    string
	flatFiles         []*FileNode
	fileCursor        int

	pendingDeletePath string
	pendingUsePath    string
	confirmToggle     bool
}

// --- INIT ---
func initialModel() model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	return model{
		state:         stateMenu,
		menuChoices:   []string{"Create Structure", "Use Structure", "Edit Structure", "Remove Structure", "Exit"},
		textInput:     ti,
		confirmToggle: true,
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

// --- CORE PACKAGING LOGIC ---

// Hash a file
func hashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil { return "", err }
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil { return "", err }
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Smart Compression check
func isStoreExt(ext string) bool {
	storeExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".mp4": true, ".zip": true, ".tar": true, ".gz": true, ".struct": true}
	return storeExts[strings.ToLower(ext)]
}

// Package a temporary directory into a .struct zip archive
func PackStruct(sourceDir string, structName string, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil { return err }
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	manifest := StructManifest{Name: structName}
	blobMap := make(map[string]bool)

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == sourceDir { return nil }

		relPath, _ := filepath.Rel(sourceDir, path)
		record := FileRecord{Path: filepath.ToSlash(relPath), IsDir: info.IsDir()}

		if !info.IsDir() {
			hash, err := hashFile(path)
			if err != nil { return err }
			record.Hash = hash

			// Deduplication: Only write the blob if we haven't seen this hash yet
			if !blobMap[hash] {
				blobMap[hash] = true
				
				// Smart Compression Setup
				header, _ := zip.FileInfoHeader(info)
				header.Name = "data/" + hash + ".structdata"
				header.Method = zip.Deflate
				if isStoreExt(filepath.Ext(path)) { header.Method = zip.Store }

				w, err := zipWriter.CreateHeader(header)
				if err != nil { return err }

				src, err := os.Open(path)
				if err != nil { return err }
				io.Copy(w, src)
				src.Close()
			}
		}
		manifest.Files = append(manifest.Files, record)
		return nil
	})

	if err != nil { return err }

	// Write structure.yaml
	yamlData, _ := yaml.Marshal(&manifest)
	yamlWriter, _ := zipWriter.Create("structure.yaml")
	yamlWriter.Write(yamlData)

	return nil
}

// Extract a .struct archive to a destination
func UnpackStruct(srcZip string, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil { return err }
	defer r.Close()

	var manifest StructManifest
	var yamlFound bool

	// Find and parse structure.yaml first
	for _, f := range r.File {
		if f.Name == "structure.yaml" {
			rc, _ := f.Open()
			yamlData, _ := io.ReadAll(rc)
			yaml.Unmarshal(yamlData, &manifest)
			rc.Close()
			yamlFound = true
			break
		}
	}

	if !yamlFound { return fmt.Errorf("invalid struct package: missing structure.yaml") }

	// Extract files based on map
	for _, record := range manifest.Files {
		fullDest := filepath.Join(destDir, record.Path)
		if record.IsDir {
			os.MkdirAll(fullDest, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(fullDest), 0755)
		
		// Find the blob in the zip
		for _, f := range r.File {
			if f.Name == "data/"+record.Hash+".structdata" {
				rc, _ := f.Open()
				destFile, _ := os.Create(fullDest)
				io.Copy(destFile, rc)
				destFile.Close()
				rc.Close()
				break
			}
		}
	}
	return nil
}

// --- LOGIC ---
func scanCurrentDir(path string) ([]*FileNode, error) {
	entries, err := os.ReadDir(path)
	if err != nil { return nil, err }
	var nodes []*FileNode
	for _, e := range entries {
		info, err := e.Info()
		if err != nil { continue }
		nodes = append(nodes, &FileNode{Name: e.Name(), Path: filepath.Join(path, e.Name()), IsDir: e.IsDir(), Size: info.Size(), Mode: info.Mode()})
	}
	return nodes, nil
}

func scanStructures(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil { return []string{} }
	var structs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".struct") {
			structs = append(structs, strings.TrimSuffix(e.Name(), ".struct"))
		}
	}
	return structs
}

// --- UPDATE ---
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.state != statePopupInput && msg.String() == "ctrl+c" {
			if m.activeTempDir != "" { os.RemoveAll(m.activeTempDir) }
			return m, tea.Quit
		}

		switch m.state {
		
		case stateMenu:
			switch msg.String() {
			case "up", "k":
				if m.menuCursor > 0 { m.menuCursor-- }
			case "down", "j":
				if m.menuCursor < len(m.menuChoices)-1 { m.menuCursor++ }
			case "enter":
				choice := m.menuChoices[m.menuCursor]
				if choice == "Exit" {
					if m.activeTempDir != "" { os.RemoveAll(m.activeTempDir) }
					return m, tea.Quit
				}
				
				if choice == "Create Structure" {
					m.state = statePopupInput
					m.activeInMode = modeNewStruct
					m.textInput.Placeholder = "Enter new structure name..."
					m.textInput.Reset()
					return m, nil
				}
				
				m.structList = scanStructures(rootPath)
				m.structCursor = 0
				m.state = stateSelectStruct
				if choice == "Use Structure" { m.selectMode = "USE" }
				if choice == "Edit Structure" { m.selectMode = "EDIT" }
				if choice == "Remove Structure" { m.selectMode = "REMOVE" }
			}

		case stateSelectStruct:
			switch msg.String() {
			case "q", "esc": m.state = stateMenu
			case "up", "k": if m.structCursor > 0 { m.structCursor-- }
			case "down", "j": if m.structCursor < len(m.structList)-1 { m.structCursor++ }
			case "enter":
				if len(m.structList) == 0 { return m, nil }
				selected := m.structList[m.structCursor]
				
				if m.selectMode == "REMOVE" {
					m.pendingDeletePath = selected
					m.confirmToggle = true
					m.state = statePopupConfirm
				} else if m.selectMode == "USE" {
					m.pendingUsePath = selected
					m.confirmToggle = true
					m.state = statePopupUseChoice
				} else {
					// EDIT MODE: Unpack to workspace
					m.activeStructName = selected
					m.activeTempDir, _ = os.MkdirTemp("", "struct_workspace_*")
					
					zipPath := filepath.Join(rootPath, selected+".struct")
					UnpackStruct(zipPath, m.activeTempDir)

					m.currentStructPath = m.activeTempDir
					m.rootConstraint = m.activeTempDir
					m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
					m.fileCursor = 0
					m.state = stateDashboard
				}
			}

		case stateDashboard:
			switch msg.String() {
			case "q": 
				// Pack and cleanup workspace
				zipPath := filepath.Join(rootPath, m.activeStructName+".struct")
				PackStruct(m.activeTempDir, m.activeStructName, zipPath)
				os.RemoveAll(m.activeTempDir)
				m.activeTempDir = ""
				m.state = stateMenu
			case "up", "k": if m.fileCursor > 0 { m.fileCursor-- }
			case "down", "j": if m.fileCursor < len(m.flatFiles)-1 { m.fileCursor++ }
			case "enter":
				if len(m.flatFiles) > 0 {
					selected := m.flatFiles[m.fileCursor]
					if selected.IsDir {
						m.currentStructPath = selected.Path
						m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
						m.fileCursor = 0
					}
				}
			case "backspace":
				if m.currentStructPath != m.rootConstraint {
					m.currentStructPath = filepath.Dir(m.currentStructPath)
					m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
					m.fileCursor = 0
				}
			case "c": 
				m.state = statePopupInput
				m.activeInMode = modeNewFile
				m.textInput.Placeholder = "Enter file name..."
				m.textInput.Reset()
			case "C": 
				m.state = statePopupInput
				m.activeInMode = modeNewDir
				m.textInput.Placeholder = "Enter directory name..."
				m.textInput.Reset()
			case "s": 
				m.state = statePopupInput
				m.activeInMode = modeNewScript
				m.textInput.Placeholder = "Enter script name (no .sh)..."
				m.textInput.Reset()
			case "r": 
				if len(m.flatFiles) > 0 {
					m.state = statePopupInput
					m.activeInMode = modeRename
					m.textInput.Placeholder = "Enter new name..."
					m.textInput.Reset()
				}
			case "d": 
				if len(m.flatFiles) > 0 {
					m.pendingDeletePath = m.flatFiles[m.fileCursor].Name
					m.confirmToggle = true
					m.state = statePopupConfirm
				}
			}

		case statePopupInput:
			switch msg.String() {
			case "esc":
				if m.activeInMode == modeNewStruct { m.state = stateMenu } else { m.state = stateDashboard }
			case "enter":
				val := m.textInput.Value()
				
				if m.activeInMode == modeNewStruct {
					// Setup workspace for new struct
					m.activeStructName = val
					m.activeTempDir, _ = os.MkdirTemp("", "struct_workspace_*")
					m.currentStructPath = m.activeTempDir
					m.rootConstraint = m.activeTempDir
					m.flatFiles = []*FileNode{}
					m.state = stateDashboard
					return m, nil
				}
				
				cName := C.CString(val)
				cPath := C.CString(m.currentStructPath)
				
				if m.activeInMode == modeNewFile {
					C.Bridge_CreateFile(cPath, cName)
				} else if m.activeInMode == modeNewDir {
					C.Bridge_CreateDir(cPath, cName)
				} else if m.activeInMode == modeNewScript {
					if !strings.HasSuffix(val, ".sh") {
						C.free(unsafe.Pointer(cName))
						val += ".sh"
						cName = C.CString(val)
					}
					C.Bridge_CreateScript(cPath, cName)
				} else if m.activeInMode == modeRename {
					selectedName := m.flatFiles[m.fileCursor].Name
					oldFullPath := filepath.Join(m.currentStructPath, selectedName)
					cOldPath := C.CString(oldFullPath)
					C.Bridge_Rename(cOldPath, cName)
					C.free(unsafe.Pointer(cOldPath))
				}
				C.free(unsafe.Pointer(cName))
				C.free(unsafe.Pointer(cPath))

				m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
				m.state = stateDashboard
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case statePopupConfirm:
			switch msg.String() {
			case "left", "h", "right", "l": m.confirmToggle = !m.confirmToggle
			case "enter":
				if m.confirmToggle {
					if m.selectMode == "REMOVE" {
						os.Remove(filepath.Join(rootPath, m.pendingDeletePath+".struct"))
						m.structList = scanStructures(rootPath)
						m.state = stateSelectStruct
					} else {
						fullPath := filepath.Join(m.currentStructPath, m.pendingDeletePath)
						cPath := C.CString(fullPath)
						C.Bridge_Delete(cPath)
						C.free(unsafe.Pointer(cPath))
						m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
						m.state = stateDashboard
					}
				} else {
					if m.selectMode == "REMOVE" { m.state = stateSelectStruct } else { m.state = stateDashboard }
				}
			case "esc", "q":
				if m.selectMode == "REMOVE" { m.state = stateSelectStruct } else { m.state = stateDashboard }
			}

		case statePopupUseChoice:
			switch msg.String() {
			case "left", "h", "right", "l": m.confirmToggle = !m.confirmToggle
			case "esc", "q": m.state = stateSelectStruct
			case "enter":
				cwd, _ := os.Getwd()
				zipPath := filepath.Join(rootPath, m.pendingUsePath+".struct")
				
				var dest string
				if m.confirmToggle { dest = filepath.Join(cwd, m.pendingUsePath) } else { dest = cwd }
				
				// Deploys the package efficiently without using C
				UnpackStruct(zipPath, dest)
				m.state = stateMenu
				return m, nil
			}
		}
	}
	return m, cmd
}

// --- VIEW ---
func (m model) View() string {
	if m.state == statePopupInput { return m.viewInputPopup() }
	if m.state == statePopupConfirm { return m.viewConfirmPopup() }
	if m.state == statePopupUseChoice { return m.viewUseChoicePopup() }

	switch m.state {
	case stateMenu: return m.viewMenu()
	case stateSelectStruct: return m.viewStructList()
	case stateDashboard: return m.viewDashboard()
	default: return m.viewMenu()
	}
}

// -- VIEW COMPONENTS --

func (m model) viewMenu() string {
	asciiArt := `
   ██╗███████╗████████╗██████╗ ██╗   ██╗ ██████╗████████╗ ██╗██╗  
 ██╔╝██╔════╝╚══██╔══╝██╔══██╗██║   ██║██╔════╝╚══██╔══╝██╔╝╚██╗ 
██╔╝ ███████╗   ██║   ██████╔╝██║   ██║██║        ██║  ██╔╝  ╚██╗
╚██╗ ╚════██║   ██║   ██╔══██╗██║   ██║██║        ██║ ██╔╝   ██╔╝
 ╚██╗███████║   ██║   ██║  ██║╚██████╔╝╚██████╗   ██║██╔╝   ██╔╝ 
  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝  ╚═════╝   ╚═╝╚═╝    ╚═╝  
`
	s := titleStyle.Render(asciiArt) + "\n\n"
	for i, choice := range m.menuChoices {
		cursor := " "
		if m.menuCursor == i {
			cursor = ">"
			s += selectedItemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		} else {
			s += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		}
	}
	return s
}

func (m model) viewStructList() string {
	title := fmt.Sprintf("SELECT STRUCTURE TO %s", m.selectMode)
	s := titleStyle.Render(title) + "\n\n"
	if len(m.structList) == 0 {
		s += itemStyle.Render("No structures found.") + "\n"
	} else {
		for i, choice := range m.structList {
			cursor := " "
			if m.structCursor == i {
				cursor = ">"
				s += selectedItemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
			} else {
				s += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
			}
		}
	}
	return s + "\n" + lipgloss.NewStyle().Foreground(subtle).Render("(Press 'q' to go back)")
}

func (m model) centerPopup(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupBoxStyle.Render(content), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(subtle))
}

func (m model) viewInputPopup() string {
	var title string
	switch m.activeInMode {
	case modeNewStruct: title = "Name for New Structure"
	case modeNewFile: title = "Create New File"
	case modeNewDir: title = "Create New Directory"
	case modeNewScript: title = "Create Script (No .sh extension)"
	case modeRename: title = "Rename Item"
	}
	content := lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render(title), "\n", m.textInput.View(), "\n\n(Enter to confirm, Esc to cancel)")
	return m.centerPopup(content)
}

func (m model) viewConfirmPopup() string {
	question := fmt.Sprintf("Are you sure you want to delete:\n\n'%s'?", m.pendingDeletePath)
	var yesBtn, noBtn string
	if m.confirmToggle { yesBtn = activeButtonStyle.Render("YES"); noBtn = buttonStyle.Render("NO") } else { yesBtn = buttonStyle.Render("YES"); noBtn = activeButtonStyle.Render("NO") }
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn)
	content := lipgloss.JoinVertical(lipgloss.Center, question, "\n", buttons)
	return m.centerPopup(content)
}

func (m model) viewUseChoicePopup() string {
	question := fmt.Sprintf("How do you want to use '%s'?", m.pendingUsePath)
	var dirBtn, contentBtn string
	if m.confirmToggle { dirBtn = activeButtonStyle.Render("AS DIRECTORY"); contentBtn = buttonStyle.Render("CONTENTS ONLY") } else { dirBtn = buttonStyle.Render("AS DIRECTORY"); contentBtn = activeButtonStyle.Render("CONTENTS ONLY") }
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, dirBtn, contentBtn)
	content := lipgloss.JoinVertical(lipgloss.Center, question, "\n", buttons)
	return m.centerPopup(content)
}

func (m model) viewDashboard() string {
	w := m.width - 4
	halfWidth := w/2 - 2

	var fileList string
	if len(m.flatFiles) == 0 {
		fileList = "Directory is empty."
	} else {
		start, end := 0, len(m.flatFiles)
		if end > 12 { if m.fileCursor > 6 { start = m.fileCursor - 6 }; if start+12 < end { end = start + 12 } else { end = len(m.flatFiles); if end-12>=0 { start=end-12 } } }
		for i := start; i < end; i++ {
			f := m.flatFiles[i]
			icon := "📄"
			if f.IsDir { icon = "📁" }
			line := fmt.Sprintf("%s %s", icon, f.Name)
			if i == m.fileCursor { fileList += selectedItemStyle.Render("> " + line) + "\n" } else { fileList += itemStyle.Render("  " + line) + "\n" }
		}
	}
	leftBox := boxStyle.Width(halfWidth).Height(14).Render(fileList)

	// Clean up the temp path for display
	displayPath := strings.Replace(m.currentStructPath, m.rootConstraint, "[WORKSPACE: "+m.activeStructName+"]", 1)
	
	metaContent := fmt.Sprintf("CURRENT LOCATION\n\n%s\n\nItems: %d", displayPath, len(m.flatFiles))
	rightBox := boxStyle.Width(halfWidth).Height(14).Render(metaContent)

	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	var fileMetaContent string
	if len(m.flatFiles) > 0 {
		f := m.flatFiles[m.fileCursor]
		fileMetaContent = fmt.Sprintf("SELECTED INFO\n\nName: %s\nType: %s\nPermissions: %s\nSize: %d bytes", f.Name, func() string { if f.IsDir { return "Directory" }; return "File" }(), f.Mode, f.Size)
	} else { fileMetaContent = "No file selected." }
	middleSection := boxStyle.Width(w-2).Height(6).Render(fileMetaContent)

	col1 := "[c] Create File\n[C] Create Dir\n[s] Create Script"
	col2 := "[r] Rename\n[d] Remove\n[Enter] Move In"
	col3 := "[Bksp] Parent Dir\n[q] End/Back & Pack"
	keysText := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width((w-4)/3).Render(col1), lipgloss.NewStyle().Width((w-4)/3).Render(col2), lipgloss.NewStyle().Width((w-4)/3).Render(col3))
	bottomSection := boxStyle.Width(w-2).Height(5).Render("KEY MAPPINGS\n\n" + keysText)

	return lipgloss.JoinVertical(lipgloss.Left, topSection, middleSection, bottomSection)
}

func main() {
	home, _ := os.UserHomeDir()
	rootPath = filepath.Join(home, ".struct", "structures")
	os.MkdirAll(rootPath, 0755)

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
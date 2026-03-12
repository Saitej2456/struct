package main

/*
#include <stdlib.h>
#include "backend.h"
*/
import "C"

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/psanford/wormhole-william/wormhole"
	"gopkg.in/yaml.v3"
)

// --- GLOBALS ---
var rootPath string
var p *tea.Program // Global reference to send messages from background Goroutines

// --- ASYNC JOB TRACKING ---
type JobTracker struct {
	ID       string
	Name     string
	Status   string
	Progress float64 // 0.0 to 1.0
	IsDone   bool
}

var activeJobs []*JobTracker
var jobsMutex sync.Mutex

// --- STRUCT PACKAGE ARCHITECTURE ---
type ActiveScript struct {
	Path     string `yaml:"path"`
	Priority int    `yaml:"priority"`
}

type StructManifest struct {
	Name          string         `yaml:"name"`
	Files         []FileRecord   `yaml:"files"`
	ActiveScripts []ActiveScript `yaml:"active_scripts,omitempty"`
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
	stateUploadMenu
	statePopupUploadConfirm
	statePopupCollisionChoice
	stateProgress
	stateTransferMenu
	stateShowWormholeCode
	statePopupVerifyWormhole // NEW: Interrupt state for MITM verification
)

type inputMode int

const (
	modeNone inputMode = iota
	modeNewStruct
	modeNewFile
	modeNewDir
	modeNewScript
	modeRename
	modeManualRenameUpload
	modeSetPriority
	modeReceiveCode
)

// --- STYLES ---
var (
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	danger    = lipgloss.AdaptiveColor{Light: "#F00", Dark: "#F00"}
	text      = lipgloss.AdaptiveColor{Light: "#434343", Dark: "#DDDDDD"}

	boxStyle          = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(highlight).Padding(0, 1)
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).MarginTop(1).Foreground(highlight).Bold(true)
	selectedItemStyle = lipgloss.NewStyle().Foreground(highlight).Bold(true)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	popupBoxStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(highlight).Padding(1, 2).Align(lipgloss.Center).Width(60)
	buttonStyle       = lipgloss.NewStyle().Foreground(text).Padding(0, 2).Margin(0, 1)
	activeButtonStyle = buttonStyle.Copy().Foreground(lipgloss.Color("#FFF")).Background(highlight).Bold(true)
)

// --- DATA STRUCTURES ---
type FileNode struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
	Mode  fs.FileMode
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

	uploadMenuChoices []string
	uploadMenuCursor  int

	transferMenuChoices []string
	transferMenuCursor  int

	// Workspace & Navigation
	activeStructName   string
	activeTempDir      string
	currentStructPath  string
	rootConstraint     string
	flatFiles          []*FileNode
	fileCursor         int
	offlineBrowserMode bool

	// Active Scripts
	activeScripts map[string]int

	// Popups
	pendingDeletePath string
	pendingUsePath    string
	pendingUploadPath string
	pendingUploadName string
	confirmToggle     bool
	collisionChoice   int

	// Wormhole P2P State
	wormholeCode         string
	sendCtx              context.Context
	sendCancel           context.CancelFunc
	wormholeVerifier     string      // NEW: Stores the PAKE verifier code
	verifierResponseChan chan<- bool // NEW: Channel to unblock the Goroutine
}

// --- INIT ---
func initialModel() model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	return model{
		state:               stateMenu,
		menuChoices:         []string{"Create Structure", "Use Structure", "Edit Structure", "Remove Structure", "Upload Structure", "Transfer Structs", "See Progress", "Exit"},
		uploadMenuChoices:   []string{"Upload to Online", "Upload from Offline"},
		transferMenuChoices: []string{"Send Structure", "Receive Structure"},
		textInput:           ti,
		confirmToggle:       true,
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

// --- CUSTOM MESSAGES & ADAPTERS ---
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type editorFinishedMsg struct{ err error }

type editorCmdWrap struct{ *exec.Cmd }

func (w editorCmdWrap) SetStdin(r io.Reader)   { w.Cmd.Stdin = r }
func (w editorCmdWrap) SetStdout(wr io.Writer) { w.Cmd.Stdout = wr }
func (w editorCmdWrap) SetStderr(wr io.Writer) { w.Cmd.Stderr = wr }

type wormholeCodeMsg string

// NEW: Message sent from the background Goroutine to prompt the UI
type verifierPromptMsg struct {
	Verifier     string
	ResponseChan chan<- bool
}

// --- PROGRESS TRACKING WRITER ---
type progressWriter struct {
	io.Writer
	total   int64
	written int64
	job     *JobTracker
}

func (pw *progressWriter) Write(pr []byte) (int, error) {
	n, err := pw.Writer.Write(pr)
	pw.written += int64(n)
	if pw.total > 0 {
		pw.job.Progress = float64(pw.written) / float64(pw.total)
	}
	return n, err
}

// --- P2P MAGIC WORMHOLE LOGIC ---
func startSendWormhole(ctx context.Context, filePath string, job *JobTracker) tea.Cmd {
	return func() tea.Msg {
		c := wormhole.Client{
			// The hook that stops execution right after PAKE handshake
			VerifierOk: func(verifier string) bool {
				job.Status = "Awaiting Verification..."
				respChan := make(chan bool)
				p.Send(verifierPromptMsg{Verifier: verifier, ResponseChan: respChan})
				
				approved := <-respChan // Wait for Bubble Tea UI to reply
				if approved {
					job.Status = "Transferring..."
				} else {
					job.Status = "Verification Rejected"
				}
				return approved
			},
		}

		f, err := os.Open(filePath)
		if err != nil {
			job.Status = "Error: " + err.Error()
			job.IsDone = true
			return wormholeCodeMsg("Error opening file.")
		}

		fileName := filepath.Base(filePath)
		code, statusChan, err := c.SendFile(ctx, fileName, f)
		if err != nil {
			f.Close()
			job.Status = "Error: " + err.Error()
			job.IsDone = true
			return wormholeCodeMsg("Error generating code.")
		}

		go func() {
			defer f.Close()
			job.Status = "Waiting for peer to connect..."

			res := <-statusChan
			if res.Error != nil {
				job.Status = "Failed: " + res.Error.Error()
			} else {
				job.Status = "Completed"
				job.Progress = 1.0
			}
			job.IsDone = true
		}()

		return wormholeCodeMsg(code)
	}
}

func receiveWormhole(code string, destDir string, job *JobTracker) {
	c := wormhole.Client{
		VerifierOk: func(verifier string) bool {
			job.Status = "Awaiting Verification..."
			respChan := make(chan bool)
			p.Send(verifierPromptMsg{Verifier: verifier, ResponseChan: respChan})
			
			approved := <-respChan
			if approved {
				job.Status = "Downloading..."
			} else {
				job.Status = "Verification Rejected"
			}
			return approved
		},
	}
	ctx := context.Background()

	job.Status = "Connecting to peer..."
	msg, err := c.Receive(ctx, code)
	if err != nil {
		job.Status = "Error: " + err.Error()
		job.IsDone = true
		return
	}

	if msg.Type != wormhole.TransferFile {
		msg.Reject()
		job.Status = "Error: Expected a file."
		job.IsDone = true
		return
	}

	destPath := filepath.Join(destDir, msg.Name)
	f, err := os.Create(destPath)
	if err != nil {
		msg.Reject()
		job.Status = "Error creating file."
		job.IsDone = true
		return
	}
	defer f.Close()

	pw := &progressWriter{Writer: f, total: int64(msg.TransferBytes), job: job}

	_, err = io.Copy(pw, msg)
	if err != nil {
		job.Status = "Transfer failed."
	} else {
		job.Status = "Completed"
		job.Progress = 1.0
	}
	job.IsDone = true
}

// --- HELPERS ---
func getRelPath(root, full string) string {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return filepath.Base(full)
	}
	return filepath.ToSlash(rel)
}

func hashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isStoreExt(ext string) bool {
	storeExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".mp4": true, ".zip": true, ".tar": true, ".gz": true, ".struct": true}
	return storeExts[strings.ToLower(ext)]
}

// --- CORE PACKAGING LOGIC ---
func PackStruct(sourceDir string, structName string, destZip string, activeScripts map[string]int) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	manifest := StructManifest{Name: structName}
	for p, prio := range activeScripts {
		manifest.ActiveScripts = append(manifest.ActiveScripts, ActiveScript{Path: p, Priority: prio})
	}

	blobMap := make(map[string]bool)

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == sourceDir {
			return nil
		}
		relPath, _ := filepath.Rel(sourceDir, path)
		record := FileRecord{Path: filepath.ToSlash(relPath), IsDir: info.IsDir()}

		if !info.IsDir() {
			hash, err := hashFile(path)
			if err != nil {
				return err
			}
			record.Hash = hash

			if !blobMap[hash] {
				blobMap[hash] = true
				header, _ := zip.FileInfoHeader(info)
				header.Name = "data/" + hash + ".structdata"
				header.Method = zip.Deflate
				if isStoreExt(filepath.Ext(path)) {
					header.Method = zip.Store
				}
				w, err := zipWriter.CreateHeader(header)
				if err != nil {
					return err
				}
				src, err := os.Open(path)
				if err != nil {
					return err
				}
				io.Copy(w, src)
				src.Close()
			}
		}
		manifest.Files = append(manifest.Files, record)
		return nil
	})

	if err != nil {
		return err
	}
	yamlData, _ := yaml.Marshal(&manifest)
	yamlWriter, _ := zipWriter.Create("structure.yaml")
	yamlWriter.Write(yamlData)
	return nil
}

func PeekManifest(srcZip string) (StructManifest, error) {
	var m StructManifest
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return m, err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "structure.yaml" {
			rc, _ := f.Open()
			yamlData, _ := io.ReadAll(rc)
			yaml.Unmarshal(yamlData, &m)
			rc.Close()
			return m, nil
		}
	}
	return m, fmt.Errorf("invalid struct package")
}

func UnpackStruct(srcZip string, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer r.Close()

	manifest, err := PeekManifest(srcZip)
	if err != nil {
		return err
	}

	for _, record := range manifest.Files {
		fullDest := filepath.Join(destDir, record.Path)
		if record.IsDir {
			os.MkdirAll(fullDest, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(fullDest), 0755)

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

func deployStructure(zipPath, destDir string, manifest StructManifest, job *JobTracker) {
	job.Status = "Extracting Base Files..."
	job.Progress = 0.1
	UnpackStruct(zipPath, destDir)
	job.Progress = 0.5

	if len(manifest.ActiveScripts) > 0 {
		logPath := filepath.Join(destDir, "struct.log")
		logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		defer logFile.Close()

		scripts := manifest.ActiveScripts
		sort.Slice(scripts, func(i, j int) bool {
			if scripts[i].Priority != scripts[j].Priority {
				return scripts[i].Priority > scripts[j].Priority
			}
			depthI := strings.Count(scripts[i].Path, "/")
			depthJ := strings.Count(scripts[j].Path, "/")
			return depthI < depthJ
		})

		for i, script := range scripts {
			job.Status = fmt.Sprintf("Running script: %s...", filepath.Base(script.Path))
			logFile.WriteString(fmt.Sprintf("\n// %s //\n", script.Path))

			scriptPath := filepath.Join(destDir, script.Path)
			cmd := exec.Command("bash", scriptPath)
			cmd.Dir = destDir

			out, err := cmd.CombinedOutput()
			logFile.Write(out)
			if err != nil {
				logFile.WriteString(fmt.Sprintf("\nError Executing Script: %v\n", err))
			}

			job.Progress = 0.5 + 0.5*(float64(i+1)/float64(len(scripts)))
		}
	}

	job.Status = "Completed"
	job.Progress = 1.0
	job.IsDone = true
}

func scanCurrentDir(path string) ([]*FileNode, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var nodes []*FileNode
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		nodes = append(nodes, &FileNode{Name: e.Name(), Path: filepath.Join(path, e.Name()), IsDir: e.IsDir(), Size: info.Size(), Mode: info.Mode()})
	}
	return nodes, nil
}

func scanStructures(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return []string{}
	}
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

	case tickMsg:
		if m.state == stateProgress {
			return m, tickCmd()
		}
		return m, nil

	case wormholeCodeMsg:
		m.wormholeCode = string(msg)
		return m, nil

	case verifierPromptMsg:
		m.wormholeVerifier = msg.Verifier
		m.verifierResponseChan = msg.ResponseChan
		m.confirmToggle = false // Force user to manually tap Left to 'YES'
		m.state = statePopupVerifyWormhole
		return m, nil

	case editorFinishedMsg:
		m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
		return m, nil

	case tea.KeyMsg:
		if m.state != statePopupInput && msg.String() == "ctrl+c" {
			if m.activeTempDir != "" {
				os.RemoveAll(m.activeTempDir)
			}
			if m.sendCancel != nil {
				m.sendCancel()
			}
			// Unblock goroutine to prevent memory leak
			if m.verifierResponseChan != nil {
				m.verifierResponseChan <- false
			}
			return m, tea.Quit
		}

		switch m.state {

		case stateMenu:
			switch msg.String() {
			case "up", "k":
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			case "down", "j":
				if m.menuCursor < len(m.menuChoices)-1 {
					m.menuCursor++
				}
			case "enter":
				choice := m.menuChoices[m.menuCursor]
				if choice == "Exit" {
					if m.activeTempDir != "" {
						os.RemoveAll(m.activeTempDir)
					}
					return m, tea.Quit
				}

				if choice == "Create Structure" {
					m.offlineBrowserMode = false
					m.activeScripts = make(map[string]int)
					m.state = statePopupInput
					m.activeInMode = modeNewStruct
					m.textInput.Placeholder = "Enter new structure name..."
					m.textInput.Reset()
					return m, nil
				}

				if choice == "Upload Structure" {
					m.uploadMenuCursor = 0
					m.state = stateUploadMenu
					return m, nil
				}

				if choice == "Transfer Structs" {
					m.transferMenuCursor = 0
					m.state = stateTransferMenu
					return m, nil
				}

				if choice == "See Progress" {
					m.state = stateProgress
					return m, tickCmd()
				}

				m.offlineBrowserMode = false
				m.structList = scanStructures(rootPath)
				m.structCursor = 0
				m.state = stateSelectStruct
				if choice == "Use Structure" {
					m.selectMode = "USE"
				}
				if choice == "Edit Structure" {
					m.selectMode = "EDIT"
				}
				if choice == "Remove Structure" {
					m.selectMode = "REMOVE"
				}
			}

		case stateTransferMenu:
			switch msg.String() {
			case "q", "esc":
				m.state = stateMenu
			case "up", "k":
				if m.transferMenuCursor > 0 {
					m.transferMenuCursor--
				}
			case "down", "j":
				if m.transferMenuCursor < len(m.transferMenuChoices)-1 {
					m.transferMenuCursor++
				}
			case "enter":
				choice := m.transferMenuChoices[m.transferMenuCursor]
				if choice == "Send Structure" {
					m.structList = scanStructures(rootPath)
					m.structCursor = 0
					m.selectMode = "SEND"
					m.state = stateSelectStruct
				} else if choice == "Receive Structure" {
					m.state = statePopupInput
					m.activeInMode = modeReceiveCode
					m.textInput.Placeholder = "Enter Wormhole Code..."
					m.textInput.Reset()
				}
			}

		case stateShowWormholeCode:
			switch msg.String() {
			case "q", "esc":
				if m.sendCancel != nil {
					m.sendCancel()
					m.sendCancel = nil
				}
				m.state = stateMenu
			}

		case statePopupVerifyWormhole:
			switch msg.String() {
			case "left", "h", "right", "l":
				m.confirmToggle = !m.confirmToggle
			case "enter":
				if m.verifierResponseChan != nil {
					m.verifierResponseChan <- m.confirmToggle
					m.verifierResponseChan = nil
				}
				if m.confirmToggle {
					m.state = stateProgress
					return m, tickCmd()
				} else {
					m.state = stateMenu
					return m, nil
				}
			case "esc", "q":
				if m.verifierResponseChan != nil {
					m.verifierResponseChan <- false
					m.verifierResponseChan = nil
				}
				m.state = stateMenu
				return m, nil
			}

		case stateProgress:
			switch msg.String() {
			case "q", "esc":
				m.state = stateMenu
			}

		case stateUploadMenu:
			switch msg.String() {
			case "q", "esc":
				m.state = stateMenu
			case "up", "k":
				if m.uploadMenuCursor > 0 {
					m.uploadMenuCursor--
				}
			case "down", "j":
				if m.uploadMenuCursor < len(m.uploadMenuChoices)-1 {
					m.uploadMenuCursor++
				}
			case "enter":
				choice := m.uploadMenuChoices[m.uploadMenuCursor]
				if choice == "Upload to Online" {
					m.state = stateMenu
				} else if choice == "Upload from Offline" {
					m.offlineBrowserMode = true
					m.rootConstraint = ""
					home, _ := os.UserHomeDir()
					m.currentStructPath = home
					m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
					m.activeScripts = make(map[string]int)
					m.fileCursor = 0
					m.state = stateDashboard
				}
			}

		case stateSelectStruct:
			switch msg.String() {
			case "q", "esc":
				m.state = stateMenu
			case "up", "k":
				if m.structCursor > 0 {
					m.structCursor--
				}
			case "down", "j":
				if m.structCursor < len(m.structList)-1 {
					m.structCursor++
				}
			case "enter":
				if len(m.structList) == 0 {
					return m, nil
				}
				selected := m.structList[m.structCursor]

				if m.selectMode == "SEND" {
					zipPath := filepath.Join(rootPath, selected+".struct")

					job := &JobTracker{
						ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
						Name:     "Send: " + selected,
						Status:   "Generating wormhole code...",
						Progress: 0.0,
						IsDone:   false,
					}
					jobsMutex.Lock()
					activeJobs = append(activeJobs, job)
					jobsMutex.Unlock()

					m.sendCtx, m.sendCancel = context.WithCancel(context.Background())
					m.wormholeCode = "Generating secure code..."
					m.state = stateShowWormholeCode

					return m, startSendWormhole(m.sendCtx, zipPath, job)
				}

				if m.selectMode == "REMOVE" {
					m.pendingDeletePath = selected
					m.confirmToggle = true
					m.state = statePopupConfirm
				} else if m.selectMode == "USE" {
					m.pendingUsePath = selected
					m.confirmToggle = true
					m.state = statePopupUseChoice
				} else {
					m.activeStructName = selected
					m.activeTempDir, _ = os.MkdirTemp("", "struct_workspace_*")
					zipPath := filepath.Join(rootPath, selected+".struct")

					manifest, _ := PeekManifest(zipPath)
					m.activeScripts = make(map[string]int)
					for _, s := range manifest.ActiveScripts {
						m.activeScripts[s.Path] = s.Priority
					}

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
				if m.offlineBrowserMode {
					m.offlineBrowserMode = false
					m.state = stateMenu
				} else {
					zipPath := filepath.Join(rootPath, m.activeStructName+".struct")
					PackStruct(m.activeTempDir, m.activeStructName, zipPath, m.activeScripts)
					os.RemoveAll(m.activeTempDir)
					m.activeTempDir = ""
					m.state = stateMenu
				}
			case "up", "k":
				if m.fileCursor > 0 {
					m.fileCursor--
				}
			case "down", "j":
				if m.fileCursor < len(m.flatFiles)-1 {
					m.fileCursor++
				}
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
				parent := filepath.Dir(m.currentStructPath)
				if !m.offlineBrowserMode {
					if m.currentStructPath != m.rootConstraint {
						m.currentStructPath = parent
						m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
						m.fileCursor = 0
					}
				} else {
					if m.currentStructPath != parent {
						m.currentStructPath = parent
						m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
						m.fileCursor = 0
					}
				}
			case "u":
				if m.offlineBrowserMode && len(m.flatFiles) > 0 {
					selected := m.flatFiles[m.fileCursor]
					if selected.IsDir {
						m.pendingUploadPath = selected.Path
						m.pendingUploadName = selected.Name
						m.confirmToggle = true
						m.state = statePopupUploadConfirm
					}
				}

			case "v", "n":
				if !m.offlineBrowserMode && len(m.flatFiles) > 0 {
					selected := m.flatFiles[m.fileCursor]
					if !selected.IsDir {
						editor := "vim"
						if msg.String() == "n" {
							editor = "nano"
						}
						fullPath := filepath.Join(m.currentStructPath, selected.Name)
						cmd := exec.Command(editor, fullPath)
						return m, tea.Exec(editorCmdWrap{cmd}, func(err error) tea.Msg { return editorFinishedMsg{err} })
					}
				}

			case "a":
				if !m.offlineBrowserMode && len(m.flatFiles) > 0 {
					f := m.flatFiles[m.fileCursor]
					if !f.IsDir && strings.HasSuffix(f.Name, ".sh") {
						rel := getRelPath(m.rootConstraint, f.Path)
						if _, exists := m.activeScripts[rel]; exists {
							delete(m.activeScripts, rel)
						} else {
							m.activeScripts[rel] = 0
						}
					}
				}
			case "p":
				if !m.offlineBrowserMode && len(m.flatFiles) > 0 {
					f := m.flatFiles[m.fileCursor]
					if !f.IsDir && strings.HasSuffix(f.Name, ".sh") {
						rel := getRelPath(m.rootConstraint, f.Path)
						if _, exists := m.activeScripts[rel]; exists {
							m.state = statePopupInput
							m.activeInMode = modeSetPriority
							m.textInput.Placeholder = "Enter numerical priority..."
							m.textInput.Reset()
						}
					}
				}

			case "c":
				if !m.offlineBrowserMode {
					m.state = statePopupInput
					m.activeInMode = modeNewFile
					m.textInput.Placeholder = "Enter file name..."
					m.textInput.Reset()
				}
			case "C":
				if !m.offlineBrowserMode {
					m.state = statePopupInput
					m.activeInMode = modeNewDir
					m.textInput.Placeholder = "Enter directory name..."
					m.textInput.Reset()
				}
			case "s":
				if !m.offlineBrowserMode {
					m.state = statePopupInput
					m.activeInMode = modeNewScript
					m.textInput.Placeholder = "Enter script name (no .sh)..."
					m.textInput.Reset()
				}
			case "r":
				if !m.offlineBrowserMode && len(m.flatFiles) > 0 {
					m.state = statePopupInput
					m.activeInMode = modeRename
					m.textInput.Placeholder = "Enter new name..."
					m.textInput.Reset()
				}
			case "d":
				if !m.offlineBrowserMode && len(m.flatFiles) > 0 {
					m.pendingDeletePath = m.flatFiles[m.fileCursor].Name
					m.confirmToggle = true
					m.state = statePopupConfirm
				}
			}

		case statePopupUploadConfirm:
			switch msg.String() {
			case "left", "h", "right", "l":
				m.confirmToggle = !m.confirmToggle
			case "esc", "q":
				m.state = stateDashboard
			case "enter":
				if m.confirmToggle {
					destFile := filepath.Join(rootPath, m.pendingUploadName+".struct")
					if _, err := os.Stat(destFile); err == nil {
						m.collisionChoice = 0
						m.state = statePopupCollisionChoice
					} else {
						PackStruct(m.pendingUploadPath, m.pendingUploadName, destFile, m.activeScripts)
						m.state = stateDashboard
					}
				} else {
					m.state = stateDashboard
				}
			}

		case statePopupCollisionChoice:
			switch msg.String() {
			case "left", "h":
				if m.collisionChoice > 0 {
					m.collisionChoice--
				}
			case "right", "l":
				if m.collisionChoice < 2 {
					m.collisionChoice++
				}
			case "esc", "q":
				m.state = stateDashboard
			case "enter":
				if m.collisionChoice == 0 {
					baseName := m.pendingUploadName
					finalName := baseName
					counter := 1
					for {
						finalName = fmt.Sprintf("%s%d", baseName, counter)
						if _, err := os.Stat(filepath.Join(rootPath, finalName+".struct")); os.IsNotExist(err) {
							break
						}
						counter++
					}
					dest := filepath.Join(rootPath, finalName+".struct")
					PackStruct(m.pendingUploadPath, finalName, dest, m.activeScripts)
					m.state = stateDashboard
				} else if m.collisionChoice == 1 {
					m.state = statePopupInput
					m.activeInMode = modeManualRenameUpload
					m.textInput.Placeholder = "Enter new structure name..."
					m.textInput.Reset()
				} else if m.collisionChoice == 2 {
					dest := filepath.Join(rootPath, m.pendingUploadName+".struct")
					PackStruct(m.pendingUploadPath, m.pendingUploadName, dest, m.activeScripts)
					m.state = stateDashboard
				}
			}

		case statePopupInput:
			switch msg.String() {
			case "esc":
				if m.activeInMode == modeNewStruct || m.activeInMode == modeReceiveCode {
					m.state = stateMenu
				} else {
					m.state = stateDashboard
				}
			case "enter":
				val := m.textInput.Value()

				if m.activeInMode == modeReceiveCode {
					job := &JobTracker{
						ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
						Name:     "Receive Wormhole",
						Status:   "Connecting...",
						Progress: 0.0,
						IsDone:   false,
					}
					jobsMutex.Lock()
					activeJobs = append(activeJobs, job)
					jobsMutex.Unlock()

					go receiveWormhole(val, rootPath, job)
					m.state = stateProgress
					return m, tickCmd()
				}

				if m.activeInMode == modeManualRenameUpload {
					dest := filepath.Join(rootPath, val+".struct")
					PackStruct(m.pendingUploadPath, val, dest, m.activeScripts)
					m.state = stateDashboard
					return m, nil
				}

				if m.activeInMode == modeSetPriority {
					if pVal, err := strconv.Atoi(val); err == nil {
						selectedName := m.flatFiles[m.fileCursor].Name
						rel := getRelPath(m.rootConstraint, filepath.Join(m.currentStructPath, selectedName))
						m.activeScripts[rel] = pVal
					}
					m.state = stateDashboard
					return m, nil
				}

				if m.activeInMode == modeNewStruct {
					m.activeStructName = val
					m.activeTempDir, _ = os.MkdirTemp("", "struct_workspace_*")
					m.currentStructPath = m.activeTempDir
					m.rootConstraint = m.activeTempDir
					m.flatFiles = []*FileNode{}
					m.activeScripts = make(map[string]int)
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
					oldRel := getRelPath(m.rootConstraint, oldFullPath)

					cOldPath := C.CString(oldFullPath)
					C.Bridge_Rename(cOldPath, cName)
					C.free(unsafe.Pointer(cOldPath))

					if prio, exists := m.activeScripts[oldRel]; exists {
						delete(m.activeScripts, oldRel)
						newFullPath := filepath.Join(m.currentStructPath, val)
						newRel := getRelPath(m.rootConstraint, newFullPath)
						m.activeScripts[newRel] = prio
					}
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
			case "left", "h", "right", "l":
				m.confirmToggle = !m.confirmToggle
			case "enter":
				if m.confirmToggle {
					if m.selectMode == "REMOVE" {
						os.Remove(filepath.Join(rootPath, m.pendingDeletePath+".struct"))
						m.structList = scanStructures(rootPath)
						m.state = stateSelectStruct
					} else {
						fullPath := filepath.Join(m.currentStructPath, m.pendingDeletePath)
						rel := getRelPath(m.rootConstraint, fullPath)
						delete(m.activeScripts, rel)

						cPath := C.CString(fullPath)
						C.Bridge_Delete(cPath)
						C.free(unsafe.Pointer(cPath))
						m.flatFiles, _ = scanCurrentDir(m.currentStructPath)
						m.state = stateDashboard
					}
				} else {
					if m.selectMode == "REMOVE" {
						m.state = stateSelectStruct
					} else {
						m.state = stateDashboard
					}
				}
			case "esc", "q":
				if m.selectMode == "REMOVE" {
					m.state = stateSelectStruct
				} else {
					m.state = stateDashboard
				}
			}

		case statePopupUseChoice:
			switch msg.String() {
			case "left", "h", "right", "l":
				m.confirmToggle = !m.confirmToggle
			case "esc", "q":
				m.state = stateSelectStruct
			case "enter":
				cwd, _ := os.Getwd()
				zipPath := filepath.Join(rootPath, m.pendingUsePath+".struct")
				var dest string
				if m.confirmToggle {
					dest = filepath.Join(cwd, m.pendingUsePath)
				} else {
					dest = cwd
				}

				manifest, _ := PeekManifest(zipPath)

				job := &JobTracker{
					ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
					Name:     manifest.Name,
					Status:   "Pending Execution...",
					Progress: 0.0,
					IsDone:   false,
				}

				jobsMutex.Lock()
				activeJobs = append(activeJobs, job)
				jobsMutex.Unlock()
				go deployStructure(zipPath, dest, manifest, job)

				m.state = stateMenu
				return m, nil
			}
		}
	}
	return m, cmd
}

// --- VIEW ---
func (m model) View() string {
	if m.state == statePopupInput {
		return m.viewInputPopup()
	}
	if m.state == statePopupConfirm {
		return m.viewConfirmPopup()
	}
	if m.state == statePopupUseChoice {
		return m.viewUseChoicePopup()
	}
	if m.state == statePopupUploadConfirm {
		return m.viewUploadConfirmPopup()
	}
	if m.state == statePopupCollisionChoice {
		return m.viewCollisionChoicePopup()
	}
	if m.state == statePopupVerifyWormhole {
		return m.viewVerifyWormholePopup()
	}

	switch m.state {
	case stateMenu:
		return m.viewMenu()
	case stateUploadMenu:
		return m.viewUploadMenu()
	case stateTransferMenu:
		return m.viewTransferMenu()
	case stateSelectStruct:
		return m.viewStructList()
	case stateDashboard:
		return m.viewDashboard()
	case stateProgress:
		return m.viewProgress()
	case stateShowWormholeCode:
		return m.viewWormholeCode()
	default:
		return m.viewMenu()
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

func (m model) viewUploadMenu() string {
	s := titleStyle.Render("UPLOAD STRUCTURE") + "\n\n"
	for i, choice := range m.uploadMenuChoices {
		cursor := " "
		if m.uploadMenuCursor == i {
			cursor = ">"
			s += selectedItemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		} else {
			s += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		}
	}
	return s + "\n" + lipgloss.NewStyle().Foreground(subtle).Render("(Press 'q' to go back)")
}

func (m model) viewTransferMenu() string {
	s := titleStyle.Render("TRANSFER STRUCTS (P2P Magic Wormhole)") + "\n\n"
	for i, choice := range m.transferMenuChoices {
		cursor := " "
		if m.transferMenuCursor == i {
			cursor = ">"
			s += selectedItemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		} else {
			s += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		}
	}
	return s + "\n" + lipgloss.NewStyle().Foreground(subtle).Render("(Press 'q' to go back)")
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

func (m model) viewWormholeCode() string {
	title := titleStyle.Render("SECURE P2P TRANSFER")
	instruction := "Share this code with the receiver:\n(Transfer will start automatically when they connect)"
	codeBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(highlight).Padding(1, 4).Render(m.wormholeCode)

	content := lipgloss.JoinVertical(lipgloss.Center, title, "\n", instruction, "\n", codeBox, "\n\n(Press 'q' or 'esc' to cancel)")
	return m.centerPopup(content)
}

func (m model) viewProgress() string {
	s := titleStyle.Render("ACTIVE TASKS PROGRESS") + "\n\n"

	jobsMutex.Lock()
	if len(activeJobs) == 0 {
		s += itemStyle.Render("No background tasks running.") + "\n"
	} else {
		for _, job := range activeJobs {
			barWidth := 30
			filled := int(job.Progress * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}

			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			statusLine := fmt.Sprintf("[%s] %s - %s", bar, job.Name, job.Status)

			if job.IsDone {
				s += selectedItemStyle.Render(statusLine) + "\n"
			} else {
				s += itemStyle.Render(statusLine) + "\n"
			}
		}
	}
	jobsMutex.Unlock()

	return s + "\n\n" + lipgloss.NewStyle().Foreground(subtle).Render("(Press 'q' to go back)")
}

func (m model) centerPopup(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupBoxStyle.Render(content), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(subtle))
}

func (m model) viewVerifyWormholePopup() string {
	question := fmt.Sprintf("SECURITY VERIFICATION\n\nDoes the peer see this exact fingerprint?\n\n%s",
		lipgloss.NewStyle().Foreground(highlight).Bold(true).Render(m.wormholeVerifier))

	var yesBtn, noBtn string
	if m.confirmToggle {
		yesBtn = activeButtonStyle.Render("YES (Match)")
		noBtn = buttonStyle.Render("NO (Abort)")
	} else {
		yesBtn = buttonStyle.Render("YES (Match)")
		noBtn = activeButtonStyle.Render("NO (Abort)")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn)
	content := lipgloss.JoinVertical(lipgloss.Center, question, "\n", buttons)
	return m.centerPopup(content)
}

func (m model) viewInputPopup() string {
	var title string
	switch m.activeInMode {
	case modeNewStruct:
		title = "Name for New Structure"
	case modeNewFile:
		title = "Create New File"
	case modeNewDir:
		title = "Create New Directory"
	case modeNewScript:
		title = "Create Script (No .sh extension)"
	case modeRename:
		title = "Rename Item"
	case modeManualRenameUpload:
		title = "Structure collision. Enter new name:"
	case modeSetPriority:
		title = "Set Execution Priority (High Number = First)"
	case modeReceiveCode:
		title = "Enter Wormhole Code from Sender:"
	}
	content := lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render(title), "\n", m.textInput.View(), "\n\n(Enter to confirm, Esc to cancel)")
	return m.centerPopup(content)
}

func (m model) viewConfirmPopup() string {
	question := fmt.Sprintf("Are you sure you want to delete:\n\n'%s'?", m.pendingDeletePath)
	var yesBtn, noBtn string
	if m.confirmToggle {
		yesBtn = activeButtonStyle.Render("YES")
		noBtn = buttonStyle.Render("NO")
	} else {
		yesBtn = buttonStyle.Render("YES")
		noBtn = activeButtonStyle.Render("NO")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn)
	content := lipgloss.JoinVertical(lipgloss.Center, question, "\n", buttons)
	return m.centerPopup(content)
}

func (m model) viewUploadConfirmPopup() string {
	question := fmt.Sprintf("Are you sure you want to upload:\n\n'%s'?", m.pendingUploadName)
	var yesBtn, noBtn string
	if m.confirmToggle {
		yesBtn = activeButtonStyle.Render("YES")
		noBtn = buttonStyle.Render("NO")
	} else {
		yesBtn = buttonStyle.Render("YES")
		noBtn = activeButtonStyle.Render("NO")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn)
	content := lipgloss.JoinVertical(lipgloss.Center, question, "\n", buttons)
	return m.centerPopup(content)
}

func (m model) viewCollisionChoicePopup() string {
	question := fmt.Sprintf("Structure '%s' already exists. What to do?", m.pendingUploadName)
	var autoBtn, manualBtn, overBtn string
	autoBtn = buttonStyle.Render("AUTO RENAME")
	manualBtn = buttonStyle.Render("MANUAL RENAME")
	overBtn = buttonStyle.Render("OVERWRITE")

	if m.collisionChoice == 0 {
		autoBtn = activeButtonStyle.Render("AUTO RENAME")
	}
	if m.collisionChoice == 1 {
		manualBtn = activeButtonStyle.Render("MANUAL RENAME")
	}
	if m.collisionChoice == 2 {
		overBtn = activeButtonStyle.Render("OVERWRITE")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, autoBtn, manualBtn, overBtn)
	content := lipgloss.JoinVertical(lipgloss.Center, question, "\n", buttons)
	return m.centerPopup(content)
}

func (m model) viewUseChoicePopup() string {
	question := fmt.Sprintf("How do you want to use '%s'?", m.pendingUsePath)
	var dirBtn, contentBtn string
	if m.confirmToggle {
		dirBtn = activeButtonStyle.Render("AS DIRECTORY")
		contentBtn = buttonStyle.Render("CONTENTS ONLY")
	} else {
		dirBtn = buttonStyle.Render("AS DIRECTORY")
		contentBtn = activeButtonStyle.Render("CONTENTS ONLY")
	}
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
		if end > 12 {
			if m.fileCursor > 6 {
				start = m.fileCursor - 6
			}
			if start+12 < end {
				end = start + 12
			} else {
				end = len(m.flatFiles)
				if end-12 >= 0 {
					start = end - 12
				}
			}
		}
		for i := start; i < end; i++ {
			f := m.flatFiles[i]
			icon := "📄"
			if f.IsDir {
				icon = "📁"
			}
			line := fmt.Sprintf("%s %s", icon, f.Name)

			if !f.IsDir && strings.HasSuffix(f.Name, ".sh") && !m.offlineBrowserMode {
				rel := getRelPath(m.rootConstraint, f.Path)
				if _, exists := m.activeScripts[rel]; exists {
					line += lipgloss.NewStyle().Foreground(danger).Render(" [⚡]")
				}
			}

			if i == m.fileCursor {
				fileList += selectedItemStyle.Render("> " + line) + "\n"
			} else {
				fileList += itemStyle.Render("  " + line) + "\n"
			}
		}
	}
	leftBox := boxStyle.Width(halfWidth).Height(14).Render(fileList)

	displayPath := m.currentStructPath
	if !m.offlineBrowserMode {
		displayPath = strings.Replace(m.currentStructPath, m.rootConstraint, "[WORKSPACE: "+m.activeStructName+"]", 1)
	}

	metaContent := fmt.Sprintf("CURRENT LOCATION\n\n%s\n\nItems: %d", displayPath, len(m.flatFiles))
	rightBox := boxStyle.Width(halfWidth).Height(14).Render(metaContent)

	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	var fileMetaContent string
	if len(m.flatFiles) > 0 {
		f := m.flatFiles[m.fileCursor]
		extraMeta := ""

		if !f.IsDir && strings.HasSuffix(f.Name, ".sh") && !m.offlineBrowserMode {
			rel := getRelPath(m.rootConstraint, f.Path)
			if prio, exists := m.activeScripts[rel]; exists {
				extraMeta = fmt.Sprintf("\nActive: YES\nPriority: %d", prio)
			} else {
				extraMeta = "\nActive: NO"
			}
		}
		fileMetaContent = fmt.Sprintf("SELECTED INFO\n\nName: %s\nType: %s\nPermissions: %s\nSize: %d bytes%s", f.Name, func() string {
			if f.IsDir {
				return "Directory"
			}
			return "File"
		}(), f.Mode, f.Size, extraMeta)
	} else {
		fileMetaContent = "No file selected."
	}
	middleSection := boxStyle.Width(w-2).Height(6).Render(fileMetaContent)

	var keysText string
	if m.offlineBrowserMode {
		col1 := "[u] Upload Folder"
		col2 := "[Enter] Move In"
		col3 := "[Bksp] Parent Dir\n[q] End/Back"
		keysText = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width((w-4)/3).Render(col1), lipgloss.NewStyle().Width((w-4)/3).Render(col2), lipgloss.NewStyle().Width((w-4)/3).Render(col3))
	} else {
		col1 := "[c] Create File\n[C] Create Dir\n[s] Create Script\n[v/n] Edit (Vim/Nano)"
		col2 := "[r] Rename\n[d] Remove\n[Enter] Move In"
		col3 := "[a] Toggle Active Script\n[p] Set Script Priority\n[q] End/Back & Pack"
		keysText = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width((w-4)/3).Render(col1), lipgloss.NewStyle().Width((w-4)/3).Render(col2), lipgloss.NewStyle().Width((w-4)/3).Render(col3))
	}

	bottomSection := boxStyle.Width(w-2).Height(7).Render("KEY MAPPINGS\n\n" + keysText)
	return lipgloss.JoinVertical(lipgloss.Left, topSection, middleSection, bottomSection)
}

func main() {
	home, _ := os.UserHomeDir()
	rootPath = filepath.Join(home, ".struct", "structures")
	os.MkdirAll(rootPath, 0755)

	// We assign this to the global 'p' so background workers can message the UI
	p = tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
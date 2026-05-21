package session

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
)

type repoPickerMode int

const (
	repoPickerModeList repoPickerMode = iota
	repoPickerModePaste
)

// repoPicker chooses a repository for a new session. In list mode the user
// cycles through previously-used repos (recent first); a sentinel entry
// "+ New repo by path..." switches the picker to paste mode for a fresh path.
type repoPicker struct {
	mode       repoPickerMode
	projects   []domain.Project
	listIdx    int
	pasteInput textinput.Model
	focused    bool
	styles     *styles.Styles
}

func newRepoPicker(s *styles.Styles, projects []domain.Project, initialProjectID uuid.UUID, inputWidth int) repoPicker {
	sorted := append([]domain.Project(nil), projects...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
	})

	pasteInput := textinput.New()
	pasteInput.Placeholder = "/absolute/path/to/repo"
	pasteInput.CharLimit = 500
	pasteInput.SetWidth(inputWidth)
	pasteInput.SetStyles(s.Form.Input)

	listIdx := 0
	if initialProjectID != uuid.Nil {
		for i, p := range sorted {
			if p.ID == initialProjectID {
				listIdx = i
				break
			}
		}
	}

	return repoPicker{
		mode:       repoPickerModeList,
		projects:   sorted,
		listIdx:    listIdx,
		pasteInput: pasteInput,
		styles:     s,
	}
}

func (p *repoPicker) focus() {
	p.focused = true
	if p.mode == repoPickerModePaste {
		p.pasteInput.Focus()
	}
}

func (p *repoPicker) blur() {
	p.focused = false
	p.pasteInput.Blur()
}

func (p repoPicker) isPasteMode() bool {
	return p.mode == repoPickerModePaste
}

// itemCount returns total cyclable items in list mode: projects + sentinel.
func (p repoPicker) itemCount() int {
	return len(p.projects) + 1
}

// onSentinel reports whether the current list-mode selection is the
// "+ New repo by path..." sentinel entry.
func (p repoPicker) onSentinel() bool {
	return p.listIdx == len(p.projects)
}

// selectedProject returns the currently highlighted existing project, or nil
// if the sentinel is selected or there are no projects.
func (p repoPicker) selectedProject() *domain.Project {
	if p.onSentinel() {
		return nil
	}
	if p.listIdx < 0 || p.listIdx >= len(p.projects) {
		return nil
	}
	return &p.projects[p.listIdx]
}

// pastedPath returns the trimmed paste-mode input. Empty when not in paste
// mode or the field is blank.
func (p repoPicker) pastedPath() string {
	if p.mode != repoPickerModePaste {
		return ""
	}
	return strings.TrimSpace(p.pasteInput.Value())
}

func (p *repoPicker) cycle(direction int) {
	n := p.itemCount()
	if n <= 0 {
		return
	}
	p.listIdx = ((p.listIdx+direction)%n + n) % n
}

// enterPasteMode switches to paste mode and focuses the text input. The
// caller is responsible for routing subsequent key events here.
func (p *repoPicker) enterPasteMode() {
	p.mode = repoPickerModePaste
	p.pasteInput.SetValue("")
	p.pasteInput.Focus()
}

func (p *repoPicker) exitPasteMode() {
	p.mode = repoPickerModeList
	p.pasteInput.Blur()
}

// adoptRegisteredProject appends a newly-registered project to the list and
// selects it, returning the picker to list mode. Called by the create form
// after silently registering a path the user pasted.
func (p *repoPicker) adoptRegisteredProject(project domain.Project) {
	p.projects = append([]domain.Project{project}, p.projects...)
	p.listIdx = 0
	p.exitPasteMode()
}

func (p repoPicker) update(msg tea.Msg) (repoPicker, tea.Cmd) {
	if !p.focused {
		return p, nil
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if p.mode == repoPickerModeList {
			if key.Matches(keyMsg, popupSelectorNextKeyBinding) {
				p.cycle(1)
				return p, nil
			}
			if key.Matches(keyMsg, popupSelectorPrevKeyBinding) {
				p.cycle(-1)
				return p, nil
			}
			if key.Matches(keyMsg, repoPickerEnterPasteKeyBinding) {
				p.enterPasteMode()
				return p, textinput.Blink
			}
			return p, nil
		}

		if key.Matches(keyMsg, repoPickerExitPasteKeyBinding) {
			p.exitPasteMode()
			return p, nil
		}
	}

	if p.mode == repoPickerModePaste {
		var cmd tea.Cmd
		p.pasteInput, cmd = p.pasteInput.Update(msg)
		return p, cmd
	}
	return p, nil
}

func (p repoPicker) view() string {
	if p.mode == repoPickerModePaste {
		return p.pasteInput.View()
	}

	if p.itemCount() == 0 {
		return p.styles.ListRow.Normal.Render("  (no repos yet) — press p to paste a path  ")
	}

	label := p.currentLabel()
	if p.focused {
		return p.styles.ListRow.Selected.Render("< " + label + " >")
	}
	return p.styles.ListRow.Normal.Render("  " + label + "  ")
}

func (p repoPicker) currentLabel() string {
	if proj := p.selectedProject(); proj != nil {
		return proj.Name
	}
	return "+ New repo by path..."
}

// resolvedSelection describes what the picker currently points at — either an
// existing registered project, a freshly-typed path that must be registered
// before use, or nothing usable.
type resolvedSelection struct {
	Project *domain.Project
	NewPath string
}

func (s resolvedSelection) IsZero() bool {
	return s.Project == nil && s.NewPath == ""
}

func (p repoPicker) resolve() resolvedSelection {
	if p.mode == repoPickerModePaste {
		if path := p.pastedPath(); path != "" {
			return resolvedSelection{NewPath: path}
		}
		return resolvedSelection{}
	}
	if proj := p.selectedProject(); proj != nil {
		return resolvedSelection{Project: proj}
	}
	return resolvedSelection{}
}

// projectIDForCreate returns the project ID to use in CreateSessionRequest
// when the resolution is an existing project. Returns uuid.Nil otherwise.
func (s resolvedSelection) projectIDForCreate() uuid.UUID {
	if s.Project == nil {
		return uuid.Nil
	}
	return s.Project.ID
}

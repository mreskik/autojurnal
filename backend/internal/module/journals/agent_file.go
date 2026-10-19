package journals

import (
	"bytes"
	"context"
	_ "embed"
	"text/template"
	"time"

	"github.com/google/uuid"
)

//go:embed templates/agent.md.tmpl
var agentTemplateText string

var agentTemplate = template.Must(template.New("agent.md").Parse(agentTemplateText))

type agentTemplateData struct {
	JournalName  string
	RunnerName   string
	RunnerID     string
	RunnerToken  string
	JournalToken string
	BaseURL      string
	GeneratedAt  string
}

// AgentFile membuat isi AGENT.md untuk runner di jurnal ini (berisi token runner & token jurnal).
// baseURL = URL publik API (contoh https://api.domain.com/api).
func (s *Service) AgentFile(ctx context.Context, userID, journalID, runnerID uuid.UUID, baseURL string) ([]byte, error) {
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return nil, err
	}
	// runner harus join ke jurnal ini
	if _, err := s.repo.FindJoinedRunner(ctx, journalID, runnerID); err != nil {
		return nil, err
	}
	runner, err := s.runnerRepo.FindByUser(ctx, userID, runnerID)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = agentTemplate.Execute(&buf, agentTemplateData{
		JournalName:  j.Name,
		RunnerName:   runner.Name,
		RunnerID:     runner.ID.String(),
		RunnerToken:  runner.Token,
		JournalToken: j.Token,
		BaseURL:      baseURL,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
	})
	return buf.Bytes(), err
}

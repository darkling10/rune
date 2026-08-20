package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/deployerai/deployer/pkg/ai"
)

// Runner represents the pipeline execution engine.
type Runner struct {
	Config    *Config
	WorkDir   string
	AIKey     string
	AIProv    string
	AIBaseURL string
}

// NewRunner initializes a new Runner.
func NewRunner(config *Config, workDir string, aiProv, aiKey, aiBaseURL string) *Runner {
	return &Runner{
		Config:    config,
		WorkDir:   workDir,
		AIProv:    aiProv,
		AIKey:     aiKey,
		AIBaseURL: aiBaseURL,
	}
}

// Execute loops through the pipeline steps and executes them sequentially.
func (r *Runner) Execute() error {
	log.Printf("Starting execution of pipeline version: %s in workspace: %s", r.Config.Version, r.WorkDir)

	for i, step := range r.Config.Pipeline {
		log.Printf("--- Step %d: %s ---", i+1, step.Name)

		if step.Run != "" {
			if err := r.executeShellCommand(step); err != nil {
				return fmt.Errorf("step '%s' failed: %w", step.Name, err)
			}
		} else if step.Uses != "" {
			if err := r.executeUsesAction(step); err != nil {
				return fmt.Errorf("step '%s' failed: %w", step.Name, err)
			}
		} else {
			log.Printf("Warning: Step '%s' has neither 'run' nor 'uses' defined. Skipping.", step.Name)
		}
	}

	log.Println("Pipeline execution completed successfully.")
	return nil
}

func (r *Runner) executeShellCommand(step Step) error {
	log.Printf("Executing shell command: %s %v", step.Run, step.Args)

	// In a real environment, we'd want this to be running inside a secure container (e.g. Docker API)
	// For now, we execute on the host in the temporary cloned directory.
	cmd := exec.Command(step.Run, step.Args...)
	cmd.Dir = r.WorkDir

	// Stream output to stdout/stderr of the worker
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func (r *Runner) executeUsesAction(step Step) error {
	log.Printf("Executing built-in AI/Action: %s", step.Uses)

	if step.Uses == "rune/ai-review@v1" {
		log.Println("=> Initiating AI Code Review...")
		
		if r.AIKey == "" {
			return fmt.Errorf("AI Credentials not found for project. Cannot perform AI Review")
		}

		// 1. Fetch git diff
		cmd := exec.Command("git", "diff", "HEAD~1", "HEAD")
		cmd.Dir = r.WorkDir
		out, err := cmd.Output()
		if err != nil {
			// If HEAD~1 fails (e.g. initial commit), try getting diff against empty tree
			cmd = exec.Command("git", "show", "HEAD")
			cmd.Dir = r.WorkDir
			out, err = cmd.Output()
			if err != nil {
				return fmt.Errorf("failed to fetch git diff: %w", err)
			}
		}

		diff := string(out)
		if len(diff) == 0 {
			log.Println("=> No code changes detected. Skipping AI review.")
			return nil
		}

		log.Printf("=> Analyzing %d bytes of diff with %s...", len(diff), r.AIProv)

		// 2. We will call the AI provider package here.
		provider, err := ai.Factory(r.AIProv, r.AIKey, r.AIBaseURL)
		if err != nil {
			return fmt.Errorf("failed to initialize AI Provider: %w", err)
		}

		isPass, reason, err := provider.ReviewCodeDiff(context.Background(), diff)
		if err != nil {
			return fmt.Errorf("AI Review failed unexpectedly: %w", err)
		}

		log.Printf("=> AI Review Result: %s", reason)
		if !isPass {
			return fmt.Errorf("pipeline aborted: AI Review rejected the code changes")
		}
		
	} else {
		log.Printf("Unknown action: %s", step.Uses)
	}

	return nil
}

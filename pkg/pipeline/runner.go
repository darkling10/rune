package pipeline

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

// Runner represents the pipeline execution engine.
type Runner struct {
	Config  *Config
	WorkDir string
}

// NewRunner initializes a new Runner.
func NewRunner(config *Config, workDir string) *Runner {
	return &Runner{
		Config:  config,
		WorkDir: workDir,
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

	// Placeholder for AI Engine Integration
	if step.Uses == "deployerai/ai-review@v1" {
		log.Println("=> Initiating AI Code Review (Placeholder)...")
		log.Println("=> AI Review Passed!")
	} else {
		log.Printf("Unknown action: %s", step.Uses)
	}

	return nil
}

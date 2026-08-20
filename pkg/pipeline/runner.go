package pipeline

import (
	"context"
	"fmt"
	"io"
	"log"
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
	Logger    io.Writer
}

// NewRunner initializes a new Runner.
func NewRunner(config *Config, workDir string, aiProv, aiKey, aiBaseURL string, logger io.Writer) *Runner {
	return &Runner{
		Config:    config,
		WorkDir:   workDir,
		AIProv:    aiProv,
		AIKey:     aiKey,
		AIBaseURL: aiBaseURL,
		Logger:    logger,
	}
}

// Execute loops through the pipeline steps and executes them sequentially.
func (r *Runner) Execute() error {
	msg := fmt.Sprintf("Starting execution of pipeline version: %s in workspace: %s\n", r.Config.Version, r.WorkDir)
	log.Print(msg)
	fmt.Fprint(r.Logger, msg)

	for i, step := range r.Config.Pipeline {
		stepMsg := fmt.Sprintf("--- Step %d: %s ---\n", i+1, step.Name)
		log.Print(stepMsg)
		fmt.Fprint(r.Logger, stepMsg)

		if step.Run != "" {
			if err := r.executeShellCommand(step); err != nil {
				errMsg := fmt.Sprintf("step '%s' failed: %v\n", step.Name, err)
				fmt.Fprint(r.Logger, errMsg)
				return fmt.Errorf("step '%s' failed: %w", step.Name, err)
			}
		} else if step.Uses != "" {
			if err := r.executeUsesAction(step); err != nil {
				errMsg := fmt.Sprintf("step '%s' failed: %v\n", step.Name, err)
				fmt.Fprint(r.Logger, errMsg)
				return fmt.Errorf("step '%s' failed: %w", step.Name, err)
			}
		} else {
			warnMsg := fmt.Sprintf("Warning: Step '%s' has neither 'run' nor 'uses' defined. Skipping.\n", step.Name)
			log.Print(warnMsg)
			fmt.Fprint(r.Logger, warnMsg)
		}
	}

	successMsg := "Pipeline execution completed successfully.\n"
	log.Print(successMsg)
	fmt.Fprint(r.Logger, successMsg)
	return nil
}

func (r *Runner) executeShellCommand(step Step) error {
	log.Printf("Executing shell command: %s %v", step.Run, step.Args)

	// In a real environment, we'd want this to be running inside a secure container (e.g. Docker API)
	// For now, we execute on the host in the temporary cloned directory.
	cmd := exec.Command(step.Run, step.Args...)
	cmd.Dir = r.WorkDir

	// Stream output to stdout/stderr of the worker
	cmd.Stdout = r.Logger
	cmd.Stderr = r.Logger

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func (r *Runner) executeUsesAction(step Step) error {
	msg := fmt.Sprintf("Executing built-in AI/Action: %s\n", step.Uses)
	log.Print(msg)
	fmt.Fprint(r.Logger, msg)

	if step.Uses == "rune/ai-review@v1" {
		initMsg := "=> Initiating AI Code Review...\n"
		log.Print(initMsg)
		fmt.Fprint(r.Logger, initMsg)
		
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
			skipMsg := "=> No code changes detected. Skipping AI review.\n"
			log.Print(skipMsg)
			fmt.Fprint(r.Logger, skipMsg)
			return nil
		}

		analyzeMsg := fmt.Sprintf("=> Analyzing %d bytes of diff with %s...\n", len(diff), r.AIProv)
		log.Print(analyzeMsg)
		fmt.Fprint(r.Logger, analyzeMsg)

		// 2. We will call the AI provider package here.
		provider, err := ai.Factory(r.AIProv, r.AIKey, r.AIBaseURL)
		if err != nil {
			return fmt.Errorf("failed to initialize AI Provider: %w", err)
		}

		isPass, reason, err := provider.ReviewCodeDiff(context.Background(), diff)
		if err != nil {
			return fmt.Errorf("AI Review failed unexpectedly: %w", err)
		}

		resMsg := fmt.Sprintf("=> AI Review Result: %s\n", reason)
		log.Print(resMsg)
		fmt.Fprint(r.Logger, resMsg)
		
		if !isPass {
			return fmt.Errorf("pipeline aborted: AI Review rejected the code changes")
		}
		
	} else {
		unMsg := fmt.Sprintf("Unknown action: %s\n", step.Uses)
		log.Print(unMsg)
		fmt.Fprint(r.Logger, unMsg)
	}

	return nil
}

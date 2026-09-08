//go:build integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

// This suite deliberately exercises the built CLI: podman-compose children must
// enter the native bridge in main, which invoking New from a test cannot prove.
type podmanIntegrationCLI struct{ executable string }

func (c podmanIntegrationCLI) invoke(args ...string) (json.RawMessage, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, c.executable, append([]string{"--output=json", "--owner=integration"}, args...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var envelope struct {
		SchemaVersion int             `json:"schema_version"`
		Data          json.RawMessage `json:"data"`
	}
	if stdout.Len() != 0 {
		if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
			return nil, stderr.String(), fmt.Errorf("invalid CLI JSON: %w", decodeErr)
		}
		if envelope.SchemaVersion != 1 {
			return nil, stderr.String(), fmt.Errorf("schema version %d", envelope.SchemaVersion)
		}
	}
	return envelope.Data, stderr.String(), err
}

func podmanIntegrationJSON[T any](t *testing.T, c podmanIntegrationCLI, args ...string) T {
	t.Helper()
	data, stderr, err := c.invoke(args...)
	if err != nil {
		t.Fatalf("CLI %v: %v\n%s\n%s", args, err, stderr, data)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func (c podmanIntegrationCLI) cleanupLease(t *testing.T, lease domain.Lease) {
	t.Helper()
	if lease.ID == "" {
		return
	}
	t.Cleanup(func() {
		data, stderr, err := c.invoke("destroy", lease.ID, "--force")
		if err != nil {
			t.Errorf("cleanup %s: %v\n%s\n%s", lease.ID, err, stderr, data)
		}
	})
}

func podmanIntegrationFixture(t *testing.T) (podmanIntegrationCLI, string, *config.Manifest, string) {
	t.Helper()
	if os.Getenv("AGENT_ENV_PODMAN_INTEGRATION") != "1" {
		t.Skip("explicit rootless Podman integration: set AGENT_ENV_PODMAN_INTEGRATION=1")
	}
	if runtime.GOOS != "linux" {
		t.Fatal("this acceptance suite requires native Linux rootless Podman; Machine evidence is separate")
	}
	// Pin these acceptance observations to the local engine, independent of user
	// defaults. No connection definitions or global Podman settings are modified.
	for _, name := range []string{"CONTAINER_HOST", "CONTAINER_CONNECTION", "CONTAINER_SSHKEY"} {
		t.Setenv(name, "")
	}
	infoJSON := integrationCommand(t, "", "podman", "--remote=false", "info", "--format", "json")
	var info struct {
		Host struct {
			Security struct {
				Rootless bool `json:"rootless"`
			} `json:"security"`
		} `json:"host"`
	}
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		t.Fatal(err)
	}
	if !info.Host.Security.Rootless {
		t.Fatal("real acceptance requires rootless Podman")
	}
	t.Logf("Podman version: %s", integrationCommand(t, "", "podman", "--remote=false", "version", "--format", "json"))
	t.Logf("podman-compose version: %s", integrationCommand(t, "", "podman-compose", "--version"))
	base, err := os.MkdirTemp("", "agent-env-podman-acceptance-")
	if err != nil {
		t.Fatal(err)
	}
	var resourceCleanup []func() error
	home := filepath.Join(base, "state space 日本語")
	defer func() {
		if t.Failed() {
			t.Logf("preserving failed Podman fixture evidence: %s", base)
		}
	}()
	executable := filepath.Join(base, "agent-env")
	cli := podmanIntegrationCLI{executable: executable}
	t.Setenv("AGENT_ENV_HOME", home)
	// Register before all image/volume effects. Registry fallback does not depend
	// on create emitting JSON before a process timeout or abnormal termination.
	t.Cleanup(func() {
		clean := true
		if _, err := os.Stat(filepath.Join(home, "state.db")); err == nil {
			data, stderr, err := cli.invoke("list", "--cached")
			var leases []domain.Lease
			if err == nil {
				err = json.Unmarshal(data, &leases)
			}
			if err != nil {
				t.Errorf("fixture registry recovery: %v %s", err, stderr)
				clean = false
			}
			for _, lease := range leases {
				data, stderr, err := cli.invoke("destroy", lease.ID, "--force")
				var released domain.Lease
				if err == nil {
					err = json.Unmarshal(data, &released)
				}
				if err != nil || released.Observed != "released" {
					t.Errorf("fixture registry cleanup %s: %v state=%s %s", lease.ID, err, released.Observed, stderr)
					clean = false
				}
			}
		} else if !os.IsNotExist(err) {
			t.Errorf("fixture registry status: %v", err)
			clean = false
		}
		// Images and foreign fixture volumes follow lease cleanup, including when
		// create never returned its identity. Uncertain leases retain all inputs.
		if clean {
			for i := len(resourceCleanup) - 1; i >= 0; i-- {
				if err := resourceCleanup[i](); err != nil {
					t.Errorf("fixture resource cleanup: %v", err)
					clean = false
				}
			}
		}
		if !clean || t.Failed() {
			t.Logf("preserving failed Podman fixture evidence: %s", base)
			return
		}
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("remove completed fixture: %v", err)
		}
	})
	integrationCommand(t, filepath.Join("..", ".."), "go", "build", "-o", executable, "./cmd/agent-env")
	t.Setenv("FIXTURE_SECRET_TOKEN", "integration-credential-value-7f032")
	repo := filepath.Join(base, "repository space 日本語")
	if err := os.MkdirAll(filepath.Join(repo, "www"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"compose.yaml", "www/index.html"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "compose", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		// Qualify the same fixture image to avoid an interactive short-name registry
		// prompt. Add an anonymous database volume to expose down cleanup differences.
		if name == "compose.yaml" {
			data = []byte(strings.ReplaceAll(string(data), "busybox:1.37", "docker.io/library/busybox:1.37"))
			data = []byte(strings.Replace(string(data), "volumes: [fixture-data:/data]", "volumes: [fixture-data:/data, /anonymous-fixture]", 1))
		}
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(name)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("tracked fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := &config.Manifest{Version: 1, Sources: map[string]config.Source{"self": {Repository: ".", DefaultRef: "main"}}, Runtimes: map[string]config.Runtime{"compose": {Type: "compose", Provider: "podman-compose", Source: "self", ProjectDirectory: ".", Files: []string{"compose.yaml"}}}, Components: map[string]config.Component{"database": {Runtime: "compose", ComposeServices: []string{"database"}}, "api": {Runtime: "compose", ComposeServices: []string{"api"}, DependsOn: []string{"database"}, Provides: []string{"api"}}, "dashboard": {Runtime: "compose", ComposeServices: []string{"dashboard"}, DependsOn: []string{"api"}, Provides: []string{"dashboard"}}}, Stacks: map[string]config.Stack{"api": {Roots: []string{"api"}}, "dashboard": {Roots: []string{"dashboard"}}}, Tests: map[string]config.Test{}}
	for name, target := range map[string]int{"api": 8080, "dashboard": 8081} {
		component := m.Components[name]
		component.Endpoints = map[string]config.Endpoint{"http": {Service: name, Target: target, Protocol: "tcp"}}
		m.Components[name] = component
	}
	for _, mode := range []string{"pass", "fail"} {
		m.Tests[mode] = config.Test{Stack: "api", Source: "self", WorkingDirectory: ".", Command: []string{os.Args[0], "-test.run=^TestIntegrationCommandHelper$", "--", mode}, Env: map[string]string{"AGENT_ENV_FIXTURE_HELPER": "1", "TEST_TOKEN": "${env:FIXTURE_SECRET_TOKEN}"}, Timeout: "20s", Artifacts: []string{"report.txt"}}
	}
	writeIntegrationManifest(t, repo, m)
	integrationCommand(t, repo, "git", "-c", "init.templateDir=", "init", "--initial-branch=main")
	integrationCommit(t, repo)
	// Doctor must enforce the supported provider floor before any engine resource
	// is created; a missing or too-old dependency is a failure, never a skip.
	podmanIntegrationJSON[json.RawMessage](t, cli, "doctor", repo)
	imageToken := fmt.Sprintf("agent-env-image-fixture-%d-%d", os.Getpid(), time.Now().UnixNano())
	imageTag := "localhost/" + imageToken + ":fixture"
	imageDirectory := filepath.Join(base, "image build space 日本語")
	if err := os.MkdirAll(imageDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDirectory, "Dockerfile"), []byte("FROM docker.io/library/busybox:1.37\nVOLUME /image-anonymous\n"), 0600); err != nil {
		t.Fatal(err)
	}
	podmanIntegrationBuildImage(t, "podman", []string{"--remote=false"}, imageDirectory, imageTag, imageToken, &resourceCleanup)
	if os.Getenv("AGENT_ENV_PODMAN_DOCKER_COEXISTENCE") == "1" {
		integrationCommand(t, "", "docker", "info", "--format", "{{.ServerVersion}}")
		integrationCommand(t, "", "docker", "compose", "version", "--short")
		podmanIntegrationBuildImage(t, "docker", nil, imageDirectory, imageTag, imageToken, &resourceCleanup)
	}
	composePath := filepath.Join(repo, "compose.yaml")
	composeData, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	composeText := strings.Replace(string(composeData), "database:\n    image: docker.io/library/busybox:1.37", "database:\n    image: "+imageTag, 1)
	if composeText == string(composeData) {
		t.Fatal("database fixture image was not replaced")
	}
	if err := os.WriteFile(composePath, []byte(composeText), 0600); err != nil {
		t.Fatal(err)
	}
	integrationCommit(t, repo)
	foreign := fmt.Sprintf("agent-env-podman-integration-foreign-%d-%d", os.Getpid(), time.Now().UnixNano())
	t.Setenv("AGENT_ENV_TEST_FOREIGN_VOLUME", foreign)
	resourceCleanup = append(resourceCleanup, func() error {
		listing, err := podmanIntegrationCleanupCommand("podman", "--remote=false", "volume", "ls", "--quiet")
		if err != nil {
			return err
		}
		found := false
		for _, name := range strings.Fields(listing) {
			if name == foreign {
				found = true
			}
		}
		if !found {
			return nil
		}
		label, err := podmanIntegrationCleanupCommand("podman", "--remote=false", "volume", "inspect", foreign, "--format", `{{index .Labels "io.agent-env.integration-fixture"}}`)
		if err != nil {
			return err
		}
		if label != foreign {
			return fmt.Errorf("foreign fixture volume ownership changed: %q", label)
		}
		_, err = podmanIntegrationCleanupCommand("podman", "--remote=false", "volume", "rm", foreign)
		return err
	})
	integrationCommand(t, "", "podman", "--remote=false", "volume", "create", "--label", "io.agent-env.integration-fixture="+foreign, foreign)
	return cli, repo, m, foreign
}

func podmanIntegrationEndpoints(t *testing.T, cli podmanIntegrationCLI, lease domain.Lease) {
	t.Helper()
	capabilities := podmanIntegrationJSON[struct {
		Endpoints map[string]string `json:"endpoints"`
	}](t, cli, "capabilities", lease.ID)
	names := []string{"api.http"}
	if lease.Stack == "dashboard" {
		names = append(names, "dashboard.http")
	}
	for _, name := range names {
		address := capabilities.Endpoints[name]
		if !strings.HasPrefix(address, "127.0.0.1:") || strings.HasSuffix(address, ":0") {
			t.Fatalf("dynamic endpoint %s = %q", name, address)
		}
		response, err := (&http.Client{Timeout: 10 * time.Second}).Get("http://" + address + "/")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("endpoint %s HTTP %d", name, response.StatusCode)
		}
	}
}

func podmanIntegrationNoResiduals(t *testing.T, lease domain.Lease, attachedVolumes []string) {
	t.Helper()
	project := lease.Runtimes[0].Project
	for _, args := range [][]string{
		{"ps", "--all", "--quiet", "--filter", "label=io.podman.compose.project=" + project},
		{"network", "ls", "--quiet", "--filter", "label=io.podman.compose.project=" + project},
		{"volume", "ls", "--quiet", "--filter", "label=io.podman.compose.project=" + project},
	} {
		if remaining := integrationCommand(t, "", "podman", append([]string{"--remote=false"}, args...)...); remaining != "" {
			t.Fatalf("residual project resources %v: %s", args, remaining)
		}
	}
	// Listing must succeed; an inspect error alone could mean an unavailable
	// engine rather than successful cleanup. Include unlabelled anonymous volumes.
	volumes := strings.Fields(integrationCommand(t, "", "podman", "--remote=false", "volume", "ls", "--quiet"))
	for _, removed := range attachedVolumes {
		for _, current := range volumes {
			if removed == current {
				t.Fatalf("attached volume survived cleanup: %s", removed)
			}
		}
	}
}

func TestPodmanIntegrationConcurrentLeasesAndEvidence(t *testing.T) {
	cli, repo, m, foreign := podmanIntegrationFixture(t)
	type result struct {
		lease  domain.Lease
		err    error
		stderr string
	}
	results := make(chan result, 2)
	for _, stack := range []string{"api", "dashboard"} {
		go func(stack string) {
			data, stderr, err := cli.invoke("create", repo, "--stack", stack)
			var lease domain.Lease
			if len(data) > 0 {
				if decodeErr := json.Unmarshal(data, &lease); decodeErr != nil {
					err = decodeErr
				}
			}
			results <- result{lease, err, stderr}
		}(stack)
	}
	leases := map[string]domain.Lease{}
	for range 2 {
		result := <-results
		cli.cleanupLease(t, result.lease)
		if result.err != nil {
			t.Errorf("concurrent create: %v\n%s", result.err, result.stderr)
		}
		leases[result.lease.Stack] = result.lease
	}
	if t.Failed() {
		t.FailNow()
	}
	api, dashboard := leases["api"], leases["dashboard"]
	if api.Observed != "ready" || dashboard.Observed != "ready" || len(api.Runtimes) != 1 || len(dashboard.Runtimes) != 1 {
		t.Fatalf("leases not ready: %+v %+v", api, dashboard)
	}
	if api.Runtimes[0].Provider != domain.ComposeProviderPodman || dashboard.Runtimes[0].Provider != domain.ComposeProviderPodman || api.Runtimes[0].Project == dashboard.Runtimes[0].Project || api.Sources[0].WorktreePath == dashboard.Sources[0].WorktreePath || api.Sources[0].Commit != dashboard.Sources[0].Commit {
		t.Fatal("provider or lease identity not isolated")
	}
	if len(api.Runtimes[0].Services) != 2 || len(dashboard.Runtimes[0].Services) != 3 {
		t.Fatal("selected service closure differs from Docker fixture")
	}
	t.Logf("real concurrent Podman leases %s and %s", api.ID, dashboard.ID)
	attachedVolumes := podmanIntegrationAttachedVolumes(t, api)
	siblingVolumes := podmanIntegrationAttachedVolumes(t, dashboard)
	podmanIntegrationEndpoints(t, cli, api)
	podmanIntegrationEndpoints(t, cli, dashboard)
	for _, component := range []string{"api", "dashboard"} {
		logs := podmanIntegrationJSON[struct {
			Logs map[string]string `json:"logs"`
		}](t, cli, "logs", dashboard.ID, "--component", component)
		combined := ""
		for _, value := range logs.Logs {
			combined += value
		}
		if !strings.Contains(combined, component) {
			t.Fatalf("component log attribution absent: %q", combined)
		}
	}
	podmanIntegrationNamedTests(t, cli, api)
	var docker domain.Lease
	if os.Getenv("AGENT_ENV_PODMAN_DOCKER_COEXISTENCE") == "1" {
		integrationCommand(t, "", "docker", "info", "--format", "{{.ServerVersion}}")
		integrationCommand(t, "", "docker", "compose", "version", "--short")
		selected := m.Runtimes["compose"]
		selected.Provider = "docker-compose"
		m.Runtimes["compose"] = selected
		writeIntegrationManifest(t, repo, m)
		integrationCommit(t, repo)
		data, stderr, err := cli.invoke("create", repo, "--stack", "api")
		if len(data) > 0 {
			if decodeErr := json.Unmarshal(data, &docker); decodeErr != nil {
				t.Fatal(decodeErr)
			}
		}
		cli.cleanupLease(t, docker)
		if err != nil {
			t.Fatalf("Docker coexistence create: %v %s", err, stderr)
		}
		if docker.Observed != "ready" || docker.Runtimes[0].Provider != domain.ComposeProviderDocker {
			t.Fatal("Docker coexistence not ready")
		}
		podmanIntegrationEndpoints(t, cli, docker)
		podmanIntegrationNamedTests(t, cli, docker)
	} else {
		t.Log("Docker coexistence not requested; set AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1 for this separate acceptance")
	}
	before := strings.Join(resourceIDs(dashboard), "\n")
	released := podmanIntegrationJSON[domain.Lease](t, cli, "destroy", api.ID)
	if released.Observed != "released" {
		t.Fatalf("destroy state %s", released.Observed)
	}
	podmanIntegrationNoResiduals(t, api, attachedVolumes)
	if got := integrationCommand(t, "", "podman", "--remote=false", "volume", "inspect", foreign, "--format", "{{.Name}}"); got != foreign {
		t.Fatal("foreign volume changed")
	}
	sibling := podmanIntegrationJSON[integrationShow](t, cli, "show", dashboard.ID).Lease
	if sibling.Observed != "ready" || strings.Join(resourceIDs(sibling), "\n") != before {
		t.Fatal("destroy changed sibling resources")
	}
	podmanIntegrationEndpoints(t, cli, dashboard)
	if docker.ID != "" {
		live := podmanIntegrationJSON[integrationShow](t, cli, "show", docker.ID).Lease
		if live.Observed != "ready" {
			t.Fatal("Podman destroy changed Docker lease")
		}
		podmanIntegrationEndpoints(t, cli, docker)
	}
	released = podmanIntegrationJSON[domain.Lease](t, cli, "destroy", dashboard.ID)
	if released.Observed != "released" {
		t.Fatalf("sibling destroy state %s", released.Observed)
	}
	podmanIntegrationNoResiduals(t, dashboard, siblingVolumes)
}

func podmanIntegrationNamedTests(t *testing.T, cli podmanIntegrationCLI, lease domain.Lease) {
	t.Helper()
	run := podmanIntegrationJSON[domain.CommandRun](t, cli, "test", lease.ID, "pass")
	if run.Status != "passed" || run.ExitCode != 0 {
		t.Fatalf("named pass: %+v", run)
	}
	for _, name := range []string{run.StdoutPath, run.StderrPath} {
		data, err := os.ReadFile(name)
		if err != nil || !strings.Contains(string(data), "[REDACTED]") || strings.Contains(string(data), os.Getenv("FIXTURE_SECRET_TOKEN")) {
			t.Fatalf("redaction failed: %s %v", name, err)
		}
	}
	data, stderr, err := cli.invoke("test", lease.ID, "fail")
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 5 {
		t.Fatalf("named failure CLI exit: %v %s", err, stderr)
	}
	var failed domain.CommandRun
	if err := json.Unmarshal(data, &failed); err != nil {
		t.Fatal(err)
	}
	if failed.Status != "failed" || failed.ExitCode != 9 {
		t.Fatalf("named failure: %+v", failed)
	}
	show := podmanIntegrationJSON[integrationShow](t, cli, "show", lease.ID)
	if len(show.Runs) != 2 || len(show.Artifacts) < 8 || len(show.Events) == 0 {
		t.Fatal("named test evidence incomplete")
	}
}

// Build only a uniquely labelled fixture tag and retain immutable image identity.
// Cleanup never prunes images, force-removes a used image, or removes a moved tag.
func podmanIntegrationBuildImage(t *testing.T, engine string, global []string, directory, tag, token string, cleanups *[]func() error) {
	t.Helper()
	command := func(args ...string) string {
		return integrationCommand(t, "", engine, append(append([]string(nil), global...), args...)...)
	}
	for _, existing := range strings.Fields(command("image", "ls", "--format", "{{.Repository}}:{{.Tag}}")) {
		if existing == tag {
			t.Fatalf("refusing to overwrite existing %s image tag %s", engine, tag)
		}
	}
	iidFile := filepath.Join(directory, engine+"-image-id")
	*cleanups = append(*cleanups, func() error {
		data, err := os.ReadFile(iidFile)
		if os.IsNotExist(err) {
			return nil
		} // No recorded image identity; do not guess.
		if err != nil {
			return fmt.Errorf("read %s fixture image identity: %w", engine, err)
		}
		expected := strings.TrimSpace(string(data))
		if expected == "" {
			return fmt.Errorf("empty %s fixture image identity; retaining image", engine)
		}
		var images []struct {
			ID     string `json:"Id"`
			Config struct{ Labels map[string]string }
		}
		raw, err := podmanIntegrationCleanupCommand(engine, append(append([]string(nil), global...), "image", "inspect", tag)...)
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(raw), &images); err != nil {
			return fmt.Errorf("inspect %s fixture image: %w", engine, err)
		}
		if len(images) != 1 || strings.TrimPrefix(images[0].ID, "sha256:") != strings.TrimPrefix(expected, "sha256:") || images[0].Config.Labels["io.agent-env.integration-fixture"] != token {
			return fmt.Errorf("%s fixture image identity changed; retaining %s", engine, tag)
		}
		_, err = podmanIntegrationCleanupCommand(engine, append(append([]string(nil), global...), "image", "rm", tag)...)
		return err
	})
	command("build", "--label", "io.agent-env.integration-fixture="+token, "--iidfile", iidFile, "--tag", tag, directory)
}

func podmanIntegrationAttachedVolumes(t *testing.T, lease domain.Lease) []string {
	t.Helper()
	var attachedVolumes []string
	anonymous := false
	for _, resource := range lease.Resources {
		if resource.Kind != "container" {
			continue
		}
		var observations []struct {
			Mounts []struct{ Type, Name, Destination string }
		}
		raw := integrationCommand(t, "", "podman", "--remote=false", "inspect", resource.ExternalID)
		if err := json.Unmarshal([]byte(raw), &observations); err != nil {
			t.Fatal(err)
		}
		for _, observation := range observations {
			for _, mount := range observation.Mounts {
				if mount.Type == "volume" {
					attachedVolumes = append(attachedVolumes, mount.Name)
					if mount.Destination == "/image-anonymous" {
						var volumes []struct {
							Anonymous bool
							Labels    map[string]string
						}
						raw := integrationCommand(t, "", "podman", "--remote=false", "volume", "inspect", mount.Name)
						if err := json.Unmarshal([]byte(raw), &volumes); err != nil {
							t.Fatal(err)
						}
						if len(volumes) != 1 || !volumes[0].Anonymous || len(volumes[0].Labels) != 0 {
							t.Fatalf("image VOLUME did not create an unlabelled native anonymous volume: %+v", volumes)
						}
						anonymous = true
					}
				}
			}
		}
	}
	if !anonymous {
		t.Fatal("fixture did not create observable image-declared native anonymous volume")
	}
	return attachedVolumes
}

func podmanIntegrationCleanupCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	data, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %v: %w: %s", name, args, err, data)
	}
	return strings.TrimSpace(string(data)), nil
}

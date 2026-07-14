package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type kindImageState struct {
	reference     string
	importAliases []string
}

type imageIndex struct {
	Manifests []struct {
		Digest string `json:"digest"`
	} `json:"manifests"`
}

func readKindImageState(ctx context.Context, node string, image string) (kindImageState, error) {
	output, err := exec.CommandContext(ctx, "docker", "exec", node, "ctr", "-n", "k8s.io", "images", "ls").Output()
	if err != nil {
		return kindImageState{}, fmt.Errorf("list images on kind node %s: %w", node, err)
	}
	ref := containerdImageReference(image)
	digest := ""
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == ref {
			digest = fields[2]
			break
		}
	}
	if digest == "" {
		return kindImageState{}, nil
	}
	importDigests := make(map[string]string)
	sharedByAnotherTag := false
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] == ref {
			continue
		}
		if strings.HasPrefix(fields[0], "import-") {
			importDigests[fields[0]] = fields[2]
			continue
		}
		if fields[2] == digest && !strings.HasPrefix(fields[0], "sha256:") {
			sharedByAnotherTag = true
		}
	}
	if sharedByAnotherTag {
		return kindImageState{reference: ref}, nil
	}
	aliases := make([]string, 0)
	for alias, aliasDigest := range importDigests {
		if aliasDigest == digest {
			aliases = append(aliases, alias)
			continue
		}
		wrapped, err := importIndexContains(ctx, node, aliasDigest, digest)
		if err != nil {
			return kindImageState{}, err
		}
		if wrapped {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return kindImageState{reference: ref, importAliases: aliases}, nil
}

func importIndexContains(ctx context.Context, node string, indexDigest string, targetDigest string) (bool, error) {
	output, err := exec.CommandContext(ctx, "docker", "exec", node, "ctr", "-n", "k8s.io", "content", "get", indexDigest).Output()
	if err != nil {
		return false, fmt.Errorf("read imported image index %s on kind node %s: %w", indexDigest, node, err)
	}
	var index imageIndex
	if err := json.Unmarshal(output, &index); err != nil {
		return false, fmt.Errorf("parse imported image index %s on kind node %s: %w", indexDigest, node, err)
	}
	for _, manifest := range index.Manifests {
		if manifest.Digest == targetDigest {
			return true, nil
		}
	}
	return false, nil
}

func containerdImageReference(image string) string {
	first, _, _ := strings.Cut(image, "/")
	if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
		return image
	}
	return "docker.io/" + image
}

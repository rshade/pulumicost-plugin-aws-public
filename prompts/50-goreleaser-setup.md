# Prompt: Configure GoReleaser for region-specific binaries

You are OpenCode v0.15.3 using the GrokZeroFree model.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Add a `.goreleaser.yaml` that:

- Runs `tools/generate-pricing` before building.
- Builds **one binary per region** with different build tags and names:
  - `pulumicost-plugin-aws-public-us-east-1`
  - `pulumicost-plugin-aws-public-us-west-2`
  - `pulumicost-plugin-aws-public-eu-west-1`
- Produces OS/arch-specific archives suitable for publishing GitHub Releases.

## 1. Regions and build tags

For now, support three regions:

- `us-east-1` → `region_use1`
- `us-west-2` → `region_usw2`
- `eu-west-1` → `region_euw1`

Make sure these match the `//go:build` tags defined in `internal/pricing/embed_*.go`.

## 2. Create `.goreleaser.yaml`

At repo root, create `.goreleaser.yaml` with content similar to:

```yaml
project_name: pulumicost-plugin-aws-public

before:
  hooks:
    - go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy

builds:
  - id: aws-public-us-east-1
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-us-east-1
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    tags: [region_use1]

  - id: aws-public-us-west-2
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-us-west-2
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    tags: [region_usw2]

  - id: aws-public-eu-west-1
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-eu-west-1
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    tags: [region_euw1]

archives:
  - id: default
    format: tar.gz
    builds:
      - aws-public-us-east-1
      - aws-public-us-west-2
      - aws-public-eu-west-1
    name_template: "{{ .ProjectName }}-{{ .BuildID }}-{{ .Os }}-{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
```

Notes:

- `before.hooks` currently uses `--dummy` for reproducible builds in environments without AWS access.
  - Later, you can remove `--dummy` and implement a real fetch from AWS pricing APIs.
- `archives.name_template` ensures region is encoded in `BuildID`.

## 3. GoReleaser usage notes

Add a short `RELEASING.md` or section in `README.md`:

- Explain how to run:

  ```bash
  goreleaser release --clean
  ```

  or for testing:

  ```bash
  goreleaser build --snapshot --clean
  ```

- Mention that this will produce binaries like:

  - `pulumicost-plugin-aws-public-us-east-1-<os>-<arch>.tar.gz`
  - `pulumicost-plugin-aws-public-us-west-2-<os>-<arch>.tar.gz`
  - etc.

## 4. Acceptance criteria

- `goreleaser build --snapshot --clean` runs successfully in this repo (assuming GoReleaser is installed).
- Generated artifacts include one binary per region with the correct names.
- The generated binaries can be unpacked and `go version` confirms they run.

Implement `.goreleaser.yaml` and the basic release notes now.

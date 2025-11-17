# Prompt: Update README and basic docs for `pulumicost-plugin-aws-public`

You are OpenCode v0.15.3 using the GrokZeroFree model.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Update or create a `README.md` (and optionally `RELEASING.md`) that explains:

- What this plugin does.
- How it’s built and distributed (region-specific binaries, GoReleaser).
- How PulumiCost core is expected to call it.
- The error protocol (`PluginResponse`, `UNSUPPORTED_REGION`, etc.).

## 1. README structure

Ensure `README.md` contains at least:

### Title & summary

- Title: `pulumicost-plugin-aws-public`
- Short description: e.g.

  > A fallback PulumiCost plugin that estimates AWS resource costs from public on-demand pricing when no billing data is available.

### Features

- Estimates monthly/hourly cost for:
  - EC2 instances (on-demand, Linux/Shared)
  - EBS volumes
- Uses **public AWS pricing** embedded into the binary (no AWS credentials required at runtime).
- Region-specific binaries:
  - `pulumicost-plugin-aws-public-us-east-1`
  - `pulumicost-plugin-aws-public-us-west-2`
  - `pulumicost-plugin-aws-public-eu-west-1`
- Safe to use in CI as a “best effort” cost signal.

### How it works

Explain at a high level:

- At release time:
  - `tools/generate-pricing` fetches AWS public pricing.
  - It trims the data down to a small JSON per region (`data/aws_pricing_<region>.json`).
  - GoReleaser builds one binary per region with that JSON embedded via `//go:embed`.
- At runtime:
  - PulumiCost core runs the appropriate region binary (e.g. based on stack’s AWS region).
  - The plugin reads a `StackInput` JSON from stdin.
  - It calculates cost estimates using the embedded pricing.
  - It writes a `PluginResponse` JSON to stdout.

### JSON protocol

Document the envelope:

```jsonc
{
  "version": 1,
  "status": "ok",
  "result": {
    "resources": [
      {
        "urn": "...",
        "service": "ec2",
        "resourceType": "aws:ec2/instance:Instance",
        "region": "us-east-1",
        "monthlyCost": 12.34,
        "hourlyCost": 0.0169,
        "currency": "USD",
        "confidence": "high",
        "lineItems": [
          {
            "description": "On-demand Linux/Shared instance",
            "unit": "Hrs",
            "quantity": 730,
            "rate": 0.0169,
            "cost": 12.337
          }
        ],
        "assumptions": [
          {
            "key": "hours_per_month",
            "description": "Assumed 730 hours per month (24x7)",
            "value": "730"
          }
        ]
      }
    ],
    "totalMonthly": 12.34,
    "totalHourly": 0.0169,
    "currency": "USD"
  },
  "warnings": []
}
```

And the error form, especially `UNSUPPORTED_REGION`:

```jsonc
{
  "version": 1,
  "status": "error",
  "error": {
    "code": "UNSUPPORTED_REGION",
    "message": "This plugin is compiled for us-east-1 but resources are in us-west-2.",
    "meta": {
      "pluginRegion": "us-east-1",
      "requiredRegion": "us-west-2"
    }
  }
}
```

Explain that:

- Core can interpret this error and decide to download/run the correct region binary.

### Usage (standalone)

Provide a small example:

```bash
cat <<EOF > stack.json
{
  "resources": [
    {
      "urn": "urn:pulumi:dev::my-stack::aws:ec2/instance:Instance::web-1",
      "provider": "aws",
      "type": "aws:ec2/instance:Instance",
      "name": "web-1",
      "region": "us-east-1",
      "properties": {
        "instanceType": "t3.micro"
      }
    }
  ]
}
EOF

pulumicost-plugin-aws-public-us-east-1 < stack.json
```

### Building & releasing

Summarize GoReleaser usage:

- `goreleaser build --snapshot --clean`
- Mention that releases produce tarballs per region, OS, and arch.

## 2. RELEASING.md (optional but nice)

Create `RELEASING.md` with a concise check-list:

- Ensure `tools/generate-pricing` is up to date and working.
- Run tests: `go test ./...`
- Run snapshot build: `goreleaser build --snapshot --clean`
- Tag and release: `goreleaser release --clean`

## Acceptance criteria

- `README.md` clearly explains:
  - What the plugin does.
  - JSON protocol.
  - Region-specific binary naming.
  - High-level flow with PulumiCost core.
- Optional `RELEASING.md` exists with a small release checklist.

Update docs now.

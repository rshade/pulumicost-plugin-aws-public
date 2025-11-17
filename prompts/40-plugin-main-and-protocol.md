# Prompt: Implement plugin main entrypoint and JSON envelope protocol

You are OpenCode v0.15.3 using the GrokZeroFree model.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Update `cmd/pulumicost-plugin-aws-public/main.go` to:

- Read a `StackInput` JSON from stdin.
- Instantiate:
  - `config.Config` (using `config.Default()`).
  - `pricing.Client` (using embedded JSON).
  - `plugin.AWSEstimator`.
  - `plugin.Plugin`.
- Call the estimator.
- Wrap the result in a `PluginResponse` envelope.
- Handle errors and exit codes correctly.

## 1. Input format

The plugin should expect stdin to contain a JSON object matching `plugin.StackInput`:

```jsonc
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
```

Parsing errors should result in a `PluginResponse` with:

- `status: "error"`
- `error.code = "INVALID_INPUT"`

## 2. Wiring components

In `main.go`:

1. Read all stdin into a buffer.
2. Attempt to unmarshal into `plugin.StackInput`.
3. Construct config + pricing + estimator:

   ```go
   cfg := config.Default()
   pricingClient, err := pricing.NewClient()
   if err != nil {
       // wrap in PluginError with code "PRICING_INIT_FAILED"
   }

   est := plugin.NewAWSEstimator(cfg, pricingClient)
   p := plugin.NewPlugin(est)
   ```

4. Call:

   ```go
   result, warnings, perr := est.Estimate(context.Background(), &input)
   ```

5. Build `PluginResponse`:

   - If `perr != nil`:
     - `Status = "error"`
     - `Error = perr`
     - `Result = nil`
     - Exit code: `1`
   - Else:
     - `Status = "ok"`
     - `Result = result`
     - `Warnings = warnings`
     - Exit code: `0`

6. Marshal `PluginResponse` to stdout as pretty-compact JSON (default `json.Encoder` is fine).

7. On marshal/write failure, print a minimal message to stderr and exit `1`.

## 3. UNSUPPORTED_REGION behavior

The estimator already returns a `PluginError` with:

```go
Code: "UNSUPPORTED_REGION",
Message: "...",
Meta: {
  "pluginRegion": "...",
  "requiredRegion": "..."
}
```

Ensure:

- `main.go` does **not** special-case this error.
- It simply wraps it into the `PluginResponse` and exits with code `1`.
- This allows PulumiCost core to detect the error code and meta, and decide to fetch/run another region-specific binary.

## 4. Logging

- Do **not** log verbose data to stdout; stdout is strictly the JSON protocol.
- If needed, log debug info to stderr with a prefix like `[pulumicost-plugin-aws-public]`.
- For now, keep output minimal: only write the JSON envelope on stdout.

## 5. Smoke test

Add a simple test or developer note to verify behavior.

Optional: create an example JSON file under `testdata/input_ec2_ebs.json` and a small Go test or script that:

- Builds the plugin with a dummy pricing file embedded.
- Runs the binary with that input via `os/exec`.
- Parses the output and checks that:
  - `status = "ok"`
  - `result.totalMonthly > 0`.

This doesn’t have to be a full integration test yet, but set up the pattern.

## Acceptance criteria

- `go build ./cmd/pulumicost-plugin-aws-public` succeeds with default (no tags) build.
- Running the binary with invalid JSON prints a `PluginResponse` with `status="error", code="INVALID_INPUT"`.
- Running the binary with a valid stack input and a dummy pricing file can produce an `ok` response.

Implement these changes now.

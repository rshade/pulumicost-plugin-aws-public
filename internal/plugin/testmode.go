package plugin

import (
	"os"

	"github.com/rs/zerolog"
)

// testModeEnvVar is the environment variable name for enabling test mode.
const testModeEnvVar = "PULUMICOST_TEST_MODE"

// IsTestMode returns true if test mode is enabled via environment variable.
// Only the exact string "true" enables test mode (strict matching per FR-011).
func IsTestMode() bool {
	return os.Getenv(testModeEnvVar) == "true"
}

// ValidateTestModeEnv checks if PULUMICOST_TEST_MODE has an invalid value
// and logs a warning if so. Valid values are "true", "false", or unset.
// Invalid values (e.g., "1", "yes", "maybe") are treated as disabled with warning.
func ValidateTestModeEnv(logger zerolog.Logger) {
	val := os.Getenv(testModeEnvVar)
	if val != "" && val != "true" && val != "false" {
		logger.Warn().
			Str("env_var", testModeEnvVar).
			Str("value", val).
			Msg("Invalid PULUMICOST_TEST_MODE value; treating as disabled")
	}
}

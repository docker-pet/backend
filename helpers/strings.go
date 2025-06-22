package helpers

import "strings"

func EscapeEnv(input string) string {
    return strings.ReplaceAll(input, "$", "$$")
}
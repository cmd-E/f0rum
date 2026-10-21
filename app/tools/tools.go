package tools

import "strings"

func GetID(path string) string  { parts := strings.Split(path, "/"); return parts[len(parts)-1] }
func IsEmpty(value string) bool { return strings.TrimSpace(value) == "" }
